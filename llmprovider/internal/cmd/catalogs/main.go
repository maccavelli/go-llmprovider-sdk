// Command catalogs proposes the next curated model lists, for a person to
// review (0033-MADR D4, and its amendment "the second refresh, and what the
// command reads"). It reads the models.dev document and the providers'
// listings, and prints internal/catalogtool's proposals, one per line. It
// edits no list, and prints no key.
//
// The live guard's four public listings (OpenCode Zen, OpenCode Go, Kilo and
// Hugging Face) are read with no key, so a workspace's settings cannot hide a
// model; the others are read with their provider's key, when it is set. A
// ranked list is read with the utility profile and metadata, and its six is
// proposed as a diff.
//
// Usage:
//
//	go run ./llmprovider/internal/cmd/catalogs [-doc FILE] [-snapshot DIR]
//
// -doc reads a models.dev-format document from FILE instead of fetching one.
// -snapshot writes DIR, which must not exist: each ranked list's listing body
// as it was read, the document's sections the SDK reads (Metadata.Encode), and
// snapshot.json, with the time of the run and each file's SHA-256.
// Exit codes: 0 the proposals were printed, 2 a tool error.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/catalogtool"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, KeysFromEnv(), &liveFetcher{})) }

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

// publicListing are the providers whose listing is read with no key: the
// four the live guard reads with none (llmprovider/live_catalog_0033_test.go's
// guard*Models constants), which this set must match. A keyed Zen listing
// leaves out what the key's workspace disables (0033-MADR's amendment "the
// second refresh, and what the command reads", F1, D4a).
var publicListing = map[llmprovider.ProviderID]bool{
	llmprovider.ProviderOpencodeZen: true, llmprovider.ProviderOpencodeGo: true,
	llmprovider.ProviderKilo: true, llmprovider.ProviderHuggingFace: true,
}

// snapshotFiles names each ranked list's listing in a snapshot, as the
// 2026-09-26 snapshot named them (llmprovider/catalog/testdata).
var snapshotFiles = map[llmprovider.ProviderID]string{
	llmprovider.ProviderOpencodeZen: "zen.json", llmprovider.ProviderOpencodeGo: "go.json",
	llmprovider.ProviderKilo: "kilo.json", llmprovider.ProviderHuggingFace: "hf.json",
	llmprovider.ProviderTogether: "together.json",
}

// fetcher reads the sources: the document, and one provider's catalog, with
// the listing body it was built from.
type fetcher interface {
	Metadata(ctx context.Context) (catalog.Metadata, error)
	// Catalog lists provider with key, "" for a public listing. A ranked list
	// is ranked with the utility profile and metadata.
	Catalog(ctx context.Context, provider llmprovider.ProviderID, key string, ranked bool) (catalog.Catalog, error)
	// Listing is the listing body provider's catalog was built from.
	Listing(provider llmprovider.ProviderID) ([]byte, bool)
}

// liveFetcher reads the sources from the network, and keeps each listing body.
// Its zero value reads the real endpoints; a test points it at its own.
type liveFetcher struct {
	metadataURL string               // "" is models.dev's default
	extra       []llmprovider.Option // appended to each catalog.List
	mu          sync.Mutex
	listings    map[llmprovider.ProviderID][]byte
}

func (f *liveFetcher) Metadata(ctx context.Context) (catalog.Metadata, error) {
	return catalog.LookupMetadata(ctx, f.metadataURL, nil)
}

func (f *liveFetcher) Catalog(ctx context.Context, provider llmprovider.ProviderID, key string, ranked bool) (catalog.Catalog, error) {
	rec := newListingRecorder()
	opts := append([]llmprovider.Option{llmprovider.WithHTTPClient(&http.Client{Transport: rec})}, f.extra...)
	if ranked {
		opts = append(opts, catalog.WithProfile(catalog.ProfileUtility))
	} else {
		opts = append(opts, llmprovider.WithoutModelMetadata())
	}
	cat, err := catalog.List(ctx, provider, llmprovider.NewStaticToken(key), opts...)
	if err == nil {
		err = cat.Err
	}
	if err != nil {
		return cat, err
	}
	body, err := rec.listing()
	if err != nil {
		return cat, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listings == nil {
		f.listings = map[llmprovider.ProviderID][]byte{}
	}
	f.listings[provider] = body
	return cat, nil
}

func (f *liveFetcher) Listing(provider llmprovider.ProviderID) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, ok := f.listings[provider]
	return body, ok
}

// listingRecorder keeps the body of each GET whose path ends in /models: a
// provider's listing, not the models.dev document. It sends each request
// through the SDK's bounded default transport, never net/http's shared one
// (R30).
type listingRecorder struct {
	base   http.RoundTripper
	mu     sync.Mutex
	bodies [][]byte
}

// newListingRecorder is a recorder over the SDK's default transport.
func newListingRecorder() *listingRecorder {
	return &listingRecorder{base: transport.DefaultClient().Transport}
}

func (r *listingRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
	if err != nil || req.Method != http.MethodGet || !strings.HasSuffix(req.URL.Path, "/models") {
		return resp, err
	}
	body, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.bodies = append(r.bodies, body)
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// listing is the one listing body recorded; none, or more than one, is an
// error, since a snapshot must hold exactly what was ranked.
func (r *listingRecorder) listing() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.bodies) != 1 {
		return nil, fmt.Errorf("recorded %d listing(s), want 1", len(r.bodies))
	}
	return r.bodies[0], nil
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
	snapshot := flags.String("snapshot", "", "write the ranked listings and the document to this new directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *snapshot != "" {
		if _, err := os.Stat(*snapshot); !errors.Is(err, os.ErrNotExist) {
			errOut.printf("catalogs: -snapshot %s: the directory exists, or cannot be checked (%v)\n", *snapshot, err)
			return 2
		}
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
	ranked := map[llmprovider.ProviderID][]string{}
	for _, p := range providers {
		if ids := catalog.Static(p); len(ids) > 0 {
			static[p] = ids
		}
		key := keys[p]
		switch {
		case publicListing[p]:
			key = ""
		case key == "":
			errOut.printf("%s: %s unset; its listing is skipped\n", p, vars[p])
			continue
		}
		cat, err := f.Catalog(ctx, p, key, catalogtool.Ranked(p))
		if err != nil {
			errOut.printf("%s: listing: %v\n", p, err)
			continue
		}
		listed[p] = cat.Usable
		if catalogtool.Ranked(p) {
			ranked[p] = cat.Recommended[:min(len(cat.Recommended), catalog.MaxListed)]
		}
	}
	proposals := catalogtool.Propose(static, catalogtool.Sources{Metadata: meta, Listed: listed, Ranked: ranked})
	for _, p := range proposals {
		out.printf("%s\n", p)
	}
	out.printf("catalogs: %d proposal(s); nothing was changed\n", len(proposals))
	if *snapshot != "" {
		if err := writeSnapshot(*snapshot, meta, f, time.Now().UTC()); err != nil {
			errOut.printf("catalogs: -snapshot: %v\n", err)
			return 2
		}
		out.printf("snapshot: %s\n", *snapshot)
	}
	return 0
}

// writeSnapshot writes dir: each ranked list's listing body, the document's
// sections the SDK reads, and snapshot.json. It writes nothing unless every
// ranked listing was read, and builds the directory beside dir before renaming
// it into place, so a failed write leaves no snapshot.
func writeSnapshot(dir string, meta catalog.Metadata, f fetcher, now time.Time) error {
	files := map[string][]byte{}
	for p, name := range snapshotFiles {
		body, ok := f.Listing(p)
		if !ok {
			return fmt.Errorf("%s's listing was not read; a snapshot needs every ranked list", p)
		}
		files[name] = body
	}
	var doc bytes.Buffer
	if err := meta.Encode(&doc); err != nil {
		return err
	}
	files["metadata.json"] = doc.Bytes()
	manifest := struct {
		RecordedAt string            `json:"recorded_at"`
		Files      map[string]string `json:"files"`
	}{RecordedAt: now.Format(time.RFC3339), Files: map[string]string{}}
	for name, body := range files {
		sum := sha256.Sum256(body)
		manifest.Files[name] = hex.EncodeToString(sum[:])
	}
	m, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	files["snapshot.json"] = append(m, '\n')
	tmp := dir + ".partial"
	if err := os.Mkdir(tmp, 0o750); err != nil {
		return err
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tmp, name), body, 0o600); err != nil {
			return errors.Join(err, os.RemoveAll(tmp))
		}
	}
	if err := os.Rename(tmp, dir); err != nil {
		return errors.Join(err, os.RemoveAll(tmp))
	}
	return nil
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
