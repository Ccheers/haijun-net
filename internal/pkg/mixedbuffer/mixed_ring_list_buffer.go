package mixedbuffer

import (
	"github.com/Ccheers/haijun-net/internal/pkg/listbuffer"
	rbPool "github.com/Ccheers/haijun-net/internal/pkg/pool/ringbuffer"
	"github.com/Ccheers/haijun-net/internal/pkg/ringbuffer"
)

// Buffer combines ring-buffer and list-buffer.
// Ring-buffer is the top-priority buffer to store response data, gnet will only switch to
// list-buffer if the data size of ring-buffer reaches the maximum(MaxStackingBytes), list-buffer is more
// flexible and scalable, which helps the application reduce memory footprint.
type Buffer struct {
	maxStackingBytes int
	ringBuffer       *ringbuffer.RingBuffer
	listBuffer       listbuffer.ListBuffer
}

// New instantiates a mixedbuffer.Buffer and returns it.
func New(maxTopBufCap int) *Buffer {
	return &Buffer{maxStackingBytes: maxTopBufCap, ringBuffer: rbPool.GetWithSize(maxTopBufCap)}
}

// Peek returns n bytes as [][]byte, these bytes won't be discarded until Buffer.Discard() is called.
func (mb *Buffer) Peek(n int) [][]byte {
	head, tail := mb.ringBuffer.PeekAll()
	return mb.listBuffer.PeekBytesListWithBytes(n, head, tail)
}

// Discard discards n bytes in this buffer.
func (mb *Buffer) Discard(n int) {
	rbLen := mb.ringBuffer.Length()
	mb.ringBuffer.Discard(n)
	if n <= rbLen {
		return
	}
	n -= rbLen
	mb.listBuffer.DiscardBytes(n)
}

// Write appends data to this buffer.
func (mb *Buffer) Write(p []byte) (n int, err error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Length() >= mb.maxStackingBytes {
		mb.listBuffer.PushBytesBack(p)
		return len(p), nil
	}
	freeSize := mb.ringBuffer.Free()
	if len(p) > freeSize {
		n, err = mb.ringBuffer.Write(p[:freeSize])
		mb.listBuffer.PushBytesBack(p[n:])
		return
	}
	return mb.ringBuffer.Write(p)
}

// Writev appends multiple byte slices to this buffer.
func (mb *Buffer) Writev(bs [][]byte) (int, error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Length() >= mb.maxStackingBytes {
		var n int
		for _, b := range bs {
			mb.listBuffer.PushBytesBack(b)
			n += len(b)
		}
		return n, nil
	}
	var pos, sum int
	freeSize := mb.ringBuffer.Free()
	for i, b := range bs {
		pos = i
		sum += len(b)
		if len(b) > freeSize {
			n, _ := mb.ringBuffer.Write(b[:freeSize])
			mb.listBuffer.PushBytesBack(b[n:])
			break
		}
		n, _ := mb.ringBuffer.Write(b)
		freeSize -= n
	}
	for pos++; pos < len(bs); pos++ {
		sum += len(bs[pos])
		mb.listBuffer.PushBytesBack(bs[pos])
	}
	return sum, nil
}

// IsEmpty indicates whether this buffer is empty.
func (mb *Buffer) IsEmpty() bool {
	return mb.ringBuffer.IsEmpty() && mb.listBuffer.IsEmpty()
}

// Release frees all resource of this buffer.
func (mb *Buffer) Release() {
	mb.listBuffer.Reset()
	mb.ringBuffer.Reset()
	rbPool.Put(mb.ringBuffer)
}
