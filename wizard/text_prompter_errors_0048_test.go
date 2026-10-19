package wizard

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// errRead0048 is the input's own failure, as a hung-up terminal's.
var errRead0048 = errors.New("read: input/output error")

// errWrite0048 is the output's own failure, as a closed terminal's.
var errWrite0048 = errors.New("write: broken pipe")

// errWriter0048 refuses every write with errWrite0048.
type errWriter0048 struct{}

func (errWriter0048) Write([]byte) (int, error) { return 0, errWrite0048 }

// kinds0048 are the kinds a plain error matches none of: llmprovider's
// sentinels, the end of input, and the context's errors.
var kinds0048 = []error{
	llmprovider.ErrRateLimited, llmprovider.ErrQuotaExhausted, llmprovider.ErrProviderUnavailable,
	llmprovider.ErrAuthFailure, llmprovider.ErrNotPermitted, llmprovider.ErrInvalidRequest,
	llmprovider.ErrContextOverflow, llmprovider.ErrIncomplete, llmprovider.ErrUnsupported,
	llmprovider.ErrInvalidProvider, io.EOF, context.Canceled, context.DeadlineExceeded,
}

// assertPlain0048 fails t unless err is the plain error want: non-nil, its
// message want's, and matching none of kinds0048.
func assertPlain0048(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err = nil; want the plain error %q", want)
	}
	for _, kind := range kinds0048 {
		if errors.Is(err, kind) {
			t.Errorf("err = %v matches %v; want a plain error", err, kind)
		}
	}
	if err.Error() != want {
		t.Errorf("err = %q; want %q", err, want)
	}
}

// exhausted0048 is a TextPrompter whose input has already ended: its first
// read answered the prompt's default (0020-MADR F6).
func exhausted0048(t *testing.T) *TextPrompter {
	t.Helper()
	p := &TextPrompter{In: strings.NewReader(""), Out: io.Discard}
	if _, err := p.Input("first", "def"); err != nil {
		t.Fatalf("first Input at the end of input = %v; want its default", err)
	}
	return p
}

var choices0048 = []Choice{{Label: "a"}, {Label: "b"}}

// TestTextPrompter_SelectNoChoicesIsPlain_0048: Select with no choices is a
// plain error (0048-MADR D4).
func TestTextPrompter_SelectNoChoicesIsPlain_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("1\n"), Out: io.Discard}
	_, err := p.Select("Pick", nil, 0)
	assertPlain0048(t, err, "wizard: Select called with no choices")
}

// TestTextPrompter_SelectReadError_0048: Select returns In's read error
// (0048-MADR D4).
func TestTextPrompter_SelectReadError_0048(t *testing.T) {
	p := &TextPrompter{In: iotest.ErrReader(errRead0048), Out: io.Discard}
	if _, err := p.Select("Pick", choices0048, 0); !errors.Is(err, errRead0048) {
		t.Errorf("Select = %v; want the reader's error", err)
	}
}

// TestTextPrompter_SelectExhaustedIsEOF_0048: a Select after the input ended
// matches io.EOF (0048-MADR D4).
func TestTextPrompter_SelectExhaustedIsEOF_0048(t *testing.T) {
	if _, err := exhausted0048(t).Select("Pick", choices0048, 0); !errors.Is(err, io.EOF) {
		t.Errorf("Select after the end of input = %v; want io.EOF", err)
	}
}

// TestTextPrompter_SelectInvalidAtEndIsPlain_0048: an invalid answer on the
// input's last line is a plain error, not io.EOF (0048-MADR D4).
func TestTextPrompter_SelectInvalidAtEndIsPlain_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("x"), Out: io.Discard}
	_, err := p.Select("Pick", choices0048, 0)
	assertPlain0048(t, err, "wizard: input exhausted while selecting")
}

// TestTextPrompter_SelectWriteError_0048: Select returns Out's write error
// (0048-MADR D4).
func TestTextPrompter_SelectWriteError_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("1\n"), Out: errWriter0048{}}
	if _, err := p.Select("Pick", choices0048, 0); !errors.Is(err, errWrite0048) {
		t.Errorf("Select = %v; want the writer's error", err)
	}
}

// TestTextPrompter_MultiSelectNoChoices_0048: MultiSelect with no choices is
// nil, nil (0048-MADR D4).
func TestTextPrompter_MultiSelectNoChoices_0048(t *testing.T) {
	p := &TextPrompter{In: iotest.ErrReader(errRead0048), Out: errWriter0048{}}
	if got, err := p.MultiSelect("Pick any", nil, []int{0}); got != nil || err != nil {
		t.Errorf("MultiSelect = %v, %v; want nil, nil", got, err)
	}
}

// TestTextPrompter_MultiSelectReadError_0048: MultiSelect returns In's read
// error (0048-MADR D4).
func TestTextPrompter_MultiSelectReadError_0048(t *testing.T) {
	p := &TextPrompter{In: iotest.ErrReader(errRead0048), Out: io.Discard}
	if _, err := p.MultiSelect("Pick any", choices0048, nil); !errors.Is(err, errRead0048) {
		t.Errorf("MultiSelect = %v; want the reader's error", err)
	}
}

// TestTextPrompter_MultiSelectExhaustedIsEOF_0048: a MultiSelect after the
// input ended matches io.EOF (0048-MADR D4).
func TestTextPrompter_MultiSelectExhaustedIsEOF_0048(t *testing.T) {
	if _, err := exhausted0048(t).MultiSelect("Pick any", choices0048, nil); !errors.Is(err, io.EOF) {
		t.Errorf("MultiSelect after the end of input = %v; want io.EOF", err)
	}
}

// TestTextPrompter_MultiSelectInvalidAtEndIsPlain_0048: an invalid answer on
// the input's last line is a plain error, not io.EOF (0048-MADR D4).
func TestTextPrompter_MultiSelectInvalidAtEndIsPlain_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("1,x"), Out: io.Discard}
	_, err := p.MultiSelect("Pick any", choices0048, nil)
	assertPlain0048(t, err, "wizard: input exhausted while selecting")
}

// TestTextPrompter_MultiSelectWriteError_0048: MultiSelect returns Out's
// write error (0048-MADR D4).
func TestTextPrompter_MultiSelectWriteError_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("1\n"), Out: errWriter0048{}}
	if _, err := p.MultiSelect("Pick any", choices0048, nil); !errors.Is(err, errWrite0048) {
		t.Errorf("MultiSelect = %v; want the writer's error", err)
	}
}

// TestTextPrompter_ConfirmReadError_0048: Confirm returns In's read error
// (0048-MADR D4).
func TestTextPrompter_ConfirmReadError_0048(t *testing.T) {
	p := &TextPrompter{In: iotest.ErrReader(errRead0048), Out: io.Discard}
	if _, err := p.Confirm("OK?", true); !errors.Is(err, errRead0048) {
		t.Errorf("Confirm = %v; want the reader's error", err)
	}
}

// TestTextPrompter_ConfirmExhaustedIsEOF_0048: a Confirm after the input
// ended matches io.EOF (0048-MADR D4).
func TestTextPrompter_ConfirmExhaustedIsEOF_0048(t *testing.T) {
	if _, err := exhausted0048(t).Confirm("OK?", true); !errors.Is(err, io.EOF) {
		t.Errorf("Confirm after the end of input = %v; want io.EOF", err)
	}
}

// TestTextPrompter_ConfirmWriteError_0048: Confirm returns Out's write error
// (0048-MADR D4).
func TestTextPrompter_ConfirmWriteError_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("y\n"), Out: errWriter0048{}}
	if _, err := p.Confirm("OK?", false); !errors.Is(err, errWrite0048) {
		t.Errorf("Confirm = %v; want the writer's error", err)
	}
}

// TestTextPrompter_InputReadError_0048: Input returns In's read error
// (0048-MADR D4).
func TestTextPrompter_InputReadError_0048(t *testing.T) {
	p := &TextPrompter{In: iotest.ErrReader(errRead0048), Out: io.Discard}
	if _, err := p.Input("URL", "def"); !errors.Is(err, errRead0048) {
		t.Errorf("Input = %v; want the reader's error", err)
	}
}

// TestTextPrompter_InputExhaustedIsEOF_0048: an Input after the input ended
// matches io.EOF (0048-MADR D4).
func TestTextPrompter_InputExhaustedIsEOF_0048(t *testing.T) {
	if _, err := exhausted0048(t).Input("URL", "def"); !errors.Is(err, io.EOF) {
		t.Errorf("Input after the end of input = %v; want io.EOF", err)
	}
}

// TestTextPrompter_InputWriteError_0048: Input returns Out's write error
// (0048-MADR D4).
func TestTextPrompter_InputWriteError_0048(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("value\n"), Out: errWriter0048{}}
	if _, err := p.Input("URL", ""); !errors.Is(err, errWrite0048) {
		t.Errorf("Input = %v; want the writer's error", err)
	}
}
