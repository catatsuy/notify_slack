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

// Exec reads input line by line and buffers it, flushing at specified intervals.
// It's designed to batch multiple lines of output before sending to Slack.
type Exec struct {
	reader *bufio.Reader
	rc     io.ReadCloser  // Optional: for closing the input when needed
	pr     *io.PipeReader // Optional: for pipe-specific closing

	buffer *bytes.Buffer
	mu     sync.Mutex // Protects buffer access

	done chan struct{} // Signals when reading is complete
}

// NewExec creates a new Exec that reads from the given input.
func NewExec(input io.Reader) *Exec {
	ex := &Exec{
		reader: bufio.NewReader(input),
		buffer: new(bytes.Buffer),
		done:   make(chan struct{}),
		mu:     sync.Mutex{},
	}

	// Store references to closers if the input supports closing
	// This allows us to interrupt blocked reads on context cancellation
	if rc, ok := input.(io.ReadCloser); ok {
		ex.rc = rc
	}
	if pr, ok := input.(*io.PipeReader); ok {
		ex.pr = pr
	}

	return ex
}

// Start begins reading input and processing it.
// - Reads input line by line in a background goroutine
// - Flushes buffered content on each interval tick
// - Calls doneCallback with remaining content when input closes or context is cancelled
func (ex *Exec) Start(
	ctx context.Context,
	interval <-chan time.Time,
	flushCallback func(ctx context.Context, output string) error,
	doneCallback func(ctx context.Context, output string) error,
) {
	// Start background reader
	go ex.readInput()

	// Process events
	ex.processEvents(ctx, interval, flushCallback, doneCallback)
}

// readInput reads lines from input until EOF or error
func (ex *Exec) readInput() {
	defer close(ex.done)

	for {
		line, _, err := ex.reader.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) ||
				errors.Is(err, io.ErrClosedPipe) ||
				errors.Is(err, context.Canceled) {
				return
			}
			panic(err) // Unexpected error
		}

		ex.appendLine(line)
	}
}

// processEvents handles interval ticks and completion events
func (ex *Exec) processEvents(
	ctx context.Context,
	interval <-chan time.Time,
	flushCallback func(ctx context.Context, output string) error,
	doneCallback func(ctx context.Context, output string) error,
) {
	for {
		select {
		case <-interval:
			// Periodic flush of buffered content
			flushCallback(ctx, ex.getAndResetBuffer())

		case <-ctx.Done():
			// Context cancelled - stop reading and flush remaining content
			ex.closeInput(ctx.Err())
			doneCallback(ctx, ex.getAndResetBuffer())
			return

		case <-ex.done:
			// Input closed - flush remaining content
			doneCallback(ctx, ex.getAndResetBuffer())
			return
		}
	}
}

// appendLine adds a line to the buffer (thread-safe)
func (ex *Exec) appendLine(line []byte) {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	ex.buffer.Write(line)
	ex.buffer.WriteByte('\n')
}

// getAndResetBuffer returns the buffer content and clears it (thread-safe)
func (ex *Exec) getAndResetBuffer() string {
	ex.mu.Lock()
	defer ex.mu.Unlock()

	content := ex.buffer.String()
	ex.buffer.Reset()
	return content
}

// closeInput closes the underlying reader to unblock any blocked read operation
func (ex *Exec) closeInput(err error) {
	// Try pipe-specific close first (if applicable)
	if ex.pr != nil {
		ex.pr.CloseWithError(err)
		return
	}

	// Otherwise try generic close
	if ex.rc != nil {
		ex.rc.Close()
	}
}
