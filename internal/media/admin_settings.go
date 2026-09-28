package media

import (
	"context"
	"sync"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type AdminSettingsCache struct {
	mu     sync.Mutex
	values map[string]AdminSettings
}

func NewAdminSettingsCache() *AdminSettingsCache {
	return &AdminSettingsCache{values: make(map[string]AdminSettings)}
}

func settingsCacheKey(ctx context.Context, required bool) (string, error) {
	if !required {
		return "", nil
	}
	scope, err := tenant.Require(ctx)
	if err != nil {
		return "", err
	}
	return scope.TenantID, nil
}

func (c *AdminSettingsCache) Load(ctx context.Context, required bool, defaults AdminSettings, repository AdminSettingsRepository, db database.Executor) (AdminSettings, error) {
	if c == nil || repository == nil {
		return defaults, nil
	}
	if defaults.WatermarkEnabled && defaults.WatermarkOpacity == 0 {
		defaults.WatermarkOpacity = .6
	}
	if defaults.WatermarkPosition == "" {
		defaults.WatermarkPosition = "bottom-right"
	}
	key, err := settingsCacheKey(ctx, required)
	if err != nil {
		return AdminSettings{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.values == nil {
		c.values = make(map[string]AdminSettings)
	}
	if value, ok := c.values[key]; ok {
		return value, nil
	}
	value, found, err := repository.LoadAdminSettings(ctx, db)
	if err != nil {
		return AdminSettings{}, err
	}
	if !found {
		value = defaults
	}
	c.values[key] = value
	return value, nil
}

func (c *AdminSettingsCache) Store(ctx context.Context, required bool, value AdminSettings) error {
	if c == nil {
		return nil
	}
	key, err := settingsCacheKey(ctx, required)
	if err != nil {
		return err
	}
	c.mu.Lock()
	if c.values == nil {
		c.values = make(map[string]AdminSettings)
	}
	c.values[key] = value
	c.mu.Unlock()
	return nil
}
