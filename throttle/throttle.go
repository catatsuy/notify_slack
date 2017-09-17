package throttle

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"time"
)

type ThrottleWriter struct {
	sc     *bufio.Scanner
	writer *bytes.Buffer
	exitC  chan struct{}
}

func NewWriter(input io.Reader, output *bytes.Buffer) *ThrottleWriter {
	sc := bufio.NewScanner(input)

	tw := &ThrottleWriter{
		sc:     sc,
		writer: output,
		exitC:  make(chan struct{}, 0),
	}

	return tw
}

func (tw *ThrottleWriter) Setup() {
	go func() {
		for tw.sc.Scan() {
			_, err := tw.writer.Write(tw.sc.Bytes())
			if err != nil {
				panic(err)
			}

			err = tw.writer.WriteByte('\n')
			if err != nil {
				panic(err)
			}
		}
		tw.exitC <- struct{}{}
	}()
}

func (tw *ThrottleWriter) Run(ctx context.Context, interval <-chan time.Time, flushCallback func(ctx context.Context, output string) error, doneCallback func(ctx context.Context, output string) error) <-chan struct{} {
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

	return tw.exitC
}

func (tw *ThrottleWriter) flush() string {
	defer tw.writer.Reset()

	return tw.writer.String()
}
