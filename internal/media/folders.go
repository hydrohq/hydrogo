package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrInvalidFolder = errors.New("invalid media folder")

type Folder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  string    `json:"parent_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type FolderRepository interface {
	CreateFolder(context.Context, database.Executor, Folder) error
	ListFolders(context.Context, database.Executor) ([]Folder, error)
}

func NewFolder(name, parentID string) (Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 128 || strings.ContainsAny(name, "/\\\x00") {
		return Folder{}, ErrInvalidFolder
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Folder{}, err
	}
	return Folder{ID: hex.EncodeToString(id[:]), Name: name, ParentID: strings.TrimSpace(parentID)}, nil
}
