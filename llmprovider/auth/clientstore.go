package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ClientRecord is the OAuth client OpenAI issued an app at its first ChatGPT
// plan sign-in, and the host id it was registered with: what a later sign-in
// reuses so that a sign-out, or a switch to another OpenAI session, does not
// register another client under the user's account (0075-MADR D7). It is a
// value, safe for concurrent use while no goroutine changes it. The zero
// record is none.
type ClientRecord struct {
	ClientID string
	HostID   string
}

// ClientStore keeps a ClientRecord apart from the session, so that it outlives
// it. A TokenStore may also implement it, as it may implement RefreshLocker;
// FileTokenStore does, in <provider>.client beside the session. An
// implementation must be safe for concurrent use, as a TokenStore must, and
// must not delete the record when the session is deleted.
type ClientStore interface {
	// LoadClient returns the provider's record, the zero record when there is
	// none. A record that exists but cannot be decoded is told from every
	// other failure by the error: wrap encoding/json's *json.SyntaxError or
	// *json.UnmarshalTypeError, as FileTokenStore does (errors.As finds it
	// through any wrapping; no text is compared). A sign-in replaces such a
	// record with the client it registers, saying so once in the wizard and
	// not at all through x/signin. Any other error, a permission refused, an
	// input or output error, a file too large, is a read error: the sign-in
	// fails with it, before a client is registered, and the record is not
	// replaced (0082-MADR, fourth amendment of 2026-11-18, R-12). Naming the
	// file in a read error helps the person; showing its content does not.
	LoadClient(ctx context.Context, provider llmprovider.ProviderID) (ClientRecord, error)
	// SaveClient keeps the provider's record; saving the zero record removes
	// it.
	SaveClient(ctx context.Context, provider llmprovider.ProviderID, c ClientRecord) error
}

var _ ClientStore = (*FileTokenStore)(nil)

// clientRecordFile is the record's JSON form.
type clientRecordFile struct {
	ClientID string `json:"client_id,omitempty"`
	HostID   string `json:"host_id,omitempty"`
}

func (fs *FileTokenStore) clientPath(provider llmprovider.ProviderID) string {
	return filepath.Join(fs.Dir, string(provider)+".client")
}

// LoadClient returns the provider's client record from <provider>.client, the
// zero record when the file does not exist. A provider id that is not a plain
// file name gives an error matching llmprovider.ErrInvalidProvider. A file that
// is not a record gives an error that wraps encoding/json's, which a sign-in
// replaces; one that cannot be read, or is larger than 64 KiB, a plain error
// that names the file, which fails the sign-in and leaves the file as it is
// (0082-MADR, fourth amendment, R-12). It is safe for concurrent use.
func (fs *FileTokenStore) LoadClient(_ context.Context, provider llmprovider.ProviderID) (ClientRecord, error) {
	if err := validateProviderID(provider); err != nil {
		return ClientRecord{}, err
	}
	data, err := readBounded(fs.clientPath(provider), maxTokenFileBytes)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return ClientRecord{}, nil
		}
		return ClientRecord{}, fmt.Errorf("FileTokenStore client read: %w", err)
	}
	var rec clientRecordFile
	if err := json.Unmarshal(data, &rec); err != nil {
		return ClientRecord{}, fmt.Errorf("FileTokenStore client decode: %w", err)
	}
	return ClientRecord(rec), nil
}

// SaveClient durably writes the provider's client record to <provider>.client
// as Save writes a session: a temp file restricted to the current user before
// its first byte, then fsync and rename. Saving the zero record removes the
// file, and succeeds when it is already absent. Delete of the session leaves
// the file. A provider id that is not a plain file name gives an error
// matching llmprovider.ErrInvalidProvider, and a write or removal that fails
// a plain error. It is safe for concurrent use.
func (fs *FileTokenStore) SaveClient(_ context.Context, provider llmprovider.ProviderID, c ClientRecord) error {
	if err := validateProviderID(provider); err != nil {
		return err
	}
	if c == (ClientRecord{}) {
		if err := os.Remove(fs.clientPath(provider)); err != nil && !errors.Is(err, iofs.ErrNotExist) {
			return fmt.Errorf("FileTokenStore client remove: %w", err)
		}
		return nil
	}
	data, err := json.MarshalIndent(clientRecordFile(c), "", "  ")
	if err != nil {
		return fmt.Errorf("FileTokenStore client encode: %w", err)
	}
	return fs.writeDurable(fs.clientPath(provider), data)
}
