package throttle

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"sync"
	"time"
)

type ThrottleWriter struct {
	sc     *bufio.Scanner
	writer *bytes.Buffer
	exitC  chan struct{}
	mu     sync.Mutex
}

func NewWriter(input io.Reader) *ThrottleWriter {
	sc := bufio.NewScanner(input)

	tw := &ThrottleWriter{
		sc:     sc,
		writer: new(bytes.Buffer),
		exitC:  make(chan struct{}, 0),
		mu:     sync.Mutex{},
	}

	return tw
}

func (tw *ThrottleWriter) Write(p []byte) (n int, err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	return tw.writer.Write(p)
}

func (tw *ThrottleWriter) WriteByte(p byte) (err error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	return tw.writer.WriteByte(p)
}

func (tw *ThrottleWriter) stringAndReset() string {
	tw.mu.Lock()
	defer func() {
		tw.writer.Reset()
		tw.mu.Unlock()
	}()

	return tw.writer.String()
}

func (tw *ThrottleWriter) Start(ctx context.Context, interval <-chan time.Time, flushCallback func(ctx context.Context, output string) error, doneCallback func(ctx context.Context, output string) error) {
	go func() {
		for tw.sc.Scan() {
			_, err := tw.Write(tw.sc.Bytes())
			if err != nil {
				panic(err)
			}

			err = tw.WriteByte('\n')
			if err != nil {
				panic(err)
			}
		}
		tw.exitC <- struct{}{}
	}()

	go func() {
		for {
			select {
			case <-interval:
				flushCallback(ctx, tw.flush())
				break
			case <-ctx.Done():
				doneCallback(ctx, tw.flush())
				return
			}
		}
	}()
}

func (tw *ThrottleWriter) Wait() <-chan struct{} {
	return tw.exitC
}

func (tw *ThrottleWriter) flush() string {
	return tw.stringAndReset()
}
