package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/webhooks"
)

const defaultHookTimeout = 5 * time.Second

// syncHookCaller sends content-type hook events to the services configured
// under `hooks:`. Each call is one signed request (the webhook signature
// scheme, topic hook.<type>.<on>) with the hook's timeout and no redirects;
// the caller decides what a failure means. In safe mode no hook is reachable.
func syncHookCaller(cfg config.Config) contenttypes.HookCaller {
	endpoints := map[string]config.SyncHook{}
	if !cfg.SafeMode {
		for _, hook := range cfg.Hooks {
			endpoints[hook.ID] = hook
		}
	}
	return func(ctx context.Context, service string, event contenttypes.HookEvent) (int, string, error) {
		hook, ok := endpoints[service]
		if !ok {
			return 0, "", fmt.Errorf("hook %q is not configured", service)
		}
		timeout := defaultHookTimeout
		if parsed, err := time.ParseDuration(hook.Timeout); err == nil && parsed > 0 {
			timeout = parsed
		}
		topic := "hook." + event.Type + "." + event.On
		id := make([]byte, 16)
		_, _ = rand.Read(id)
		dispatcher := webhooks.Dispatcher{
			Client: &http.Client{
				Timeout:       timeout,
				CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			},
			// Sync hooks are configured in the same config file as the
			// database DSN and other trusted settings (unlike the generic
			// external-webhooks feature, which is why that path keeps the
			// internal-address rejection) -- a hook pointing at a sidecar
			// service on the same host (e.g. a local ledger) is the
			// expected, intended use, not an SSRF.
			AllowInternalAddresses: true,
		}
		endpoint := webhooks.Endpoint{ID: hook.ID, URL: hook.URL, Secret: hook.Secret, Topics: map[string]struct{}{topic: {}}}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		result, err := dispatcher.Test(ctx, endpoint, webhooks.Event{ID: hex.EncodeToString(id), Topic: topic, AggregateID: event.ID, PayloadVersion: 1, Data: event})
		if result.Status == 0 {
			return 0, "", err
		}
		var answer struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal([]byte(result.Response), &answer)
		message := strings.TrimSpace(answer.Message)
		if len([]rune(message)) > 200 {
			message = string([]rune(message)[:200])
		}
		return result.Status, message, nil
	}
}

// checkHookServices fails startup when a content type names a hook service
// that the config does not define.
func checkHookServices(configured []config.SyncHook, definitions []contenttypes.Definition) error {
	known := map[string]bool{}
	for _, hook := range configured {
		known[hook.ID] = true
	}
	for _, definition := range definitions {
		for _, hook := range definition.Hooks {
			if !known[hook.Service] {
				return fmt.Errorf("content type %q uses hook service %q, which is not defined under hooks: in the config", definition.ID, hook.Service)
			}
		}
	}
	return nil
}
