package haijun_net

import (
	"runtime"
	"sync"

	"github.com/Ccheers/haijun-net/internal/io"
	"github.com/Ccheers/haijun-net/internal/poller"
	"golang.org/x/sys/unix"
)

const (
	// MaxBytesToWritePerLoop is the maximum amount of bytes to be sent in one system call.
	MaxBytesToWritePerLoop = 64 * 1024
	// MaxIovSize is IOV_MAX.
	MaxIovSize = 1024
)

type connManager struct {
	connMap sync.Map
	poller  poller.Poller
}

func newConnManager(poller poller.Poller) *connManager {
	return &connManager{poller: poller}
}

func (m *connManager) getConn(fd int) (*HjConn, bool) {
	res, ok := m.connMap.Load(fd)
	if ok {
		return res.(*HjConn), true
	}
	return nil, false
}

func (m *connManager) setConn(fd int, conn *HjConn) {
	_, ok := m.getConn(fd)
	if ok {
		return
	}
	m.connMap.Store(fd, conn)
}

func (m *connManager) unsetConn(fd int) {
	conn, ok := m.getConn(fd)
	if ok {
		m.connMap.Delete(fd)
		m.poller.Remove(fd)
		conn.Close()
	}
}

func (m *connManager) RegisterConn(conn *HjConn) (err error) {
	_, ok := m.getConn(conn.fd)
	if ok {
		return
	}
	err = m.poller.Register(conn.fd, poller.PollModeRead)
	if err != nil {
		return
	}
	m.setConn(conn.fd, conn)
	return
}

func (m *connManager) ModWrite(conn *HjConn) (err error) {
	return m.poller.ModReadWrite(conn.fd)
}

func (m *connManager) ModRead(conn *HjConn) (err error) {
	return m.poller.ModRead(conn.fd)
}

func (m *connManager) Run() {
	//runtime.LockOSThread()
	for {
		events, err := m.poller.Wait()
		if err != nil {
			runtime.Gosched()
			continue
		}
		for _, event := range events {
			conn, ok := m.getConn(int(event.Fd))
			if !ok {
				m.poller.Remove(int(event.Fd))
				continue
			}
			// Don't change the ordering of processing EPOLLOUT | EPOLLRDHUP / EPOLLIN unless you're 100%
			// sure what you're doing!
			// Re-ordering can easily introduce bugs and bad side-effects, as I found out painfully in the past.

			// We should always check for the EPOLLOUT event first, as we must try to send the leftover data back to
			// the peer when any error occurs on a connection.
			//
			// Either an EPOLLOUT or EPOLLERR event may be fired when a connection is refused.
			// In either case write() should take care of it properly:
			// 1) writing data back,
			// 2) closing the connection.
			if event.Events&poller.OutEvents != 0 && !conn.writeBuffer.IsEmpty() {
				if _, err := io.Writev(int(event.Fd), conn.writeBuffer.Peek(MaxBytesToWritePerLoop)); err != nil {
					switch err {
					case nil, unix.EAGAIN:
					default:
						m.unsetConn(int(event.Fd))
					}
				}
				if conn.writeBuffer.IsEmpty() {
					err := m.poller.ModRead(conn.fd)
					if err != nil {
						m.unsetConn(conn.fd)
						continue
					}
				}
			}
			// If there is pending data in outbound buffer, then we should omit this readable event
			// and prioritize the writable events to achieve a higher performance.
			//
			// Note that the peer may send massive amounts of data to server by write() under blocking mode,
			// resulting in that it won't receive any responses before the server reads all data from the peer,
			// in which case if the server socket send buffer is full, we need to let it go and continue reading
			// the data to prevent blocking forever.
			// 读事件处理
			if event.Events&poller.InEvents != 0 && (event.Events&poller.OutEvents == 0 || conn.writeBuffer.IsEmpty()) {
				if _, err := conn.readBuffer.CopyFromSocket(int(event.Fd)); err != nil {
					switch err {
					case nil, unix.EAGAIN:
					default:
						m.unsetConn(int(event.Fd))
					}
				}

				select {
				case conn.waitRead <- struct{}{}:
				default:
				}
			}
		}
	}
}
