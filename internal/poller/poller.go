package poller

import (
	"errors"
	"os"

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
	// MaxAsyncTasksAtOneTime is the maximum amount of asynchronous tasks that the event-loop will process at one time.
	MaxAsyncTasksAtOneTime = 256
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
	Wait() (events []unix.EpollEvent, err error)
}

type PollMode int

const (
	PollModeRead PollMode = 1 << iota
	PollModeWrite
)

const (
	readEvents  = unix.EPOLLPRI | unix.EPOLLIN
	writeEvents = unix.EPOLLOUT
)

var (
	errFdIsZero   = errors.New("fd must be greeter than 0")
	errModeIsNone = errors.New("mode must be greeter than 0")
)

type pollerImpl struct {
	pollFD    int // epoll fd
	eventList *eventList
}

func NewPoller() (p Poller, err error) {
	impl := new(pollerImpl)
	if impl.pollFD, err = unix.EpollCreate1(unix.EPOLL_CLOEXEC); err != nil {
		return nil, os.NewSyscallError("epoll_create1", err)
	}
	impl.eventList = newEventList(InitPollEventsCap)
	return impl, nil
}

func (p *pollerImpl) Register(fd int, mode PollMode) error {
	if fd <= 0 {
		return errFdIsZero
	}
	var events uint32 = 0
	if mode&PollModeWrite > 0 {
		events = events | writeEvents
	}
	if mode&PollModeRead > 0 {
		events = events | readEvents
	}
	if events == 0 {
		return errModeIsNone
	}

	return os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{Fd: int32(fd), Events: events}))
}

func (p *pollerImpl) Remove(fd int) error {
	if fd <= 0 {
		return errFdIsZero
	}
	return os.NewSyscallError("epoll_ctl mod", unix.EpollCtl(p.pollFD, unix.EPOLL_CTL_DEL, fd, nil))
}

func (p *pollerImpl) Wait() (events []unix.EpollEvent, err error) {
	n, err := unix.EpollWait(p.pollFD, p.eventList.events, 5)
	if n == 0 || (n < 0 && err == unix.EINTR) {
		return nil, nil
	} else if err != nil {
		return nil, os.NewSyscallError("epoll_wait", err)
	}
	return events, nil
}

type eventList struct {
	size   int
	events []unix.EpollEvent
}

func newEventList(size int) *eventList {
	return &eventList{size, make([]unix.EpollEvent, size)}
}

func (el *eventList) expand() {
	if newSize := el.size << 1; newSize <= MaxPollEventsCap {
		el.size = newSize
		el.events = make([]unix.EpollEvent, newSize)
	}
}

func (el *eventList) shrink() {
	if newSize := el.size >> 1; newSize >= MinPollEventsCap {
		el.size = newSize
		el.events = make([]unix.EpollEvent, newSize)
	}
}
