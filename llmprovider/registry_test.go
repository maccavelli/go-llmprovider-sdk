package llmprovider

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func stubFactory(id ProviderID) Factory {
	return func(opts ...Option) (Provider, error) {
		st, err := ResolveOptions(id, opts)
		if err != nil {
			return nil, err
		}
		return &stubProvider{id: id, gen: func(context.Context, *Request) (*Response, error) {
			return &Response{Model: st.Model()}, nil
		}}, nil
	}
}

func TestRegistry_RefusesADuplicate(t *testing.T) {
	var r Registry // the zero value is ready to use
	if err := r.Register(Descriptor{ID: "a"}, stubFactory("a")); err != nil {
		t.Fatal(err)
	}
	err := r.Register(Descriptor{ID: "a", Label: "again"}, stubFactory("a"))
	if !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("a duplicate: err = %v, want ErrInvalidProvider", err)
	}
	if d, _ := r.Descriptor("a"); d.Label != "" {
		t.Fatalf("the duplicate replaced the first: %+v", d)
	}
}

func TestRegistry_RefusesAnIncompleteEntry(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(Descriptor{}, stubFactory("x")); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("no id: err = %v", err)
	}
	if err := r.Register(Descriptor{ID: "x"}, nil); !errors.Is(err, ErrInvalidProvider) {
		t.Errorf("no factory: err = %v", err)
	}
	if n := len(r.Descriptors()); n != 0 {
		t.Fatalf("%d descriptors registered, want none", n)
	}
}

func TestRegistry_NewBuildsByIDWithOptions(t *testing.T) {
	r := NewRegistry()
	for _, id := range []ProviderID{"b", "a"} {
		if err := r.Register(Descriptor{ID: id}, stubFactory(id)); err != nil {
			t.Fatal(err)
		}
	}
	p, err := r.New("a", WithModel("m"))
	if err != nil || p.ID() != "a" {
		t.Fatalf("New(a) = %v, %v", p, err)
	}
	if resp, _ := p.Generate(context.Background(), &Request{}); resp.Model != "m" {
		t.Fatalf("the option did not reach the factory: model %q", resp.Model)
	}
	if _, err := r.New("missing"); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("an unknown id: err = %v", err)
	}
	if _, err := r.New("a", ScopedOption("b", "b.WithThing", 1)); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("a foreign option through the registry: err = %v", err)
	}
	var order []ProviderID
	for _, d := range r.Descriptors() {
		order = append(order, d.ID)
	}
	if len(order) != 2 || order[0] != "b" || order[1] != "a" {
		t.Fatalf("order %v, want registration order [b a]", order)
	}
}

func TestRegistry_RefusesAFactoryThatBuildsAnotherProvider(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(Descriptor{ID: "a"}, stubFactory("b")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.New("a"); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("err = %v, want ErrInvalidProvider", err)
	}
	if err := r.Register(Descriptor{ID: "nil"}, func(...Option) (Provider, error) { return nil, nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := r.New("nil"); !errors.Is(err, ErrInvalidProvider) {
		t.Fatalf("a nil provider: err = %v, want ErrInvalidProvider", err)
	}
}

func TestRegistry_ReturnsCopies(t *testing.T) {
	r := NewRegistry()
	d := Descriptor{ID: "a", AuthMethods: []AuthMethod{{ID: AuthAPIKey}}, StaticModels: []string{"m1"}}
	if err := r.Register(d, stubFactory("a")); err != nil {
		t.Fatal(err)
	}
	d.StaticModels[0] = "changed by the registrant"
	got, ok := r.Descriptor("a")
	if !ok || got.StaticModels[0] != "m1" {
		t.Fatalf("Descriptor(a) = %+v, %v", got, ok)
	}
	got.AuthMethods[0].ID = AuthDeviceCode
	r.Descriptors()[0].StaticModels[0] = "changed by a caller"
	again, _ := r.Descriptor("a")
	if again.AuthMethods[0].ID != AuthAPIKey || again.StaticModels[0] != "m1" {
		t.Fatalf("a caller's change reached the registry: %+v", again)
	}
	if _, ok := r.Descriptor("missing"); ok {
		t.Fatal("Descriptor(missing) found one")
	}
}

func TestRegistry_ConcurrentUse(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := range 8 {
		id := ProviderID(rune('a' + i))
		wg.Go(func() {
			if err := r.Register(Descriptor{ID: id}, stubFactory(id)); err != nil {
				t.Error(err)
			}
			_ = r.Descriptors()
			if _, err := r.New(id); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := len(r.Descriptors()); n != 8 {
		t.Fatalf("%d descriptors, want 8", n)
	}
}
