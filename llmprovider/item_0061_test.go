package llmprovider

import (
	"errors"
	"strings"
	"testing"
)

// TestImageItem_Validate_0061 (0061-MADR D5): each of D5's refusals is an
// ErrInvalidRequest naming the input, a refused URL is not quoted, and an
// image by Data in each of the four media types, or by an https URL with or
// without one, is valid.
func TestImageItem_Validate_0061(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G'}
	const page = "https://example.com/cat.png"
	for _, c := range []struct {
		name string
		img  ImageItem
		want string // in the error; empty when valid
	}{
		{"the assistant's", ImageItem{Role: RoleAssistant, MediaType: "image/png", Data: png}, `has role "assistant"`},
		{"the system's", ImageItem{Role: RoleSystem, MediaType: "image/png", Data: png}, `has role "system"`},
		{"an unknown role", ImageItem{Role: "model", MediaType: "image/png", Data: png}, `has role "model"`},
		{"both Data and URL", ImageItem{MediaType: "image/png", Data: png, URL: page}, "sets both Data and URL"},
		{"neither", ImageItem{MediaType: "image/png"}, "sets neither Data nor URL"},
		{"empty Data", ImageItem{MediaType: "image/png", Data: []byte{}}, "sets neither Data nor URL"},
		{"an http URL", ImageItem{URL: "http://example.com/cat.png?sig=s3cr3t"}, "has a URL that is not an absolute https URL"},
		{"a relative URL", ImageItem{URL: "/cat.png"}, "has a URL that is not an absolute https URL"},
		{"an https URL with no host", ImageItem{URL: "https:cat.png"}, "has a URL that is not an absolute https URL"},
		{"a URL that does not parse", ImageItem{URL: "https://example.com/%zz"}, "has a URL that is not an absolute https URL"},
		{"a data URL", ImageItem{URL: "data:image/png;base64,iVBORw0KGgo="}, "has a URL that is not an absolute https URL"},
		{"Data with no media type", ImageItem{Data: png}, `has media type ""`},
		{"Data as BMP", ImageItem{MediaType: "image/bmp", Data: png}, `has media type "image/bmp"`},
		{"Data's media type in capitals", ImageItem{MediaType: "IMAGE/PNG", Data: png}, `has media type "IMAGE/PNG"`},
		{"a URL as BMP", ImageItem{MediaType: "image/bmp", URL: page}, `has media type "image/bmp"`},
		{"a PNG", ImageItem{MediaType: "image/png", Data: png}, ""},
		{"a JPEG, the user's", ImageItem{Role: RoleUser, MediaType: "image/jpeg", Data: png}, ""},
		{"a GIF", ImageItem{MediaType: "image/gif", Data: png}, ""},
		{"a WebP", ImageItem{MediaType: "image/webp", Data: png}, ""},
		{"a URL with no media type", ImageItem{URL: page}, ""},
		{"a URL with one", ImageItem{Role: RoleUser, MediaType: "image/jpeg", URL: page}, ""},
		{"a URL's scheme in capitals", ImageItem{URL: "HTTPS://example.com/cat.png"}, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			req := &Request{Input: []Item{MessageItem{Role: RoleUser, Text: "what is this?"}, c.img}}
			err := req.validate()
			if c.want == "" {
				if err != nil {
					t.Fatalf("validate = %v; want valid", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("validate = %v; want an error matching ErrInvalidRequest", err)
			}
			if msg := err.Error(); !strings.Contains(msg, "input 1 is an image that "+c.want) {
				t.Errorf("validate = %q; want it to say %q", msg, "input 1 is an image that "+c.want)
			}
			if c.img.URL != "" && strings.Contains(err.Error(), c.img.URL) {
				t.Errorf("validate = %q; want the URL left out", err)
			}
		})
	}
	if got := (ImageItem{}).itemKind(); got != "image" {
		t.Errorf("itemKind = %q; want %q", got, "image")
	}
	err := (&Request{Input: []Item{nil}}).validate()
	if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), "ToolLoadItem or ImageItem value") {
		t.Errorf("a nil item: %v; want ErrInvalidRequest naming ImageItem among the types", err)
	}
	if WarningImageDropped != "image_dropped" {
		t.Errorf("WarningImageDropped = %q; want %q", WarningImageDropped, "image_dropped")
	}
}

// TestCheck_ImagesUnsupported_0061 (0061-MADR D5): Check refuses a request
// holding an image where Images is Unsupported, with ErrUnsupported naming
// the need; takes it where Images is BestEffort or Supported; refuses an
// invalid image as invalid first; and takes a request with no image whatever
// Images says.
func TestCheck_ImagesUnsupported_0061(t *testing.T) {
	image := ImageItem{MediaType: "image/png", Data: []byte{0x89, 'P', 'N', 'G'}}
	withImage := &Request{Input: []Item{MessageItem{Role: RoleUser, Text: "what is this?"}, image}}
	err := (Capabilities{}).Check(withImage)
	if !errors.Is(err, ErrUnsupported) || !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Images Unsupported: Check = %v; want an error matching ErrUnsupported and errors.ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "the request needs image input") {
		t.Errorf("Images Unsupported: Check = %q; want the need named", err)
	}
	for _, support := range []Support{BestEffort, Supported} {
		if err := (Capabilities{Images: support}).Check(withImage); err != nil {
			t.Errorf("Images %v: Check = %v; want nil", support, err)
		}
	}
	invalid := &Request{Input: []Item{ImageItem{MediaType: "image/png"}}}
	if err := (Capabilities{}).Check(invalid); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("an invalid image, Images Unsupported: Check = %v; want ErrInvalidRequest", err)
	}
	if err := (Capabilities{}).Check(&Request{Input: textInput("hi")}); err != nil {
		t.Errorf("no image, Images Unsupported: Check = %v; want nil", err)
	}
}
