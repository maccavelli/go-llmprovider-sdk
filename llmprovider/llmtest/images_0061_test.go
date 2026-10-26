package llmtest

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRun_ImagesNeed_0061 (0061-MADR D5): a provider that sends a request
// holding an image while it declares Images Unsupported fails R11, naming the
// need, as for any other need.
func TestRun_ImagesNeed_0061(t *testing.T) {
	rec := newRecorder()
	runChecks(rec, refHarness(flaws{skipCheck: true}))
	const want = "R10-R11-capabilities: R11 (refusal before the network): a request needing images, which is Unsupported"
	for _, msg := range rec.failures() {
		if strings.HasPrefix(msg, want) {
			return
		}
	}
	t.Fatalf("failures %q; want one starting %q", rec.failures(), want)
}

// TestImageRequest_0061: the suite's image is a 1×1 PNG, sent by its bytes in
// the user's turn after the text.
func TestImageRequest_0061(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(onePixelPNG))
	if err != nil {
		t.Fatalf("decoding onePixelPNG: %v", err)
	}
	if got := img.Bounds(); got != image.Rect(0, 0, 1, 1) {
		t.Errorf("onePixelPNG's bounds = %v; want 1×1", got)
	}
	req := imageRequest()
	if len(req.Input) != 2 {
		t.Fatalf("imageRequest's Input = %d items; want the text, then the image", len(req.Input))
	}
	// Unsupported, not invalid: the image passed validation.
	if err := refCaps.Check(req); !errors.Is(err, llmprovider.ErrUnsupported) {
		t.Errorf("imageRequest with Images Unsupported: Check = %v; want ErrUnsupported", err)
	}
}
