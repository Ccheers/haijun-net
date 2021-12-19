package haijun_net

import (
	"fmt"
	"net"
	"os"

	"github.com/Ccheers/haijun-net/internal/socket"
	"golang.org/x/sys/unix"
)

type Listener = net.Listener

type HjListener struct {
	listenFd int
	tcpAddr  *net.TCPAddr
}

func NewHjListener(addr string) (*HjListener, error) {

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

	return &HjListener{
		listenFd: listenFd,
		tcpAddr:  tcpAddr,
	}, nil
}

func (h *HjListener) Accept() (net.Conn, error) {
	nfd, sa, err := unix.Accept(h.listenFd)
	err = unix.SetNonblock(nfd, true)
	if err != nil {
		fmt.Println("block err", err)
	}

	netAddr := socket.SockaddrToTCPOrUnixAddr(sa)
	if err = os.NewSyscallError("setsockopt", unix.SetsockoptInt(nfd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, 5)); err != nil {
		return nil, err
	}

	conn := NewHjConn(nfd, h.Addr(), netAddr)
	return conn, err
}

func (h *HjListener) Close() error {
	return os.NewSyscallError("unix close", unix.Close(h.listenFd))
}

func (h *HjListener) Addr() net.Addr {
	return h.tcpAddr
}
