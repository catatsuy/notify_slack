package throttle

import (
	"bufio"
	"io"
)

type ThrottleWriter struct {
	sc     *bufio.Scanner
	writer *bufio.Writer
	exitC  chan struct{}
}

func NewWriter(input io.Reader, output io.Writer) *ThrottleWriter {
	sc := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)

	tw := &ThrottleWriter{
		sc:     sc,
		writer: writer,
		exitC:  make(chan struct{}, 0),
	}

	return tw
}

func (tw *ThrottleWriter) Run() {
	for tw.sc.Scan() {
		if _, err := tw.writer.Write(tw.sc.Bytes()); err != nil {
			panic(err)
		}
		if err := tw.writer.WriteByte('\n'); err != nil {
			panic(err)
		}
	}
	tw.exitC <- struct{}{}
}

func (tw *ThrottleWriter) Exit() <-chan struct{} {
	return tw.exitC
}

func (tw *ThrottleWriter) Flush() {
	tw.writer.Flush()
}
