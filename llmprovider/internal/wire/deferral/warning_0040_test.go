package deferral

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D7: WarningDeferralEmulated, on a request with a deferred tool
// sent as D5 or D6, or with a load appended under D3.

func TestWarning_0040(t *testing.T) {
	deferred := &llmprovider.Request{Tools: deferralTools(), Input: history()}
	searchOnly := &llmprovider.Request{Tools: deferralTools(), Input: history()[:5]}
	plain := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "a"}}}
	continued := &llmprovider.Request{Tools: deferralTools(), PreviousResponseID: "r"}
	for _, tc := range []struct {
		name   string
		req    *llmprovider.Request
		form   Form
		want   bool
		detail string
	}{
		{"nothing deferred", plain, Append, false, ""},
		{"appended", deferred, Append, true, "appended"},
		{"appended, continued", continued, Append, true, "every deferred tool"},
		{"dispatched", deferred, Dispatch, true, "use_tool"},
		{"Messages native", deferred, MessagesNative, false, ""},
		{"Responses, a search's loads only", searchOnly, ResponsesNative, false, ""},
		{"Responses, an event's load appended", deferred, ResponsesNative, true, "send_email"},
	} {
		w, ok := Warning(tc.form, tc.req, "m1")
		if ok != tc.want {
			t.Errorf("%s: warned %t, want %t", tc.name, ok, tc.want)
			continue
		}
		if ws := Warnings(tc.form, tc.req, "m1"); ok && (len(ws) != 1 || ws[0] != w) || !ok && ws != nil {
			t.Errorf("%s: Warnings %+v; want Warning's %+v, or nil", tc.name, ws, w)
		}
		if !ok {
			continue
		}
		if w.Kind != llmprovider.WarningDeferralEmulated || w.Model != "m1" || !strings.Contains(w.Detail, tc.detail) {
			t.Errorf("%s: %+v; want kind %q, model m1 and a detail containing %q", tc.name, w,
				llmprovider.WarningDeferralEmulated, tc.detail)
		}
	}
}
