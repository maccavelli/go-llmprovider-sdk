//go:build live_gateways

package llmprovider_test

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_Workspace_0064 is 0064-PLAN Phase 1's measurement. It runs only
// with LLMPROVIDER_LIVE_0064 set, with a key not scoped to a workspace in
// ANTHROPIC_API_KEY, its workspace in LLMPROVIDER_LIVE_0064_WORKSPACE and,
// when set, a well-formed workspace the key may not use in
// LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE. It sends Anthropic's GET
// /v1/models?limit=1 once per case (workspaceCases), which bills nothing, and
// writes each record to the directory LLMPROVIDER_LIVE_0064_OUT names,
// outside the module, as <case>.json, every workspace and request ID
// redacted (recordWorkspace, whose offline twin is
// TestWorkspaceRecord_Twin_0064). It logs each case's status, classes and
// whether the reply named a workspace; never a body, a workspace or the key.
func TestLive_Workspace_0064(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0064") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0064 to run 0064-PLAN Phase 1's measurement")
	}
	key, workspace := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("LLMPROVIDER_LIVE_0064_WORKSPACE")
	foreign, out := os.Getenv("LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE"), os.Getenv("LLMPROVIDER_LIVE_0064_OUT")
	switch {
	case key == "":
		t.Fatal("ANTHROPIC_API_KEY unset: set it to the key not scoped to a workspace")
	case workspace == "":
		t.Fatal("LLMPROVIDER_LIVE_0064_WORKSPACE unset: name the key's workspace, wrkspc_…")
	case out == "":
		t.Fatal("LLMPROVIDER_LIVE_0064_OUT unset: name a directory outside the module for the records")
	}
	if err := outsideModule(out); err != nil {
		t.Fatal(err)
	}
	for _, c := range workspaceCases(workspace, foreign) {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			r, err := recordWorkspace(ctx, http.DefaultClient, "https://api.anthropic.com", key, c, workspace, foreign)
			if err != nil {
				t.Fatal(err)
			}
			if err := writeWorkspaceRecord(out, r); err != nil {
				t.Fatal(err)
			}
			t.Logf("%s (sent %s): HTTP %d %s, classes %s, reply workspace header %s",
				c.name, c.sent, r.Status, r.Type, strings.Join(r.Classes, ","), r.ReplyWorkspace)
		})
	}
	if foreign == "" {
		t.Log("LLMPROVIDER_LIVE_0064_FOREIGN_WORKSPACE unset: the foreign case, E5's 404, was not sent")
	}
}

// TestLive_WorkspaceConfirm_0064 is 0064-PLAN Phase 6 step 1, run by the
// owner: with LLMPROVIDER_LIVE_0064 set, a key not scoped to a workspace in
// ANTHROPIC_API_KEY and its workspace in LLMPROVIDER_LIVE_0064_WORKSPACE, it
// runs checkWorkspaceLive against Anthropic (its offline twin is
// TestWorkspaceLive_Twin_0064): no workspace and a malformed one are refused
// with their classes, and the workspace generates, streams and lists. It
// sends four generations to workspaceLiveModel, two of them refused, each
// at most 256 tokens. It reports each problem by class and status, never
// the key, the workspace or the service's text.
func TestLive_WorkspaceConfirm_0064(t *testing.T) {
	if os.Getenv("LLMPROVIDER_LIVE_0064") == "" {
		t.Skip("set LLMPROVIDER_LIVE_0064 to run 0064-PLAN Phase 6's confirmation")
	}
	key, workspace := os.Getenv("ANTHROPIC_API_KEY"), os.Getenv("LLMPROVIDER_LIVE_0064_WORKSPACE")
	if key == "" || workspace == "" {
		t.Fatal("set ANTHROPIC_API_KEY to the key not scoped to a workspace, and LLMPROVIDER_LIVE_0064_WORKSPACE to its workspace")
	}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	problems := checkWorkspaceLive(ctx, key, workspace)
	for _, problem := range problems {
		t.Error(problem)
	}
	if len(problems) == 0 {
		t.Log("no workspace: ErrWorkspaceRequired; malformed: ErrWorkspaceNotFound; the workspace: generated, streamed and listed")
	}
}
