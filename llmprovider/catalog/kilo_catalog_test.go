package catalog

import (
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The tests of what stayed here when Kilo moved to providers/kilo (0015-PLAN
// S7): the catalog's curation, the error classification and the endpoint
// resolver. They were in kilo_test.go and kilo_data_collection_test.go.

// TestKiloPriceRank pins trap 1: "-1" is variable pricing on the kilo-auto
// tiers and must sort LAST, not before free.
func TestKiloPriceRank(t *testing.T) {
	if got := kiloPriceRank("0"); got != 0 {
		t.Errorf(`kiloPriceRank("0") = %v, want 0`, got)
	}
	if got := kiloPriceRank("0.0000004"); got != 0.0000004 {
		t.Errorf(`kiloPriceRank("0.0000004") = %v`, got)
	}
	for _, s := range []string{"-1", "", "abc"} {
		if got := kiloPriceRank(s); got != math.MaxFloat64 {
			t.Errorf("kiloPriceRank(%q) = %v, want MaxFloat64 (sorts last)", s, got)
		}
	}
	if kiloPriceRank("-1") <= kiloPriceRank("0") {
		t.Error(`"-1" (variable) must sort after "0" (free), not before`)
	}
}

// TestIsUsableKiloModel_ColonFree pins trap 2: ":free" is part of the id, not a
// policy suffix. Kilo must never use splitHuggingFaceModelPolicy.
func TestIsUsableKiloModel_ColonFree(t *testing.T) {
	for _, id := range []string{"tencent/hy3:free", "nvidia/nemotron-3.5-lightning:free", "minimax/minimax-m3:free"} {
		if !isUsableKiloModel(id) {
			t.Errorf("%q must be usable: :free is part of the id", id)
		}
	}
	if !isUsableKiloModel("kilo-auto/free") {
		t.Error("kilo-auto/free must be usable")
	}
	for _, id := range []string{"deepseek/deepseek-v4-flash-vision-exp", "mimo/mimo-v2-omni", "no-slash", ""} {
		if isUsableKiloModel(id) {
			t.Errorf("%q must be rejected", id)
		}
	}
}

func TestRankKiloModel(t *testing.T) {
	if rankKiloModel("kilo-auto/small") <= rankKiloModel("meta-llama/llama-3.1-8b-instruct") {
		t.Error("kilo-auto managed tiers should rank above concrete ids")
	}
	if rankKiloModel("kilo-auto/small") <= rankKiloModel("kilo-auto/frontier") {
		t.Error("small should rank above frontier for hook latency")
	}
}

func TestStaticKilo_Count(t *testing.T) {
	if len(staticKilo) == 0 || len(staticKilo) > MaxListed {
		t.Errorf("staticKilo has %d entries, want 1..%d", len(staticKilo), MaxListed)
	}
	for _, m := range staticKilo {
		if !isUsableKiloModel(m) {
			t.Errorf("%q fails its own usability filter", m)
		}
	}
}

// kiloDataCollectionRequired is gate G-K's response to deny on a model that
// requires collection (kilo-auto/free, 2026-09-27).
const kiloDataCollectionRequired = `{"error":"Data collection is required for this model. Please enable data collection to use this model or choose another model.","error_type":"data_collection_required","message":"Data collection is required for this model. Please enable data collection to use this model or choose another model."}`

// TestClassify_KiloDataCollectionRequired: the refusal is a terminal
// ErrNotPermitted carrying Kilo's error_type and message.
func TestClassify_KiloDataCollectionRequired(t *testing.T) {
	err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderKilo), &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(kiloDataCollectionRequired)),
	})
	var apiErr *llmprovider.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %T %v, want *APIError", err, err)
	}
	if !errors.Is(err, llmprovider.ErrNotPermitted) || apiErr.Retryable() {
		t.Errorf("err = %v (retryable %t), want a non-retryable ErrNotPermitted", err, apiErr.Retryable())
	}
	if apiErr.Code != "data_collection_required" || !strings.HasPrefix(apiErr.Message, "Data collection is required") {
		t.Errorf("Code = %q, Message = %q", apiErr.Code, apiErr.Message)
	}
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("err = %v, want it to still match the 400 status sentinel ErrInvalidRequest", err)
	}
}
