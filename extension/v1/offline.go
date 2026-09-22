package v1

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	MaxOfflineCommands = 500
	MaxOfflinePayload  = 1 << 20
	MaxSyncCursorBytes = 256
)

var offlineCommandID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,127}$`)

type OfflineConflictStrategy string

const (
	ConflictReject OfflineConflictStrategy = "reject"
	ConflictMerge  OfflineConflictStrategy = "merge"
	ConflictManual OfflineConflictStrategy = "manual"
)

type OfflineCommand struct {
	ID             string
	Type           string
	Payload        []byte
	BaseVersion    int64
	IdempotencyKey string
}

type OfflineSyncBatch struct {
	Cursor   string
	Commands []OfflineCommand
}

func ValidateOfflineSyncBatch(batch OfflineSyncBatch) (OfflineSyncBatch, error) {
	batch.Cursor = strings.TrimSpace(batch.Cursor)
	if len(batch.Cursor) > MaxSyncCursorBytes || len(batch.Commands) > MaxOfflineCommands {
		return OfflineSyncBatch{}, errors.New("offline sync budget exceeded")
	}
	seen := make(map[string]struct{}, len(batch.Commands))
	for i := range batch.Commands {
		command := &batch.Commands[i]
		command.ID, command.Type, command.IdempotencyKey = strings.TrimSpace(command.ID), strings.TrimSpace(command.Type), strings.TrimSpace(command.IdempotencyKey)
		if !offlineCommandID.MatchString(command.ID) || !offlineCommandID.MatchString(command.Type) || !offlineCommandID.MatchString(command.IdempotencyKey) {
			return OfflineSyncBatch{}, fmt.Errorf("invalid offline command identity")
		}
		if command.BaseVersion < 0 || len(command.Payload) > MaxOfflinePayload {
			return OfflineSyncBatch{}, fmt.Errorf("invalid offline command %q", command.ID)
		}
		if _, ok := seen[command.ID]; ok {
			return OfflineSyncBatch{}, fmt.Errorf("duplicate offline command %q", command.ID)
		}
		seen[command.ID] = struct{}{}
	}
	sort.SliceStable(batch.Commands, func(i, j int) bool { return batch.Commands[i].ID < batch.Commands[j].ID })
	return batch, nil
}
