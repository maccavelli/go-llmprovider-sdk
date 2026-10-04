package transport

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// TestReplyReader_Limit (0021-MADR W6): a body of exactly the limit reads
// whole; one byte more is ErrReplyTooLarge, with the limit's bytes returned.
func TestReplyReader_Limit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	rr := NewReplyReader(ctx, cancel, strings.NewReader("12345"), 5, 0)
	if got, err := io.ReadAll(rr); err != nil || string(got) != "12345" || rr.ReadErr() != nil {
		t.Errorf("a body at the limit: %q, %v, read error %v; want it whole", got, err, rr.ReadErr())
	}
	rr = NewReplyReader(ctx, cancel, strings.NewReader("123456"), 5, 0)
	got, err := io.ReadAll(rr)
	if !errors.Is(err, ErrReplyTooLarge) || string(got) != "12345" || rr.ReadErr() != nil {
		t.Errorf("a body over the limit: %q, %v, read error %v; want ErrReplyTooLarge after 5 bytes", got, err, rr.ReadErr())
	}
	rr = NewReplyReader(ctx, cancel, strings.NewReader(strings.Repeat("x", 100)), 0, 0)
	if got, err := io.ReadAll(rr); err != nil || len(got) != 100 {
		t.Errorf("no limit: %d bytes, %v; want 100", len(got), err)
	}
	rr.Stop()
}

// failAfter returns data, then err.
type failAfter struct {
	data string
	err  error
}

func (f *failAfter) Read(p []byte) (int, error) {
	if f.data == "" {
		return 0, f.err
	}
	n := copy(p, f.data)
	f.data = f.data[n:]
	return n, nil
}

// TestReplyReader_KeepsTheFirstReadFailure (0021-MADR D1): a cut body's read
// failure is kept for DecodeError; io.EOF is not one.
func TestReplyReader_KeepsTheFirstReadFailure(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	reset := &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}
	rr := NewReplyReader(ctx, cancel, &failAfter{data: "{", err: reset}, 0, time.Minute)
	defer rr.Stop()
	if _, err := io.ReadAll(rr); !errors.Is(err, reset) || !errors.Is(rr.ReadErr(), reset) {
		t.Errorf("read error %v, kept %v; want the reset", err, rr.ReadErr())
	}
	if _, err := rr.Read(make([]byte, 1)); !errors.Is(rr.ReadErr(), reset) || err == nil {
		t.Errorf("a second failure replaced the first: %v", rr.ReadErr())
	}
}

// blockingBody returns nothing until its context ends.
type blockingBody struct{ ctx context.Context }

func (b blockingBody) Read([]byte) (int, error) {
	<-b.ctx.Done()
	return 0, context.Cause(b.ctx)
}

// TestReplyReader_IdleLimit (0021-MADR D2): a body that sends nothing for the
// idle limit cancels its request, and the failure kept is ErrIdleTimeout.
func TestReplyReader_IdleLimit(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	rr := NewReplyReader(ctx, cancel, blockingBody{ctx}, 0, 20*time.Millisecond)
	defer rr.Stop()
	_, err := rr.Read(make([]byte, 8))
	if !errors.Is(err, ErrIdleTimeout) || !errors.Is(rr.ReadErr(), ErrIdleTimeout) {
		t.Fatalf("an idle body: %v, kept %v; want ErrIdleTimeout", err, rr.ReadErr())
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Errorf("ErrIdleTimeout is not a timeout net.Error: %v", err)
	}
	if ErrIdleTimeout.Error() != "no data for 300s" {
		t.Errorf("ErrIdleTimeout = %q", ErrIdleTimeout)
	}
}

// TestAfterReply (0021-MADR D1): the mark wraps and unwraps; nil stays nil.
func TestAfterReply(t *testing.T) {
	if AfterReply(nil) != nil || IsAfterReply(nil) || IsAfterReply(io.EOF) {
		t.Error("AfterReply(nil) or IsAfterReply of an unmarked error")
	}
	marked := AfterReply(io.ErrUnexpectedEOF)
	if !IsAfterReply(marked) || !errors.Is(marked, io.ErrUnexpectedEOF) || marked.Error() != io.ErrUnexpectedEOF.Error() {
		t.Errorf("AfterReply = %v; want the mark around the error, which still matches", marked)
	}
}
