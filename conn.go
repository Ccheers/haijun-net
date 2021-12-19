package haijun_net

import (
	"net"
	"sync"
	"time"

	"github.com/Ccheers/haijun-net/internal/io"
	"github.com/Ccheers/haijun-net/internal/poller"
	"golang.org/x/sys/unix"
)

var connPoller poller.Poller
var connOnce sync.Once

type HjConn struct {
	fd              int
	localAddr       net.Addr
	remoteAddr      net.Addr
	readDeadline    time.Time
	readDeadlineMu  sync.RWMutex
	writeDeadline   time.Time
	writeDeadlineMu sync.RWMutex
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
	h.readDeadlineMu.Lock()
	h.readDeadline = t
	h.readDeadlineMu.Unlock()
	return nil
}

func (h *HjConn) SetWriteDeadline(t time.Time) error {
	h.writeDeadlineMu.Lock()
	h.writeDeadline = t
	h.writeDeadlineMu.Unlock()
	return nil
}

func initConnPoller() {
	var err error
	connPoller, err = poller.NewPoller()
	if err != nil {
		panic(err)
	}
}
