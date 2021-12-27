package haijun_net

import (
	"log"
	"net"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/Ccheers/haijun-net/internal/poller"
	"github.com/Ccheers/haijun-net/internal/socket"
	"golang.org/x/sys/unix"
)

type Listener = net.Listener

type HjListener struct {
	listenFd int
	tcpAddr  *net.TCPAddr

	poller     poller.Poller
	hasNewConn uint32
	wakeChan   chan struct{}
}

func NewHjListener(addr string) (Listener, error) {

	// 获取是tcp的listenFd
	listenFd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM|unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC, unix.IPPROTO_TCP)

	if err != nil {
		err = os.NewSyscallError("socket", err)
		return nil, err
	}
	if err := os.NewSyscallError("set sock opt int", unix.SetsockoptInt(listenFd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)); err != nil {
		return nil, err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, os.NewSyscallError("resove err", err)
	}
	ip := tcpAddr.IP.To4()
	sa := &unix.SockaddrInet4{
		Port: tcpAddr.Port,
		Addr: [4]byte{ip[0], ip[1], ip[2], ip[3]},
	}
	// 绑定的端口
	err = unix.Bind(listenFd, sa)
	if err != nil {
		return nil, os.NewSyscallError("socket bind err", err)
	}

	var n int
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}
	// 监听服务
	err = unix.Listen(listenFd, n)
	if err != nil {
		return nil, os.NewSyscallError("listen err", err)
	}
	p, err := poller.NewPoller()
	if err != nil {
		return nil, err
	}
	l := &HjListener{
		listenFd: listenFd,
		tcpAddr:  tcpAddr,
		poller:   p,
		wakeChan: make(chan struct{}, 1),
	}
	l.Run()
	return l, nil
}

func (h *HjListener) Run() {
	log.Println("register listenFd")
	err := h.poller.Register(h.listenFd, poller.PollModeRead)
	if err != nil {
		panic(err)
	}
	go func() {
		//defer func() {
		//	if err := recover(); err != nil {
		//		log.Println(errors.New(fmt.Sprintf("%v", err)))
		//	}
		//}()
		for {
			events, err := h.poller.Wait()
			if err != nil {
				panic(err)
			}
			if len(events) == 0 {
				runtime.Gosched()
				continue
			}
			for _, event := range events {
				if int(event.Fd) == h.listenFd {
					if atomic.CompareAndSwapUint32(&h.hasNewConn, 0, 1) {
						h.wakeChan <- struct{}{}
					}
				}
			}
		}
	}()
}

func (h *HjListener) Accept() (net.Conn, error) {
	var (
		nfd int
		sa  unix.Sockaddr
		err error
	)

	for {
		if atomic.LoadUint32(&h.hasNewConn) != 1 {
			<-h.wakeChan
		}
		nfd, sa, err = unix.Accept(h.listenFd)
		if err != nil && err != unix.EAGAIN {
			return nil, err
		}
		if nfd > 0 {
			log.Println("accept new conn", nfd)
			break
		}
		atomic.StoreUint32(&h.hasNewConn, 0)
	}
	err = unix.SetNonblock(nfd, true)
	if err != nil {
		return nil, os.NewSyscallError("block err", err)
	}

	netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if err = os.NewSyscallError("setsockopt", unix.SetsockoptInt(nfd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, 5)); err != nil {
		return nil, err
	}

	return NewHjConn(nfd, h.Addr(), netAddr)
}

func (h *HjListener) Close() error {
	return os.NewSyscallError("unix close", unix.Close(h.listenFd))
}

func (h *HjListener) Addr() net.Addr {
	return h.tcpAddr
}
