package wizard

import (
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// procPeekConsoleInputW is kernel32's PeekConsoleInputW, which
// golang.org/x/sys does not wrap.
var procPeekConsoleInputW = windows.NewLazySystemDLL("kernel32.dll").NewProc("PeekConsoleInputW")

// inputRecord is the console's INPUT_RECORD, its union read as a
// KEY_EVENT_RECORD.
type inputRecord struct {
	eventType   uint16
	_           uint16
	keyDown     int32
	repeatCount uint16
	virtualKey  uint16
	scanCode    uint16
	char        uint16
	controlKeys uint32
}

// settleSlice is the longest one wait on the console handle lasts: the
// handle stays signalled while any event is pending, a key release among
// them, so the wait sleeps a slice between looks.
const settleSlice = 5 * time.Millisecond

// charPending reports whether records hold a key-down event that carries a
// character: a key release, a focus, mouse, menu or resize event, and a key
// with no character (a cursor key, a modifier) do not count (0087-MADR, "The
// Windows mechanism, if seen"; ST32, OR15).
func charPending(records []inputRecord) bool {
	for _, rec := range records {
		if rec.eventType == windows.KEY_EVENT && rec.keyDown != 0 && rec.char != 0 {
			return true
		}
	}
	return false
}

// pendingWithin reports whether a key-down event that carries a character
// reaches f's console input within wait: on the console input handle it
// looks at the pending events with PeekConsoleInputW, without taking them,
// and waits in slices until wait has passed. Events without a character (a
// key release, focus, mouse, a resize) do not count. A failed call, or a
// handle that is not a console's, is false and the error (0087-MADR, "The
// Windows mechanism, if seen", A3).
func pendingWithin(f *os.File, wait time.Duration) (bool, error) {
	h := windows.Handle(f.Fd())
	deadline := time.Now().Add(wait)
	var records [16]inputRecord
	for {
		var n uint32
		//nolint:gosec // G103: PeekConsoleInputW takes the records' and the count's addresses, and golang.org/x/sys does not wrap it (0087-MADR, "The Windows mechanism, if seen").
		r, _, err := procPeekConsoleInputW.Call(uintptr(h), uintptr(unsafe.Pointer(&records[0])), uintptr(len(records)), uintptr(unsafe.Pointer(&n)))
		if r == 0 {
			return false, err
		}
		if charPending(records[:n]) {
			return true, nil
		}
		left := time.Until(deadline)
		if left <= 0 {
			return false, nil
		}
		slice := min(left, settleSlice)
		ev, err := windows.WaitForSingleObject(h, uint32(slice/time.Millisecond)+1)
		if err != nil {
			return false, err
		}
		if ev == windows.WAIT_OBJECT_0 && n > 0 {
			// Only events without a character are pending, and they keep
			// the handle signalled: sleep the slice before looking again.
			time.Sleep(slice)
		}
	}
}
