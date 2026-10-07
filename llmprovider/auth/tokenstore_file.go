package auth

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
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/filelock"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/ownerperm"
)

// maxTokenFileBytes bounds a session file on Load (0016-MADR D3).
const maxTokenFileBytes = 64 << 10

// lockWait is how long a refresh waits for another holder's lock before it
// gives up with a retryable error (0016-MADR amendment A2).
const lockWait = 40 * time.Second

// FileTokenStore persists one OAuth session per provider in a directory.
// Writes are durable (0016-MADR D3), and a refresh holds an OS file lock
// beside the session so two processes never spend the same refresh token
// (0016-MADR amendment A2; 0026-MADR F13).
type FileTokenStore struct {
	Dir string

	// wait bounds LockRefresh; zero is lockWait. Tests shorten it.
	wait time.Duration
}

// NewFileTokenStore creates the token directory when needed, private to the
// current user: mode 0700 on Unix; on Windows a protected DACL whose one
// entry, the user's, the directory's files inherit (0010-MADR D10). On Unix
// an existing directory must be the user's and not a symlink, and loses any
// group or other write bit (0021-MADR T15).
func NewFileTokenStore(dir string) (*FileTokenStore, error) {
	if dir == "" {
		return nil, fmt.Errorf("FileTokenStore: empty dir")
	}
	if err := tokenStoreMkdir(dir); err != nil {
		return nil, fmt.Errorf("FileTokenStore mkdir: %w", err)
	}
	return &FileTokenStore{Dir: dir}, nil
}

func (fs *FileTokenStore) path(provider llmprovider.ProviderID) string {
	return filepath.Join(fs.Dir, string(provider)+".json")
}

// Load reads a provider session, returning nil when no token file exists. A
// file larger than 64 KiB is refused.
func (fs *FileTokenStore) Load(ctx context.Context, provider llmprovider.ProviderID) (*OAuthSession, error) {
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
	s.storedRefresh = rec.Refresh // what this store holds (0026-MADR F1)
	s.Store = fs
	return s, nil
}

// readBounded reads path, failing when it holds more than limit bytes. It
// opens the file shared, so a save's rename is never refused while it reads
// (0026-MADR F42).
func readBounded(path string, limit int64) (data []byte, err error) {
	f, err := openShared(path)
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
// failure. tokenStoreMkdir and tokenStoreRestrict keep the directory and each
// temp file private to the current user: mode 0o700 and 0o600 on Unix, a
// protected DACL on Windows (0010-MADR D10); tests observe them.
var (
	tokenStoreBeforeWrite = func(string) {}
	tokenStoreRename      = renameInDir
	tokenStoreMkdir       = ownerperm.MkdirAll
	tokenStoreRestrict    = ownerperm.File
)

// renameInDir renames oldPath to newPath, both in one directory, through an
// os.Root on it. On Windows that is a POSIX-semantics rename, which replaces a
// file another process holds open for reading, as long as the reader shares
// delete access, as openShared does; os.Rename is refused there with "Access
// is denied" (0026-MADR F42, deviation D3). On Unix it is renameat.
func renameInDir(oldPath, newPath string) error {
	root, err := os.OpenRoot(filepath.Dir(newPath))
	if err != nil {
		return err
	}
	defer func() { ignoreOAuthError(root.Close()) }()
	return root.Rename(filepath.Base(oldPath), filepath.Base(newPath))
}

// Save durably replaces a provider session file: a temp file in the store
// directory, restricted to the current user before any byte is written (mode
// 0600 on Unix, a protected DACL on Windows; 0010-MADR D10), then write,
// fsync, close,
// rename, and an fsync of the directory where the platform allows it
// (0016-MADR D3). On any failure the previous file is intact and the temp
// file is removed.
func (fs *FileTokenStore) Save(ctx context.Context, provider llmprovider.ProviderID, s *OAuthSession) (err error) {
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
	if err := tokenStoreRestrict(tmp); err != nil {
		return fmt.Errorf("FileTokenStore restrict: %w", err)
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
func (fs *FileTokenStore) Delete(ctx context.Context, provider llmprovider.ProviderID) error {
	if err := validateProviderID(provider); err != nil {
		return err
	}
	err := os.Remove(fs.path(provider))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LockRefresh takes the provider's refresh lock: an exclusive OS file lock,
// flock on Unix and LockFileEx on Windows, on <provider>.oslock beside the
// session. The operating system releases it when its holder unlocks, closes
// or dies, so nothing stale is left to judge and nothing is taken over. The
// file is never deleted, so no process can lock an unlinked one (0026-MADR
// F13, Q1 a and its amendment of 2026-10-06). A waiter gives up after 40 s
// with an error matching ErrProviderUnavailable, which callers may retry; it
// never refreshes unlocked (0016-MADR amendment A2). A context already done
// tries once and does not wait. The returned func releases the lock.
func (fs *FileTokenStore) LockRefresh(ctx context.Context, provider llmprovider.ProviderID) (unlock func(), err error) {
	if err := validateProviderID(provider); err != nil {
		return nil, err
	}
	wait := lockWait
	if fs.wait > 0 {
		wait = fs.wait
	}
	path := filepath.Join(fs.Dir, string(provider)+".oslock")
	f, err := os.OpenFile(filepath.Clean(path), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: FileTokenStore lock: %w", llmprovider.ErrProviderUnavailable, err)
	}
	release := func() { ignoreOAuthError(f.Close()) }
	if err := tokenStoreRestrict(f); err != nil {
		release()
		return nil, fmt.Errorf("%w: FileTokenStore lock: %w", llmprovider.ErrProviderUnavailable, err)
	}
	deadline := time.Now().Add(wait)
	for {
		err := filelock.TryLock(f)
		if err == nil {
			return func() {
				ignoreOAuthError(filelock.Unlock(f))
				release()
			}, nil
		}
		if !errors.Is(err, filelock.ErrLocked) {
			release()
			return nil, fmt.Errorf("%w: FileTokenStore lock: %w", llmprovider.ErrProviderUnavailable, err)
		}
		if ctx.Err() != nil {
			release()
			return nil, fmt.Errorf("%w: oauth: another process is refreshing the %s session: %w",
				llmprovider.ErrProviderUnavailable, provider, ctx.Err())
		}
		if time.Now().After(deadline) {
			release()
			return nil, fmt.Errorf("%w: oauth: another process is refreshing the %s session", llmprovider.ErrProviderUnavailable, provider)
		}
		//nolint:gosec // G404: non-crypto jitter between lock attempts
		pause := 50*time.Millisecond + rand.N(100*time.Millisecond)
		select {
		case <-ctx.Done():
			release()
			return nil, ctx.Err()
		case <-time.After(pause):
		}
	}
}
