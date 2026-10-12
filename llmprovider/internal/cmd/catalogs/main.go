// Command catalogs proposes the next curated model lists, for a person to
// review (0033-MADR D4). It reads the models.dev document, and each listing
// whose provider's key variable is set, and prints internal/catalogtool's
// proposals, one per line. It edits nothing, and prints no key.
//
// Usage:
//
//	go run ./llmprovider/internal/cmd/catalogs [-doc FILE]
//
// -doc reads a models.dev-format document from FILE instead of fetching one.
// Exit codes: 0 the proposals were printed, 2 a tool error.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/catalogtool"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, KeysFromEnv(), liveFetcher{})) }

// KeysFromEnv reads each provider's key from its variable
// (llmprovider.ProviderEnvVars), for its listing alone. It is the one place the
// command reads the environment (R30), and no key is printed.
func KeysFromEnv() map[llmprovider.ProviderID]string {
	keys := map[llmprovider.ProviderID]string{}
	for p, v := range llmprovider.ProviderEnvVars() {
		if key := os.Getenv(v); key != "" {
			keys[p] = key
		}
	}
	return keys
}

// fetcher reads the sources: the document, and one provider's listing.
type fetcher interface {
	Metadata(ctx context.Context) (catalog.Metadata, error)
	Listed(ctx context.Context, provider llmprovider.ProviderID, key string) ([]string, error)
}

// liveFetcher reads the sources from the network.
type liveFetcher struct{}

func (liveFetcher) Metadata(ctx context.Context) (catalog.Metadata, error) {
	return catalog.LookupMetadata(ctx, "", nil)
}

func (liveFetcher) Listed(ctx context.Context, provider llmprovider.ProviderID, key string) ([]string, error) {
	cat, err := catalog.List(ctx, provider, llmprovider.NewStaticToken(key), llmprovider.WithoutModelMetadata())
	if err == nil {
		err = cat.Err
	}
	return cat.Usable, err
}

// writer writes lines, and keeps the first write error.
type writer struct {
	w   io.Writer
	err error
}

func (l *writer) printf(format string, a ...any) {
	if l.err == nil {
		_, l.err = fmt.Fprintf(l.w, format, a...)
	}
}

// run is the command, with its arguments, output, keys and sources given, so
// that it can be tested without a network or a key.
func run(args []string, stdout, stderr io.Writer, keys map[llmprovider.ProviderID]string, f fetcher) int {
	out, errOut := &writer{w: stdout}, &writer{w: stderr}
	code := runTo(args, out, errOut, stderr, keys, f)
	if out.err != nil || errOut.err != nil {
		return 2
	}
	return code
}

// runTo is run, writing through out and errOut.
func runTo(args []string, out, errOut *writer, stderr io.Writer, keys map[llmprovider.ProviderID]string, f fetcher) int {
	flags := flag.NewFlagSet("catalogs", flag.ContinueOnError)
	flags.SetOutput(stderr)
	doc := flags.String("doc", "", "read a models.dev-format document from this file instead of fetching one")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	meta, err := readMetadata(ctx, *doc, f)
	if err != nil {
		errOut.printf("catalogs: %v\n", err)
		return 2
	}
	vars := llmprovider.ProviderEnvVars()
	providers := make([]llmprovider.ProviderID, 0, len(vars))
	for p := range vars {
		providers = append(providers, p)
	}
	slices.Sort(providers)
	static := map[llmprovider.ProviderID][]string{}
	listed := map[llmprovider.ProviderID][]string{}
	for _, p := range providers {
		if ids := catalog.Static(p); len(ids) > 0 {
			static[p] = ids
		}
		key := keys[p]
		if key == "" {
			errOut.printf("%s: %s unset; its listing is skipped\n", p, vars[p])
			continue
		}
		ids, err := f.Listed(ctx, p, key)
		if err != nil {
			errOut.printf("%s: listing: %v\n", p, err)
			continue
		}
		listed[p] = ids
	}
	proposals := catalogtool.Propose(static, catalogtool.Sources{Metadata: meta, Listed: listed})
	for _, p := range proposals {
		out.printf("%s\n", p)
	}
	out.printf("catalogs: %d proposal(s); nothing was changed\n", len(proposals))
	return 0
}

// readMetadata reads the document from path, or fetches it when path is "".
func readMetadata(ctx context.Context, path string, f fetcher) (catalog.Metadata, error) {
	if path == "" {
		return f.Metadata(ctx)
	}
	// The path is the maintainer's own argument.
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return catalog.Metadata{}, err
	}
	return catalog.ParseMetadata(bytes.NewReader(data))
}
