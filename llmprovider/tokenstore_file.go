package llmprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// maxTokenFileBytes bounds a session file on Load (0016-MADR D3).
const maxTokenFileBytes = 64 << 10

// Refresh-lock timing (0016-MADR amendment A2): a lock whose file has not
// been touched for lockStaleAfter belongs to a dead holder and is taken over;
// a live holder touches it every lockHeartbeat; a waiter gives up after
// lockWait with a retryable error.
const (
	lockStaleAfter = 30 * time.Second
	lockHeartbeat  = 5 * time.Second
	lockWait       = 25 * time.Second
)

// FileTokenStore persists one OAuth session per provider in a directory.
// Writes are durable (0016-MADR D3), and a refresh holds a lock file beside
// the session so two processes never spend the same refresh token
// (0016-MADR amendment A2).
type FileTokenStore struct {
	Dir string

	// lock timing; zero values use the package defaults. Tests shorten them.
	staleAfter, heartbeat, wait time.Duration
}

// NewFileTokenStore creates the token directory when needed.
func NewFileTokenStore(dir string) (*FileTokenStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("FileTokenStore: empty dir")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("FileTokenStore mkdir: %w", err)
	}
	return &FileTokenStore{Dir: dir}, nil
}

func (fs *FileTokenStore) path(provider ProviderID) string {
	return filepath.Join(fs.Dir, string(provider)+".json")
}

// Load reads a provider session, returning nil when no token file exists. A
// file larger than 64 KiB is refused.
func (fs *FileTokenStore) Load(ctx context.Context, provider ProviderID) (*OAuthSession, error) {
	if err := validateProviderID(provider); err != nil {
		return nil, err
	}
	data, err := readBounded(fs.path(provider), maxTokenFileBytes)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("FileTokenStore read: %w", err)
	}
	var rec fileRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("FileTokenStore decode: %w", err)
	}
	s := &OAuthSession{
		Provider:  rec.Provider,
		Access:    rec.Access,
		Refresh:   rec.Refresh,
		Expiry:    rec.Expiry,
		Issuer:    rec.Issuer,
		ClientID:  rec.ClientID,
		AccountID: rec.AccountID,
		FedRAMP:   rec.FedRAMP,
		TokenURL:  rec.TokenURL,
	}
	s.Store = fs
	return s, nil
}

// readBounded reads path, failing when it holds more than limit bytes.
func readBounded(path string, limit int64) (data []byte, err error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	data, err = io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s is larger than %d bytes", filepath.Base(path), limit)
	}
	return data, nil
}

// tokenStoreBeforeWrite and tokenStoreRename are seams for tests: the first
// observes the temp file before its first byte, the second plants a rename
// failure.
var (
	tokenStoreBeforeWrite = func(string) {}
	tokenStoreRename      = os.Rename
)

// Save durably replaces a provider session file: a temp file in the store
// directory, mode 0600 before any byte is written, then write, fsync, close,
// rename, and an fsync of the directory where the platform allows it
// (0016-MADR D3). On any failure the previous file is intact and the temp
// file is removed.
func (fs *FileTokenStore) Save(ctx context.Context, provider ProviderID, s *OAuthSession) (err error) {
	if err := validateProviderID(provider); err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("FileTokenStore Save: nil session")
	}
	rec := fileRecord{
		Provider:  provider,
		Access:    s.Access,
		Refresh:   s.Refresh,
		Expiry:    s.Expiry,
		Issuer:    s.Issuer,
		ClientID:  s.ClientID,
		AccountID: s.AccountID,
		FedRAMP:   s.FedRAMP,
		TokenURL:  s.TokenURL,
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("FileTokenStore encode: %w", err)
	}
	tmp, err := os.CreateTemp(fs.Dir, ".tok-*.json")
	if err != nil {
		return fmt.Errorf("FileTokenStore temp: %w", err)
	}
	tmpName := tmp.Name()
	closed := false
	defer func() {
		if !closed {
			err = errors.Join(err, tmp.Close())
		}
		cleanupErr := os.Remove(tmpName)
		if cleanupErr != nil && !errors.Is(cleanupErr, iofs.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("FileTokenStore cleanup: %w", cleanupErr))
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("FileTokenStore chmod: %w", err)
	}
	tokenStoreBeforeWrite(tmpName)
	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("FileTokenStore write: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("FileTokenStore sync: %w", err)
	}
	closed = true
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("FileTokenStore close: %w", err)
	}
	if err := tokenStoreRename(tmpName, fs.path(provider)); err != nil {
		return fmt.Errorf("FileTokenStore rename: %w", err)
	}
	if err := syncDir(fs.Dir); err != nil {
		return fmt.Errorf("FileTokenStore sync dir: %w", err)
	}
	return nil
}

// Delete removes a provider session and succeeds when it is already absent.
func (fs *FileTokenStore) Delete(ctx context.Context, provider ProviderID) error {
	if err := validateProviderID(provider); err != nil {
		return err
	}
	err := os.Remove(fs.path(provider))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LockRefresh takes the provider's refresh lock: a file created exclusively
// beside the session, holding the pid and time, and touched every heartbeat
// while held. A lock untouched for 30 s belongs to a dead holder and is taken
// over. A waiter gives up after 25 s with an error matching
// ErrProviderUnavailable, which callers may retry; it never refreshes
// unlocked (0016-MADR amendment A2). The returned func releases the lock.
func (fs *FileTokenStore) LockRefresh(ctx context.Context, provider ProviderID) (unlock func(), err error) {
	if err := validateProviderID(provider); err != nil {
		return nil, err
	}
	staleAfter, heartbeat, wait := fs.lockTiming()
	path := filepath.Join(fs.Dir, string(provider)+".lock")
	deadline := time.Now().Add(wait)
	for {
		created, err := createLockFile(path)
		if err == nil && created {
			return holdLock(path, heartbeat), nil
		}
		if err != nil {
			return nil, fmt.Errorf("FileTokenStore lock: %w", err)
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > staleAfter {
			// The holder is gone: take the lock over.
			if rmErr := os.Remove(path); rmErr != nil && !errors.Is(rmErr, iofs.ErrNotExist) {
				return nil, fmt.Errorf("FileTokenStore lock takeover: %w", rmErr)
			}
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%w: oauth: another process is refreshing the %s session", ErrProviderUnavailable, provider)
		}
		//nolint:gosec // G404: non-crypto jitter between lock attempts
		pause := 50*time.Millisecond + rand.N(100*time.Millisecond)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pause):
		}
	}
}

func (fs *FileTokenStore) lockTiming() (staleAfter, heartbeat, wait time.Duration) {
	staleAfter, heartbeat, wait = lockStaleAfter, lockHeartbeat, lockWait
	if fs.staleAfter > 0 {
		staleAfter = fs.staleAfter
	}
	if fs.heartbeat > 0 {
		heartbeat = fs.heartbeat
	}
	if fs.wait > 0 {
		wait = fs.wait
	}
	return staleAfter, heartbeat, wait
}

// createLockFile creates path exclusively, reporting false when it exists.
func createLockFile(path string) (bool, error) {
	f, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, iofs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	holder := strconv.Itoa(os.Getpid()) + " " + time.Now().UTC().Format(time.RFC3339Nano) + "\n"
	_, writeErr := f.WriteString(holder)
	return true, errors.Join(writeErr, f.Close())
}

// holdLock touches path every heartbeat until the returned func removes it.
func holdLock(path string, heartbeat time.Duration) func() {
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(heartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				ignoreOAuthError(os.Chtimes(path, now, now))
			}
		}
	}()
	return func() {
		close(done)
		<-stopped
		ignoreOAuthError(os.Remove(path))
	}
}
