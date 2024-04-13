package throttle

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"sync"
	"time"
)

type Exec struct {
	reader *bufio.Reader
	writer *bytes.Buffer
	exitC  chan struct{}
	mu     sync.Mutex
}

func NewExec(input io.Reader) *Exec {
	reader := bufio.NewReader(input)

	return &Exec{
		reader: reader,
		writer: new(bytes.Buffer),
		exitC:  make(chan struct{}),
		mu:     sync.Mutex{},
	}
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
				if err == io.EOF {
					break
				}
				panic(err)
			}
			_, err = ex.write(line)
			if err != nil {
				panic(err)
			}

			err = ex.writeByte('\n')
			if err != nil {
				panic(err)
			}
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
