package throttle

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

type Exec struct {
	reader *bufio.Reader
	rc     io.ReadCloser
	pr     *io.PipeReader
	writer *bytes.Buffer
	exitC  chan struct{}
	mu     sync.Mutex
}

func NewExec(input io.Reader) *Exec {
	reader := bufio.NewReader(input)
	ex := &Exec{
		reader: reader,
		writer: new(bytes.Buffer),
		exitC:  make(chan struct{}),
		mu:     sync.Mutex{},
	}
	// capture closers if present
	if rc, ok := input.(io.ReadCloser); ok {
		ex.rc = rc
	}
	if pr, ok := input.(*io.PipeReader); ok {
		ex.pr = pr
	}
	return ex
}

func (ex *Exec) write(p []byte) (n int, err error) {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	return ex.writer.Write(p)
}

func (ex *Exec) writeByte(p byte) (err error) {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	return ex.writer.WriteByte(p)
}

func (ex *Exec) stringAndReset() string {
	ex.mu.Lock()
	defer func() {
		ex.writer.Reset()
		ex.mu.Unlock()
	}()

	return ex.writer.String()
}

func (ex *Exec) Start(ctx context.Context, interval <-chan time.Time, flushCallback func(ctx context.Context, output string) error, doneCallback func(ctx context.Context, output string) error) {
	go func() {
		for {
			line, _, err := ex.reader.ReadLine()
			if err != nil {
				if errors.Is(err, io.EOF) ||
					errors.Is(err, io.ErrClosedPipe) ||
					errors.Is(err, context.Canceled) {
					break
				}

				panic(err)
			}

			ex.write(line)
			ex.writeByte('\n')
		}
		// if notify_slack receives EOF, this function will exit.
		close(ex.exitC)
	}()

L:
	for {
		select {
		case <-interval:
			flushCallback(ctx, ex.flush())
		case <-ctx.Done():
			ex.cancelReader(ctx.Err())
			doneCallback(ctx, ex.flush())
			break L
		case <-ex.Wait():
			doneCallback(ctx, ex.flush())
			break L
		}
	}
}

func (ex *Exec) Wait() <-chan struct{} {
	return ex.exitC
}

func (ex *Exec) flush() string {
	return ex.stringAndReset()
}

// cancelReader closes the underlying reader, if possible,
// so a blocked ReadLine can return on context cancellation.
func (ex *Exec) cancelReader(err error) {
	if ex.pr != nil {
		ex.pr.CloseWithError(err)
		return
	}

	if ex.rc != nil {
		ex.rc.Close()
	}
}
