package throttle

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	pr, pw := io.Pipe()

	output := new(bytes.Buffer)

	ex := NewExec(pr)

	ctx, cancel := context.WithCancel(context.Background())
	testC := make(chan time.Time)
	count := 0
	fc := make(chan struct{})

	flushCallback := func(_ context.Context, s string) error {
		defer func() {
			fc <- struct{}{}
			// to random fail from Go 1.12 or later
			time.Sleep(2 * time.Millisecond)
		}()

		count++

		output.WriteString(s)

		return nil
	}

	doneCount := 0

	doneCallback := func(_ context.Context, s string) error {
		defer func() {
			fc <- struct{}{}
		}()

		doneCount++

		output.WriteString(s)

		return nil
	}

	ex.Start(ctx, testC, flushCallback, doneCallback)

	testC <- time.Time{}
	<-fc

	if count != 1 {
		t.Error("the flushCallback function has not been called")
	}

	expected := []byte("abcd\nefgh\n")
	pw.Write(expected)

	if b := output.Bytes(); b != nil {
		t.Errorf("will not be written if it is not flushed %s", b)
	}

	testC <- time.Time{}
	<-fc

	if count != 2 {
		t.Errorf("the flushCallback function has not been called")
	}

	if b := output.Bytes(); !bytes.Equal(b, expected) {
		t.Errorf("It will be written %q; but %q", expected, b)
	}

	output.Reset()

	expected = []byte("ijk\nlmn\n")
	pw.Write(expected)

	// do not panic
	pw.Close()
	<-ex.Wait()

	cancel()
	<-fc

	if doneCount != 1 {
		t.Errorf("the doneCallback function has not been called")
	}

	if b := output.Bytes(); !bytes.Equal(b, expected) {
		t.Errorf("It will be written %q; but %q", expected, b)
	}
}
