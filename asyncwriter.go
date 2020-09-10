package log

import (
	"errors"
	"io"
	"sync/atomic"
)

// AsyncWriter 包装了一个 writer，让所有写入变成异步写。
type AsyncWriter struct {
	ch      chan []byte
	closing chan bool
	flushed chan bool
	done    chan bool
	writer  io.WriteCloser

	closed int32
}

var _ io.WriteCloser = new(AsyncWriter)

// NewAsyncWriter 创建一个异步 writer，使用 size 作为缓冲区的条数。
func NewAsyncWriter(writer io.WriteCloser, size int) *AsyncWriter {
	w := &AsyncWriter{
		ch:      make(chan []byte, size),
		closing: make(chan bool, 1),
		flushed: make(chan bool, 1),
		done:    make(chan bool),
		writer:  writer,
	}
	go w.flush()
	return w
}

var (
	errAsyncWriterClosed = errors.New("go-log: async writer is closed")
	errAsyncWriterFull   = errors.New("go-log: async writer is full")
)

// Write 写入 data 到异步队列里面，任何情况下这个函数不会阻塞。
// 如果缓冲区满了或者 w 已经被关闭，返回错误。
func (w *AsyncWriter) Write(data []byte) (written int, err error) {
	if len(data) == 0 {
		return
	}

	if w.isClosed() {
		err = errAsyncWriterClosed
		return
	}

	select {
	case w.ch <- data:
		written = len(data)
	default:
		// 已经 close 或者缓冲区撑爆了。
		err = errAsyncWriterFull
	}

	return
}

// Flush 用来刷新当前缓存的数据。
func (w *AsyncWriter) Flush() error {
	if w.isClosed() {
		return errAsyncWriterClosed
	}

	// 插入一个特殊数据，必须得写入才行。
	select {
	case w.ch <- nil:
	case <-w.done:
		return errAsyncWriterClosed
	}

	// 说明一定会刷新，无需再等了。
	if w.isClosed() {
		return nil
	}

	// 触发 flush 流程，等待至少一个特殊数据被消费。
	select {
	case <-w.flushed:
	case <-w.done:
	}

	return nil
}

// Close 关闭 w，释放内部的 writer，并且关闭刷数据的 goroutine。
// 这个函数会在所有数据写入之后再返回，缓冲区比较满的时候会花较长时间返回。
func (w *AsyncWriter) Close() error {
	if w.isClosed() {
		return nil
	}

	select {
	case w.closing <- true:
	case <-w.done:
		return nil
	}

	select {
	case <-w.done:
	}

	return nil
}

func (w *AsyncWriter) flush() {
	for {
		select {
		case data := <-w.ch:
			if len(data) == 0 {
				w.flushed <- true
				continue
			}

			w.writer.Write(data)

		case <-w.closing:
			atomic.StoreInt32(&w.closed, 1)

			// 清空缓存。
			for {
				select {
				case data := <-w.ch:
					if len(data) == 0 {
						continue
					}

					w.writer.Write(data)
				default:
					w.writer.Close()
					close(w.done)
					return
				}
			}
		}
	}
}

func (w *AsyncWriter) isClosed() bool {
	return atomic.LoadInt32(&w.closed) != 0
}
