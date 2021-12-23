package haijun_net

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ccheers/haijun-net/internal/io"
	"github.com/Ccheers/haijun-net/internal/poller"
	"golang.org/x/sys/unix"
)

var connOnce sync.Once

type connManager struct {
	connMap sync.Map
	poller  poller.Poller
}

func newConnManager(poller poller.Poller) *connManager {
	return &connManager{poller: poller}
}

type HjConn struct {
	fd            int
	localAddr     net.Addr
	remoteAddr    net.Addr
	readDeadline  atomic.Value
	writeDeadline atomic.Value
}

func NewHjConn(fd int, localAddr, remoteAddr net.Addr) net.Conn {
	connOnce.Do(initConnPoller)
	return &HjConn{
		fd:         fd,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

func (h *HjConn) Read(b []byte) (n int, err error) {
	return io.Read(h.fd, b)
}

func (h *HjConn) Write(b []byte) (n int, err error) {
	return io.Write(h.fd, b)
}

func (h *HjConn) Close() error {
	return unix.Close(h.fd)
}

func (h *HjConn) LocalAddr() net.Addr {
	return h.localAddr
}

func (h *HjConn) RemoteAddr() net.Addr {
	return h.remoteAddr
}

func (h *HjConn) SetDeadline(t time.Time) error {
	err := h.SetWriteDeadline(t)
	if err != nil {
		return err
	}
	return h.SetReadDeadline(t)
}

func (h *HjConn) SetReadDeadline(t time.Time) error {
	h.readDeadline.Store(t)
	return nil
}

func (h *HjConn) SetWriteDeadline(t time.Time) error {
	h.writeDeadline.Store(t)
	return nil
}

func initConnPoller() {
	connPoller, err := poller.NewPoller()
	if err != nil {
		panic(err)
	}
	newConnManager(connPoller)
}
