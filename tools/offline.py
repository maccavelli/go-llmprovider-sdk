#!/usr/bin/env python3
"""offline: read a ModelInfo at a version with no request, and prove the check would see one.

Defined by docs/decisions/0035-MADR-maintainer-tools-in-python.md D5 (amended
2026-10-12), from the post-tag check of
docs/decisions/0033-PLAN-public-model-metadata.md Phase 5. A scratch program
parses a metadata document with catalog.ParseMetadata, adds it with
catalog.WithModelLayer, turns models.dev off, passes no token source, and gives
an HTTP client whose transport counts each request and refuses it. It reads one
model through LookupModel and CachedModel, and requires every field the
document sets, from the layer, with no request.

After go vet, go run and a build for each platform, each plant is applied to a
fresh copy of the program, which must then fail.

Usage:
  tools/offline.py VERSION [--metadata FILE] [--provider ID] [--model ID] [--platforms LIST] [--out DIR]
  tools/offline.py --replace [--platforms host] [--out DIR]

LIST is comma-separated GOOS/GOARCH pairs, or "host"; six by default.

Exit codes: 0 every step passed and every plant was caught, 1 a step failed or
a plant was not caught, 2 tool error (a plant that does not match the program
exactly once).
"""
from __future__ import annotations

import argparse
import os
import shutil
import subprocess
import sys
from pathlib import Path

from _common import ROOT, out_dir, run
from consumer import MODULE, PLATFORMS, Steps, go_version, platforms

FIXTURE = ROOT / "llmprovider" / "catalog" / "testdata" / "metadata-0033" / "fixture.json"

PROGRAM = """\
// Command offline reads a ModelInfo with no request (0035-MADR D5).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// counting counts each request, and refuses it.
type counting struct{ n atomic.Int64 }

func (c *counting) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return nil, errors.New("offline: request refused")
}

func main() {
	metadata := flag.String("metadata", "", "a models.dev-format document")
	provider := flag.String("provider", "", "a built-in provider id")
	model := flag.String("model", "", "a model id the document holds")
	flag.Parse()
	os.Exit(run(*metadata, llmprovider.ProviderID(*provider), *model))
}

func run(path string, provider llmprovider.ProviderID, model string) int {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("FAIL:", err)
		return 1
	}
	meta, err := catalog.ParseMetadata(f)
	_ = f.Close()
	if err != nil {
		fmt.Println("FAIL: parse:", err)
		return 1
	}
	want, ok := meta.Model(provider, model)
	if !ok {
		fmt.Printf("FAIL: the document does not hold %s %s\\n", provider, model)
		return 1
	}
	ct := &counting{}
	opts := []llmprovider.Option{
		llmprovider.WithHTTPClient(&http.Client{Transport: ct}),
		llmprovider.WithoutModelMetadata(),
		catalog.WithModelLayer(meta),
	}
	ctx := context.Background()
	failures := 0
	fail := func(format string, args ...any) {
		fmt.Printf("FAIL: "+format+"\\n", args...)
		failures++
	}
	looked, err := catalog.LookupModel(ctx, provider, nil, model, opts...)
	if err != nil {
		fail("LookupModel: %v", err)
	}
	cached, ok := catalog.CachedModel(ctx, provider, nil, model, opts...)
	if !ok {
		fail("CachedModel: nothing known")
	}
	known := 0
	for fl := catalog.FieldName; fl <= catalog.FieldRetires; fl++ {
		if !want.Has(fl) {
			continue
		}
		known++
		for _, got := range []struct {
			call string
			info catalog.ModelInfo
		}{{"LookupModel", looked}, {"CachedModel", cached}} {
			if !got.info.Has(fl) {
				fail("%s: field %d unset", got.call, fl)
			} else if src := got.info.SourceOf(fl); src != meta.Name() {
				fail("%s: field %d from %q, want %q", got.call, fl, src, meta.Name())
			}
		}
	}
	if known == 0 {
		fail("the document sets no field of %s", model)
	}
	fmt.Printf("fields=%d window=%d output=%d input=%g output=%g tiers=%d source=%s\\n", known,
		looked.ContextWindow, looked.MaxOutputTokens, looked.Cost.Input, looked.Cost.Output,
		len(looked.Cost.Tiers), looked.SourceOf(catalog.FieldContextWindow))
	n := ct.n.Load()
	fmt.Printf("requests=%d\\n", n)
	if n != 0 {
		fail("the counting transport saw %d request(s), want 0", n)
	}
	if failures > 0 {
		return 1
	}
	fmt.Println("offline: ok")
	return 0
}
"""

# Each plant removes what keeps the read offline; the program must then fail.
PLANTS: dict[str, tuple[str, str]] = {
    "models.dev left on": ("\t\tllmprovider.WithoutModelMetadata(),\n", ""),
    "no caller layer": ("\t\tcatalog.WithModelLayer(meta),\n", ""),
}


def check_plants(plants: dict[str, tuple[str, str]]) -> None:
    for name, (old, _) in plants.items():
        if (n := PROGRAM.count(old)) != 1:
            raise ValueError(f'plant "{name}": "old" matches {n} time(s) in the program, want 1')


def module(d: Path, source: str, version: str | None, steps: Steps) -> None:
    """A fresh module at d holding source, requiring VERSION, or the tree when version is None."""
    if d.exists():
        shutil.rmtree(d)
    d.mkdir(parents=True)
    (d / "main.go").write_text(source, encoding="utf-8")
    (d / "go.mod").write_text(f"module example.com/offline\n\ngo {go_version()}\n", encoding="utf-8")
    if version:
        steps.run(d, "go", "get", f"{MODULE}@{version}")
    else:
        steps.run(d, "go", "mod", "edit", f"-require={MODULE}@v0.0.0-00010101000000-000000000000",
                  f"-replace={MODULE}={ROOT}")
    steps.run(d, "go", "mod", "tidy")


def check(work: Path, version: str | None, plats: list[dict[str, str] | None], flags: list[str],
          plants: dict[str, tuple[str, str]], steps: Steps) -> None:
    d = work / "offline"
    module(d, PROGRAM, version, steps)
    unformatted = run(["gofmt", "-l", "main.go"], cwd=d).stdout.strip()
    print(f"$ gofmt -l main.go: {unformatted or 'clean'}")
    if unformatted:
        steps.failed += 1
    steps.run(d, "go", "vet", ".")
    steps.run(d, "go", "run", ".", *flags)
    for env in plats:
        steps.run(d, "go", "build", "-o", os.devnull, ".", env=env)
    for name, (old, new) in plants.items():
        p = work / "offline-plant"
        module(p, PROGRAM.replace(old, new, 1), version, steps)
        print(f"--- plant: {name}")
        res = run(["go", "run", ".", *flags], cwd=p)
        print("   " + (res.stdout + res.stderr).strip()[-2000:].replace("\n", "\n   "))
        if res.returncode == 0:
            print(f"NOT CAUGHT: {name}")
            steps.failed += 1
        else:
            print(f"caught: {name}")
        shutil.rmtree(p)


def main(argv: list[str] | None = None, plants: dict[str, tuple[str, str]] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("version", nargs="?", help="the released version, such as v1.9.0")
    parser.add_argument("--replace", action="store_true", help="build against the working tree instead")
    parser.add_argument("--metadata", type=Path, default=FIXTURE, help="a models.dev-format document")
    parser.add_argument("--provider", default="claude", help="a built-in provider id")
    parser.add_argument("--model", default="claude-x", help="a model id the document holds")
    parser.add_argument("--platforms", default=PLATFORMS, help='GOOS/GOARCH pairs, or "host"')
    parser.add_argument("--out", type=Path, default=None, help="the work directory")
    args = parser.parse_args(argv)
    if bool(args.version) == args.replace:
        parser.error("give a VERSION, or --replace, not both")
    plants = PLANTS if plants is None else plants
    steps = Steps()
    try:
        check_plants(plants)
        flags = ["-metadata", str(args.metadata.resolve()), "-provider", args.provider, "-model", args.model]
        check(args.out or out_dir("offline"), args.version, platforms(args.platforms), flags, plants, steps)
    except (OSError, ValueError, subprocess.CalledProcessError) as err:
        print(f"offline: {err}", file=sys.stderr)
        return 2
    if steps.failed:
        print(f"offline: {steps.failed} step(s) failed")
        return 1
    print("offline: every step passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
