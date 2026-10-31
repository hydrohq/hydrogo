package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/audit"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mcp"
)

func mcpCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) > 0 && args[0] == "config" {
		return mcpConfigCommand(args[1:], stdout, stderr)
	}

	flags := flag.NewFlagSet("mcp", flag.ContinueOnError)
	flags.SetOutput(stderr)
	values := addRuntimeFlags(flags)

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *values.configPath == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			*values.configPath = "config.yaml"
		}
	}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		if *values.configPath == "" {
			cfg.Database.Dialect = database.SQLite
			cfg.Database.DSN = "file:data/hydro.db?_busy_timeout=5000&_journal_mode=WAL"
			cfg.ContentTypesDir = "content-types"
		} else {
			return fmt.Errorf("load config: %w", err)
		}
	}
	if cfg.Database.DSN == "" {
		cfg.Database.Dialect = database.SQLite
		cfg.Database.DSN = "file:data/hydro.db?_busy_timeout=5000&_journal_mode=WAL"
	}
	if cfg.ContentTypesDir == "" {
		cfg.ContentTypesDir = "content-types"
	}
	_ = os.MkdirAll("data", 0o750)
	_ = os.MkdirAll(cfg.ContentTypesDir, 0o750)

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	colService := collections.Service{
		Repository: sqlstore.Store{
			DB:      db,
			Dialect: cfg.Database.Dialect,
		},
	}

	var loadedTypes []contenttypes.Definition
	if cfg.ContentTypesDir != "" {
		if defs, err := contenttypes.LoadDir(cfg.ContentTypesDir); err == nil {
			loadedTypes = defs
		}
	}

	syncFn := func(ctx context.Context, def contenttypes.Definition) error {
		schemas, err := contenttypes.Schemas([]contenttypes.Definition{def})
		if err != nil {
			return err
		}
		return colService.SyncSchemas(ctx, schemas)
	}

	var detailRecorder audit.DetailRecorder
	rec := newAuditRecorder(db, cfg.Database.Dialect, false)
	if dr, ok := any(rec).(audit.DetailRecorder); ok {
		detailRecorder = dr
	}

	handler := mcp.NewHandler(mcp.Config{
		CollectionService: colService,
		ContentTypesDir:   cfg.ContentTypesDir,
		ContentTypes: func() []contenttypes.Definition {
			return loadedTypes
		},
		SyncSchema: syncFn,
		AuditStore: detailRecorder,
		DB:         db,
	})

	return handler.ServeStdio(ctx, stdin, stdout)
}

func mcpConfigCommand(args []string, stdout, stderr io.Writer) error {
	binPath, err := os.Executable()
	if err != nil {
		binPath = "hydro"
	}
	binPath, _ = filepath.Abs(binPath)

	configSnippet := map[string]any{
		"mcpServers": map[string]any{
			"hydrogo": map[string]any{
				"command": binPath,
				"args":    []string{"mcp"},
			},
		},
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(configSnippet)
}
