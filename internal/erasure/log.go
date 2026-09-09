// Package erasure stores durable, replayable deletion intents outside the application DB.
package erasure

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
)

var (
	ErrInvalidIntent = errors.New("invalid erasure intent")
	ErrCorruptLog    = errors.New("corrupt erasure log")
	ErrCheckpoint    = errors.New("erasure checkpoint does not match log")
)

type Intent struct {
	ID        string    `json:"id"`
	EditionID string    `json:"edition_id"`
	TargetID  string    `json:"target_id"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

type record struct {
	Sequence uint64 `json:"sequence"`
	Previous string `json:"previous"`
	Intent
	Hash string `json:"hash"`
}

type Checkpoint struct {
	Sequence uint64 `json:"sequence"`
	Hash     string `json:"hash"`
}

type Log struct {
	path string
	mu   sync.Mutex
}

func New(path string) (*Log, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("erasure log path is required")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	parent := filepath.Dir(path)
	if err := pathguard.RejectSymlinkComponents(parent); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return nil, err
	}
	if err := pathguard.RejectSymlinkComponents(parent); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, errors.New("erasure log path must be a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return &Log{path: path}, nil
}

func (l *Log) Append(intent Intent) (Checkpoint, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validatePath(); err != nil {
		return Checkpoint{}, err
	}
	if err := validate(intent); err != nil {
		return Checkpoint{}, err
	}
	records, err := readRecords(l.path)
	if err != nil {
		return Checkpoint{}, err
	}
	for _, existing := range records {
		if existing.ID == intent.ID {
			if existing.EditionID != intent.EditionID || existing.TargetID != intent.TargetID || existing.Kind != intent.Kind || (!intent.CreatedAt.IsZero() && !existing.CreatedAt.Equal(intent.CreatedAt)) {
				return Checkpoint{}, fmt.Errorf("intent id reused: %s", intent.ID)
			}
			return Checkpoint{Sequence: existing.Sequence, Hash: existing.Hash}, nil
		}
	}
	previous := ""
	sequence := uint64(1)
	if len(records) > 0 {
		previous = records[len(records)-1].Hash
		sequence = records[len(records)-1].Sequence + 1
	}
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = time.Now().UTC()
	}
	next := record{Sequence: sequence, Previous: previous, Intent: intent}
	next.Hash = hashRecord(next)
	line, err := json.Marshal(next)
	if err != nil {
		return Checkpoint{}, err
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Checkpoint{}, err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		_ = file.Close()
		return Checkpoint{}, err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return Checkpoint{}, err
	}
	if err := file.Close(); err != nil {
		return Checkpoint{}, err
	}
	return Checkpoint{Sequence: sequence, Hash: next.Hash}, nil
}

func (l *Log) Verify() (Checkpoint, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validatePath(); err != nil {
		return Checkpoint{}, err
	}
	records, err := readRecords(l.path)
	if err != nil {
		return Checkpoint{}, err
	}
	if len(records) == 0 {
		return Checkpoint{}, nil
	}
	last := records[len(records)-1]
	return Checkpoint{Sequence: last.Sequence, Hash: last.Hash}, nil
}

// EntriesAfter returns immutable intents after a previously applied sequence.
// A caller must verify the returned checkpoint before opening public traffic.
func (l *Log) EntriesAfter(sequence uint64) ([]Intent, Checkpoint, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validatePath(); err != nil {
		return nil, Checkpoint{}, err
	}
	records, err := readRecords(l.path)
	if err != nil {
		return nil, Checkpoint{}, err
	}
	result := make([]Intent, 0)
	for _, current := range records {
		if current.Sequence > sequence {
			result = append(result, current.Intent)
		}
	}
	if len(records) == 0 {
		return result, Checkpoint{}, nil
	}
	last := records[len(records)-1]
	return result, Checkpoint{Sequence: last.Sequence, Hash: last.Hash}, nil
}

// VerifyCheckpoint proves that a persisted restore checkpoint still anchors
// the current hash chain. A sequence number by itself is insufficient after
// log replacement or retention pruning.
func (l *Log) VerifyCheckpoint(checkpoint Checkpoint) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validatePath(); err != nil {
		return err
	}
	if checkpoint.Sequence == 0 && checkpoint.Hash == "" {
		return nil
	}
	if checkpoint.Sequence == 0 || checkpoint.Hash == "" {
		return ErrCheckpoint
	}
	records, err := readRecords(l.path)
	if err != nil {
		return err
	}
	if checkpoint.Sequence > uint64(len(records)) || records[checkpoint.Sequence-1].Hash != checkpoint.Hash {
		return ErrCheckpoint
	}
	return nil
}

// Replay verifies the complete chain, then applies every intent after the
// supplied checkpoint in order. The returned checkpoint is safe to persist
// only when every handler invocation succeeds; a failed replay returns the
// original checkpoint so the caller can retry idempotently.
func (l *Log) Replay(ctx context.Context, sequence uint64, apply func(context.Context, Intent) error) (Checkpoint, error) {
	if apply == nil {
		return Checkpoint{}, errors.New("erasure replay handler is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	intents, checkpoint, err := l.EntriesAfter(sequence)
	if err != nil {
		return Checkpoint{}, err
	}
	for _, intent := range intents {
		if err := ctx.Err(); err != nil {
			return Checkpoint{}, err
		}
		if err := apply(ctx, intent); err != nil {
			return Checkpoint{}, err
		}
	}
	return checkpoint, nil
}

// ReplayFromCheckpoint verifies the checkpoint hash before applying later
// intents. A mismatch leaves the restore isolated and prevents replaying from
// an unrelated or truncated log.
func (l *Log) ReplayFromCheckpoint(ctx context.Context, checkpoint Checkpoint, apply func(context.Context, Intent) error) (Checkpoint, error) {
	if apply == nil {
		return Checkpoint{}, errors.New("erasure replay handler is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := l.VerifyCheckpoint(checkpoint); err != nil {
		return Checkpoint{}, err
	}
	intents, latest, err := l.EntriesAfter(checkpoint.Sequence)
	if err != nil {
		return Checkpoint{}, err
	}
	for _, intent := range intents {
		if err := ctx.Err(); err != nil {
			return Checkpoint{}, err
		}
		if err := apply(ctx, intent); err != nil {
			return Checkpoint{}, err
		}
	}
	return latest, nil
}

func (l *Log) validatePath() error {
	if l == nil || strings.TrimSpace(l.path) == "" {
		return errors.New("erasure log path is required")
	}
	if err := pathguard.RejectSymlinkComponents(filepath.Dir(l.path)); err != nil {
		return err
	}
	info, err := os.Lstat(l.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("erasure log path must be a regular file")
	}
	return nil
}

func readRecords(path string) ([]record, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1<<20)
	result := make([]record, 0)
	previous := ""
	for expected := uint64(1); scanner.Scan(); expected++ {
		var current record
		if err := json.Unmarshal(scanner.Bytes(), &current); err != nil {
			return nil, ErrCorruptLog
		}
		if current.Sequence != expected || current.Previous != previous || current.Hash != hashRecord(current) || validate(current.Intent) != nil {
			return nil, ErrCorruptLog
		}
		result = append(result, current)
		previous = current.Hash
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func hashRecord(current record) string {
	copy := current
	copy.Hash = ""
	data, _ := json.Marshal(copy)
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func validate(intent Intent) error {
	if strings.TrimSpace(intent.ID) == "" || strings.TrimSpace(intent.EditionID) == "" || strings.TrimSpace(intent.TargetID) == "" || strings.TrimSpace(intent.Kind) == "" {
		return ErrInvalidIntent
	}
	return nil
}
