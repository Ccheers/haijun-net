package poller

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

// IOEvent is the integer type of I/O eventList on Linux.
type IOEvent = uint32

const (
	// InitPollEventsCap represents the initial capacity of Poller event-list.
	InitPollEventsCap = 128
	// MaxPollEventsCap is the maximum limitation of eventList that the Poller can process.
	MaxPollEventsCap = 1024
	// MinPollEventsCap is the minimum limitation of eventList that the Poller can process.
	MinPollEventsCap = 32
)

const (
	// ErrEvents represents exceptional eventList that are not read/write, like socket being closed,
	// reading/writing from/to a closed socket, etc.
	ErrEvents IOEvent = unix.EPOLLERR | unix.EPOLLHUP | unix.EPOLLRDHUP
	// OutEvents combines EPOLLOUT event and some exceptional eventList.
	OutEvents IOEvent = ErrEvents | unix.EPOLLOUT
	// InEvents combines EPOLLIN/EPOLLPRI eventList and some exceptional eventList.
	InEvents IOEvent = ErrEvents | unix.EPOLLIN | unix.EPOLLPRI
)

type Poller interface {
	// Register add fd to the polling set.
	// fd must be greater than 0.
	Register(fd int, mode PollMode) error
	// Remove removes fd from the polling set.
	// fd must be greater than 0.
	Remove(fd int) error
	// Wait waits for eventList.
	// fd must be greater than 0.
	Wait() ([]unix.EpollEvent, error)

	ModRead(fd int) error
	ModReadWrite(fd int) error
}

type PollMode uint32

const (
	PollModeRead PollMode = 1 << iota
	PollModeWrite
)

const (
	readEvents      = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents     = unix.EPOLLOUT
	readWriteEvents = readEvents | writeEvents
)

var (
	errFdIsZero     = errors.New("fd must be greeter than 0")
	errFdRegistered = errors.New("fd has being registered")
	errFdUnRegister = errors.New("fd not registered")
	errModeIsNone   = errors.New("mode must be greeter than 0")
)

type pollerImpl struct {
	pollFD    int // epoll fd
	eventList *eventList

	fdModes sync.Map
}

func (p *pollerImpl) getFdMode(fd int) (*PollMode, error) {
	if fd == 0 {
		return nil, errFdIsZero
	}
	mode, ok := p.fdModes.Load(fd)
	if !ok {
		return nil, errFdUnRegister
	}
	return mode.(*PollMode), nil
}

func (p *pollerImpl) setFdMode(fd int, mode PollMode) error {
	if fd == 0 {
		return errFdIsZero
	}
	if mode == 0 {
		return errModeIsNone
	}

	m, _ := p.getFdMode(fd)
	if m != nil {
		atomic.StoreUint32((*uint32)(m), uint32(mode))
	}

	p.fdModes.Store(fd, &mode)
	return nil
}

func NewPoller() (p Poller, err error) {
	impl := new(pollerImpl)
	if impl.pollFD, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}
	impl.eventList = newEventList(InitPollEventsCap)
	return impl, nil
}

// Register adds fd to the polling set. by default will register the fd with READ
func (p *pollerImpl) Register(fd int, mode PollMode) error {
	if fd <= 0 {
		return errFdIsZero
	}

	if m, _ := p.getFdMode(fd); m != nil {
		return errFdRegistered
	}

	var events = PollModeRead
	if mode&PollModeWrite > 0 {
		events = events | writeEvents
	}
	if events == 0 {
		return errModeIsNone
	}

	err := os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_ADD, fd, &unix.EpollEvent{Fd: int32(fd), Events: uint32(events)}))
	if err != nil {
		return err
	}

	return p.setFdMode(fd, mode)
}

// ModRead renews the given file-descriptor with readable event in the poller.
func (p *pollerImpl) ModRead(fd int) error {
	if fd <= 0 {
		return errFdIsZero
	}
	mode, err := p.getFdMode(fd)
	if err != nil {
		return err
	}

	ok := atomic.CompareAndSwapUint32((*uint32)(mode), readWriteEvents, readEvents)
	if !ok {
		// 已经是这个状态，无需变更
		if *(*uint32)(mode) == readEvents {
			return nil
		}
		// 强制更新成读事件
		atomic.StoreUint32((*uint32)(mode), readEvents)
	}

	return os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: uint32(*mode)}))
}

// ModReadWrite renews the given file-descriptor with readable and writable events in the poller.
func (p *pollerImpl) ModReadWrite(fd int) error {

	if fd <= 0 {
		return errFdIsZero
	}
	mode, err := p.getFdMode(fd)
	if err != nil {
		return err
	}

	ok := atomic.CompareAndSwapUint32((*uint32)(mode), readEvents, readWriteEvents)
	if !ok {
		// 已经是这个状态，无需变更
		if *(*uint32)(mode) == readWriteEvents {
			return nil
		}
		// 强制更新成读写事件
		atomic.StoreUint32((*uint32)(mode), readWriteEvents)
	}

	return os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: uint32(*mode)}))
}

func (p *pollerImpl) Remove(fd int) error {
	if fd <= 0 {
		return errFdIsZero
	}
	if m, _ := p.getFdMode(fd); m == nil {
		return errFdUnRegister
	}
	p.fdModes.Delete(fd)
	return os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_DEL, fd, nil))
}

func (p *pollerImpl) Wait() ([]unix.EpollEvent, error) {
	n, err := unix.EpollWait(p.pollFD, p.eventList.events, 5)
	if n == 0 || (n < 0 && err == unix.EINTR) {
		return nil, nil
	} else if err != nil {
		return nil, os.NewSyscallError("epoll_wait", err)
	}
	if n == p.eventList.size {
		p.eventList.expand()
	} else if n < p.eventList.size>>1 {
		p.eventList.shrink()
	}
	return p.eventList.events[:n], nil
}

type eventList struct {
	size   int
	events []unix.EpollEvent
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.EpollEvent, size)}
}

// expand 扩展监听列表
func (el *eventList) expand() {
	if newSize := el.size << 1; newSize <= MaxPollEventsCap {
		el.size = newSize
		el.events = make([]unix.EpollEvent, newSize)
	}
}

// shrink 收缩监听列表
func (el *eventList) shrink() {
	if newSize := el.size >> 1; newSize >= MinPollEventsCap {
		el.size = newSize
		el.events = make([]unix.EpollEvent, newSize)
	}
}
