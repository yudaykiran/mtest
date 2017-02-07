// The package gatedwriter is aptly prefixed with gated, as
// it buffers the data before being actually flushed / written.
package gatedwriter

import (
	"io"
	"sync"
)

// Writer is an io.Writer implementation that buffers all of its
// data into an internal buffer until it is told to let data through.
type Writer struct {
	// The actual Writer from io pkg, which will write the bytes.
	// However, this is managed by our own Writer implementation.
	Writer io.Writer

	// bug is a placeholder to store/buffer the data
	// before being flushed to the Writer
	buf [][]byte

	// flush is a flag that enables Write when enabled
	flush bool

	// lock is the Mutex that synchronises flush, & writing
	lock sync.RWMutex
}

// Flush tells the Writer to flush any buffered data and to stop
// buffering.
//
// NOTE:
//    This is meant to be called only once. Further buffering
//    needs to be done by creating a new instance of Writer.
func (w *Writer) Flush() {
	w.lock.Lock()
	w.flush = true
	w.lock.Unlock()

	for _, p := range w.buf {
		w.Write(p)
	}
	w.buf = nil
}

// Write is the custom implementation that will write if
// flush is enabled or will buffer otherwise.
func (w *Writer) Write(p []byte) (n int, err error) {
	w.lock.RLock()
	defer w.lock.RUnlock()

	if w.flush {
		return w.Writer.Write(p)
	}

	p2 := make([]byte, len(p))
	copy(p2, p)
	w.buf = append(w.buf, p2)
	return len(p), nil
}
