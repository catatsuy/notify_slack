package throttle

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	runtime.GOMAXPROCS(1)

	input := new(bytes.Buffer)
	output := new(bytes.Buffer)

	ex := NewExec(input)

	ctx, cancel := context.WithCancel(context.Background())
	testC := make(chan time.Time, 0)
	count := 0
	fc := make(chan struct{}, 0)

	flushCallback := func(_ context.Context, s string) error {
		defer func() {
			fc <- struct{}{}
		}()

		count++

		output.WriteString(s)

		return nil
	}

	ex.Start(ctx, testC, flushCallback, flushCallback)

	testC <- time.Time{}
	<-fc

	if count != 1 {
		t.Error("the flushCallback function has not been called")
	}

	expected := "abcd\nefgh\n"

	input.WriteString(expected)
	time.Sleep(time.Millisecond)

	if s := output.String(); s != "" {
		t.Error("will not be written if it is not flushed %s", s)
	}

	testC <- time.Time{}
	<-fc

	if count != 2 {
		t.Errorf("the flushCallback function has not been called")
	}

	if s := output.String(); s != expected {
		t.Errorf("It will be written %q; but %q", expected, s)
	}

	cancel()
}
