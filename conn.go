package haijun_net

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ccheers/haijun-net/internal/pkg/mixedbuffer"
	rbPool "github.com/Ccheers/haijun-net/internal/pkg/pool/ringbuffer"
	"github.com/Ccheers/haijun-net/internal/pkg/ringbuffer"
	"github.com/Ccheers/haijun-net/internal/poller"
	"golang.org/x/sys/unix"
)

var connOnce sync.Once
var manager *connManager

type HjConn struct {
	fd            int
	localAddr     net.Addr
	remoteAddr    net.Addr
	readDeadline  atomic.Value
	writeDeadline atomic.Value

	readBuffer  *ringbuffer.RingBuffer
	waitRead    chan struct{}
	writeBuffer *mixedbuffer.Buffer

	manager *connManager
}

func NewHjConn(fd int, localAddr, remoteAddr net.Addr) (net.Conn, error) {
	connOnce.Do(initConnPoller)
	conn := &HjConn{
		fd:            fd,
		localAddr:     localAddr,
		remoteAddr:    remoteAddr,
		readDeadline:  atomic.Value{},
		writeDeadline: atomic.Value{},
		readBuffer:    rbPool.GetWithSize(ringbuffer.MaxStreamBufferCap),
		waitRead:      make(chan struct{}, 1),
		writeBuffer:   mixedbuffer.New(ringbuffer.MaxStreamBufferCap),
		manager:       manager,
	}
	err := conn.manager.RegisterConn(conn)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return conn, nil
}

func (h *HjConn) Read(b []byte) (n int, err error) {
	if h.readBuffer.IsEmpty() {
		<-h.waitRead
	}
	return h.readBuffer.Read(b)
}

func (h *HjConn) Write(b []byte) (n int, err error) {
	err = h.manager.ModWrite(h)
	if err != nil {
		return
	}
	return h.writeBuffer.Write(b)
}

func (h *HjConn) Close() error {
	h.readBuffer.Reset()
	rbPool.Put(h.readBuffer)
	h.writeBuffer.Release()

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
	manager = newConnManager(connPoller)
	go func() {
		//defer func() {
		//	// recover
		//	err := recover()
		//	if err != nil {
		//		log.Println(errors.New(fmt.Sprintf("%v", err)))
		//	}
		//}()
		manager.Run()
	}()
}
