package v1

import (
	"context"
	"fmt"
	"strings"
)

// LifecycleRequest carries release-owned identity to an optional lifecycle
// hook. Hooks are opt-in; a module that does not need one implements nothing.
type LifecycleRequest struct {
	Module      string
	FromVersion string
	ToVersion   string
	DryRun      bool
}

// MigrationPlan is returned by a preflight hook. IDs are descriptive and are
// executed by Core; extensions never run DDL directly from an admin request.
type MigrationPlan struct {
	MigrationIDs    []string
	Destructive     bool
	RequiresRestart bool
}

type InstallHook interface {
	Install(context.Context, LifecycleRequest) error
}

type EnableHook interface {
	Enable(context.Context, LifecycleRequest) error
}

type DisableHook interface {
	Disable(context.Context, LifecycleRequest) error
}

type UpgradeHook interface {
	Upgrade(context.Context, LifecycleRequest) error
}

type MigrationPreflightHook interface {
	PreflightMigration(context.Context, LifecycleRequest) (MigrationPlan, error)
}

func ValidateLifecycleRequest(request LifecycleRequest) error {
	if strings.TrimSpace(request.Module) == "" {
		return fmt.Errorf("lifecycle module is required")
	}
	if strings.TrimSpace(request.ToVersion) == "" {
		return fmt.Errorf("lifecycle target version is required")
	}
	if request.FromVersion == request.ToVersion && request.FromVersion != "" {
		return fmt.Errorf("lifecycle source and target versions are identical")
	}
	return nil
}

func ValidateMigrationPlan(plan MigrationPlan) error {
	seen := make(map[string]struct{}, len(plan.MigrationIDs))
	for _, rawID := range plan.MigrationIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			return fmt.Errorf("migration plan contains an empty ID")
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("migration plan repeats %q", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}
