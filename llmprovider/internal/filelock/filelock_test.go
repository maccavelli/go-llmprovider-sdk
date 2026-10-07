package filelock

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func open(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.OpenFile(filepath.Clean(path), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestTryLock_ExcludesAnotherHandle: two handles on one file, in one process,
// exclude each other, and Unlock and Close each release the lock.
func TestTryLock_ExcludesAnotherHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.oslock")
	a, b := open(t, path), open(t, path)
	if err := TryLock(a); err != nil {
		t.Fatalf("first TryLock: %v", err)
	}
	if err := TryLock(b); !errors.Is(err, ErrLocked) {
		t.Fatalf("second TryLock = %v, want ErrLocked", err)
	}
	if err := Unlock(a); err != nil {
		t.Fatalf("Unlock: %v", err)
	}
	if err := TryLock(b); err != nil {
		t.Fatalf("TryLock after Unlock: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := TryLock(a); err != nil {
		t.Fatalf("TryLock after the holder closed: %v", err)
	}
}

// TestTryLock_ClosedFileFails: a lock on a closed file is an error, not
// ErrLocked.
func TestTryLock_ClosedFileFails(t *testing.T) {
	f := open(t, filepath.Join(t.TempDir(), "x.oslock"))
	_ = f.Close()
	if err := TryLock(f); err == nil || errors.Is(err, ErrLocked) {
		t.Fatalf("TryLock on a closed file = %v, want an error other than ErrLocked", err)
	}
	if err := Unlock(f); err == nil {
		t.Fatal("Unlock on a closed file succeeded")
	}
}

// TestTryLock_ReleasedWhenHolderDies: a child process takes the lock and is
// killed; the operating system releases it, with nothing stale left behind.
func TestTryLock_ReleasedWhenHolderDies(t *testing.T) {
	if os.Getenv("FILELOCK_HOLD") != "" {
		f, err := os.OpenFile(os.Getenv("FILELOCK_HOLD"), os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil || TryLock(f) != nil {
			os.Exit(3)
		}
		_, _ = os.Stdout.WriteString("held\n")
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	path := filepath.Join(t.TempDir(), "x.oslock")
	cmd := exec.Command(os.Args[0], "-test.run=^TestTryLock_ReleasedWhenHolderDies$")
	cmd.Env = append(os.Environ(), "FILELOCK_HOLD="+path)
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(out).ReadString('\n'); err != nil || line != "held\n" {
		_ = cmd.Process.Kill()
		t.Fatalf("child: %q, %v; want it to hold the lock", line, err)
	}
	f := open(t, path)
	if err := TryLock(f); !errors.Is(err, ErrLocked) {
		_ = cmd.Process.Kill()
		t.Fatalf("TryLock while the child holds it = %v, want ErrLocked", err)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := TryLock(f)
		if err == nil {
			return
		}
		if !errors.Is(err, ErrLocked) || time.Now().After(deadline) {
			t.Fatalf("TryLock after the holder died = %v, want the lock", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
