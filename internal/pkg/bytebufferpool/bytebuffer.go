package bytebufferpool

import "github.com/valyala/bytebufferpool"

type ByteBuffer = bytebufferpool.ByteBuffer

var (
	// Get returns an empty byte buffer from the pool.
	Get = bytebufferpool.Get
	// Put returns byte buffer to the pool.
	Put = func(b *bytebufferpool.ByteBuffer) {
		if b != nil {
			bytebufferpool.Put(b)
		}
	}
)
