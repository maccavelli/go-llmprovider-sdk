package llmtest

import (
	"context"
	"fmt"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Fake is a scriptable provider for tests. Each call to Generate takes the
// next reply scripted with Reply, ReplyText, Fail or Handle, and records its
// request. A request needing a capability the Fake lacks is refused, as a real
// provider refuses it. A Fake is safe for concurrent use.
type Fake struct {
	id   llmprovider.ProviderID
	caps llmprovider.Capabilities

	mu       sync.Mutex
	script   []func(context.Context, *llmprovider.Request) (*llmprovider.Response, error)
	requests []*llmprovider.Request
}

// NewFake returns a Fake with the given id and capabilities and no replies.
func NewFake(id llmprovider.ProviderID, caps llmprovider.Capabilities) *Fake {
	return &Fake{id: id, caps: caps}
}

// ID returns the Fake's id.
func (f *Fake) ID() llmprovider.ProviderID { return f.id }

// Capabilities returns the Fake's capabilities.
func (f *Fake) Capabilities() llmprovider.Capabilities { return f.caps }

// Reply scripts the next call to return a copy of resp.
func (f *Fake) Reply(resp *llmprovider.Response) *Fake {
	return f.Handle(func(context.Context, *llmprovider.Request) (*llmprovider.Response, error) {
		copied := *resp
		copied.Output = append([]llmprovider.Item(nil), resp.Output...)
		return &copied, nil
	})
}

// ReplyText scripts the next call to answer with text.
func (f *Fake) ReplyText(text string) *Fake {
	return f.Reply(&llmprovider.Response{
		Output:       []llmprovider.Item{llmprovider.MessageItem{Role: string(llmprovider.RoleAssistant), Text: text}},
		FinishReason: llmprovider.FinishStop,
	})
}

// Fail scripts the next call to fail with err.
func (f *Fake) Fail(err error) *Fake {
	return f.Handle(func(context.Context, *llmprovider.Request) (*llmprovider.Response, error) {
		return nil, err
	})
}

// Handle scripts the next call to run fn.
func (f *Fake) Handle(fn func(context.Context, *llmprovider.Request) (*llmprovider.Response, error)) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.script = append(f.script, fn)
	return f
}

// Generate runs the next scripted reply. With none left, it fails with an
// error matching ErrProviderUnavailable.
func (f *Fake) Generate(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := f.caps.Check(req); err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.requests = append(f.requests, cloneRequest(req))
	if len(f.script) == 0 {
		f.mu.Unlock()
		return nil, fmt.Errorf("%w: llmtest fake %q has no scripted reply left", llmprovider.ErrProviderUnavailable, f.id)
	}
	next := f.script[0]
	f.script = f.script[1:]
	f.mu.Unlock()
	return next(ctx, req)
}

// Requests returns copies of the requests Generate accepted, in order.
func (f *Fake) Requests() []*llmprovider.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*llmprovider.Request, len(f.requests))
	for i, req := range f.requests {
		out[i] = cloneRequest(req)
	}
	return out
}

// cloneRequest copies req and the slices and reasoning it points to.
func cloneRequest(req *llmprovider.Request) *llmprovider.Request {
	copied := *req
	copied.Input = append([]llmprovider.Item(nil), req.Input...)
	copied.Tools = append([]llmprovider.Tool(nil), req.Tools...)
	if req.Reasoning != nil {
		reasoning := *req.Reasoning
		copied.Reasoning = &reasoning
	}
	return &copied
}
