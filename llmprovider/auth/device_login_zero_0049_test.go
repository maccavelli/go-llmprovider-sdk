package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDeviceLogin_ZeroWaitIsAnError_0049 (0049-MADR D1): Wait on a DeviceLogin
// StartDeviceOAuth did not return gives ErrInvalidRequest, each time, never a
// panic; its Cancel still returns.
func TestDeviceLogin_ZeroWaitIsAnError_0049(t *testing.T) {
	var d DeviceLogin
	for i := range 2 {
		var err error
		func() {
			defer func() {
				if v := recover(); v != nil {
					t.Fatalf("Wait %d on a zero DeviceLogin panicked: %v; want an error", i+1, v)
				}
			}()
			_, err = d.Wait(context.Background())
		}()
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("Wait %d on a zero DeviceLogin = %v; want ErrInvalidRequest", i+1, err)
		}
	}
	d.Cancel()
	if _, err := d.Wait(context.Background()); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("Wait after Cancel on a zero DeviceLogin = %v; want ErrInvalidRequest", err)
	}
}
