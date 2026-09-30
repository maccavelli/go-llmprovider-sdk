package llmprovider

import (
	"context"
	"fmt"
)

// GenerateText runs req on p and returns the text of its output (0015-MADR
// D3; the name is the standards guide's R8).
func GenerateText(ctx context.Context, p Provider, req *Request) (string, error) {
	resp, err := generate(ctx, p, req)
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

// GenerateToolCall forces req's one tool and returns the model's call to it
// (0015-MADR D3; R8). A request without exactly one tool gives an error
// matching ErrInvalidRequest, and a response without the call one matching
// ErrIncomplete.
func GenerateToolCall(ctx context.Context, p Provider, req *Request) (FunctionCallItem, error) {
	if req == nil || len(req.Tools) != 1 {
		n := 0
		if req != nil {
			n = len(req.Tools)
		}
		return FunctionCallItem{}, fmt.Errorf("%w: GenerateToolCall needs one tool, not %d", ErrInvalidRequest, n)
	}
	name := req.Tools[0].Name
	forced := *req
	forced.ToolChoice = ForceTool(name)
	resp, err := generate(ctx, p, &forced)
	if err != nil {
		return FunctionCallItem{}, err
	}
	for _, item := range resp.Output {
		if call, ok := item.(FunctionCallItem); ok && call.Name == name {
			return call, nil
		}
	}
	return FunctionCallItem{}, fmt.Errorf("%w: %s returned no call to tool %q", ErrIncomplete, p.ID(), name)
}

// generate is p.Generate, with a nil response treated as incomplete.
func generate(ctx context.Context, p Provider, req *Request) (*Response, error) {
	resp, err := p.Generate(ctx, req)
	if err == nil && resp == nil {
		err = fmt.Errorf("%w: %s returned no response", ErrIncomplete, p.ID())
	}
	return resp, err
}
