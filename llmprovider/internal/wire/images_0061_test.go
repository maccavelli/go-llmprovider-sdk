package wire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDataURL_0061 (0061-MADR D5): an image's bytes as a data URL.
func TestDataURL_0061(t *testing.T) {
	got := DataURL(llmprovider.ImageItem{MediaType: "image/jpeg", Data: []byte("PNG")})
	if want := "data:image/jpeg;base64,UE5H"; got != want {
		t.Errorf("DataURL = %q; want %q", got, want)
	}
}

// TestRefuseImages_0061 (0061-MADR D5): a model is refused only when its
// facts state its input, without image, and the input holds an image.
func TestRefuseImages_0061(t *testing.T) {
	image := []llmprovider.Item{llmprovider.MessageItem{Text: "look"}, llmprovider.ImageItem{URL: "https://example.com/a.png"}}
	text := []llmprovider.Item{llmprovider.MessageItem{Text: "look"}}
	for _, c := range []struct {
		name               string
		input              []llmprovider.Item
		known, takesImages bool
		refused            bool
	}{
		{"facts say text only", image, true, false, true},
		{"facts say images", image, true, true, false},
		{"no facts", image, false, false, false},
		{"no image", text, true, false, false},
	} {
		err := RefuseImages("text-model", c.input, c.known, c.takesImages)
		switch {
		case c.refused && (!errors.Is(err, llmprovider.ErrUnsupported) || !strings.Contains(err.Error(), `"text-model"`)):
			t.Errorf("%s: %v; want ErrUnsupported naming the model", c.name, err)
		case !c.refused && err != nil:
			t.Errorf("%s: %v; want nil", c.name, err)
		}
	}
}

// TestDropImageURLs_0061 (0061-MADR D5): each image by URL is left out, with
// a WarningImageDropped naming its index and why but not its URL; an image's
// bytes and every other item are kept in order; input with nothing to drop
// is returned as it is.
func TestDropImageURLs_0061(t *testing.T) {
	const secret = "https://example.com/a.png?sig=s3cr3t"
	data := llmprovider.ImageItem{MediaType: "image/png", Data: []byte("PNG")}
	input := []llmprovider.Item{llmprovider.MessageItem{Text: "look"}, llmprovider.ImageItem{URL: secret}, data,
		llmprovider.ImageItem{URL: "https://example.com/b.png"}}
	kept, warnings := DropImageURLs("m", input, "no URLs here")
	if len(kept) != 2 || kept[0] != input[0] {
		t.Fatalf("kept = %v; want the text and the image's bytes", kept)
	}
	if img, ok := kept[1].(llmprovider.ImageItem); !ok || string(img.Data) != "PNG" {
		t.Fatalf("kept[1] = %v; want the image by its bytes", kept[1])
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings = %v; want two", warnings)
	}
	for i, at := range []string{"Input[1]", "Input[3]"} {
		w := warnings[i]
		if w.Kind != llmprovider.WarningImageDropped || w.Model != "m" ||
			w.Detail != at+", an image by URL, was not sent: no URLs here" {
			t.Errorf("warnings[%d] = %+v; want WarningImageDropped for %s", i, w, at)
		}
		if strings.Contains(w.Detail, "s3cr3t") {
			t.Errorf("warnings[%d] quotes the URL: %q", i, w.Detail)
		}
	}
	plain := []llmprovider.Item{llmprovider.MessageItem{Text: "look"}, data}
	if got, warnings := DropImageURLs("m", plain, "x"); &got[0] != &plain[0] || warnings != nil {
		t.Errorf("nothing to drop: %v, %v; want the input as it is and no warning", got, warnings)
	}
}

// TestCallRefusal_0061 (0061-MADR, amendment "D5 made exact before Phase
// 6", item 3): a Call carrying a Refusal is not sent, by Post or by
// OpenStream; its Refusal is returned.
func TestCallRefusal_0061(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = io.WriteString(w, "{}")
	}))
	defer srv.Close()
	refusal := errors.New("refused here")
	c := Call{Provider: "test", Client: srv.Client(), URL: srv.URL, Body: map[string]any{}, Refusal: refusal,
		Prepare: func(*http.Request, llmprovider.Token) {}}
	if _, err := Post(context.Background(), c, llmprovider.Token{}, func(io.Reader) (any, error) { return nil, nil }); !errors.Is(err, refusal) {
		t.Errorf("Post = %v; want the refusal", err)
	}
	if _, err := OpenStream(context.Background(), c, llmprovider.Token{}); !errors.Is(err, refusal) {
		t.Errorf("OpenStream = %v; want the refusal", err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("%d request(s) sent; want none", n)
	}
	c.Refusal = nil
	if _, err := Post(context.Background(), c, llmprovider.Token{}, func(io.Reader) (any, error) { return nil, nil }); err != nil || hits.Load() != 1 {
		t.Errorf("with no Refusal: %v, %d request(s); want it sent", err, hits.Load())
	}
}
