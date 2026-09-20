package extensionpkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

var ErrInvalidStateRecord = errors.New("invalid extension lifecycle record")

type Record struct {
	ID                    string                   `json:"id"`
	Version               string                   `json:"version"`
	State                 extensionv1.PackageState `json:"state"`
	Artifact              string                   `json:"artifact"`
	RequestedCapabilities []string                 `json:"requested_capabilities,omitempty"`
	GrantedCapabilities   []string                 `json:"granted_capabilities,omitempty"`
	UpdatedAt             time.Time                `json:"updated_at"`
}

type Store struct{ Root string }

func (s Store) recordPath(id string) (string, error) {
	if strings.TrimSpace(s.Root) == "" || id == "" || filepath.Base(id) != id {
		return "", ErrInvalidStateRecord
	}
	return filepath.Join(s.Root, id, "state.json"), nil
}

func (s Store) Load(id string) (Record, error) {
	name, err := s.recordPath(id)
	if err != nil {
		return Record{}, err
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return Record{}, err
	}
	var record Record
	if err := json.Unmarshal(data, &record); err != nil || record.ID != id || record.State == "" {
		return Record{}, ErrInvalidStateRecord
	}
	return record, nil
}

func (s Store) Save(record Record) error {
	name, err := s.recordPath(record.ID)
	if err != nil || record.Version == "" || record.Artifact == "" || record.State == "" {
		return ErrInvalidStateRecord
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), ".state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, name); err != nil {
		return fmt.Errorf("replace lifecycle state: %w", err)
	}
	return nil
}

func (s Store) Transition(record Record, next extensionv1.PackageState) (Record, error) {
	if err := extensionv1.Transition(record.State, next); err != nil {
		return Record{}, err
	}
	record.State = next
	record.UpdatedAt = time.Now().UTC()
	if err := s.Save(record); err != nil {
		return Record{}, err
	}
	return record, nil
}
