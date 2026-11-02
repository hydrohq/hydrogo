package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/ai/prompts"
	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/modules/site"
)

const copilotActionsSettingKey = "ai.copilot.actions"
const maxStoredCopilotActions = 50

type copilotRequestPayload struct {
	Prompt string `json:"prompt"`
}

func (a *serveApp) currentWorkspaceTenantID(r *http.Request) string {
	if scope, err := tenant.FromContext(r.Context()); err == nil && strings.TrimSpace(scope.TenantID) != "" {
		return strings.TrimSpace(scope.TenantID)
	}
	return "default"
}

func (a *serveApp) adminAICopilotHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var payload copilotRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid JSON payload", nil)
		return
	}
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		api.WriteError(w, r, http.StatusBadRequest, "prompt is required", nil)
		return
	}

	tenantID := a.currentWorkspaceTenantID(r)
	dialect := string(a.cfg.Database.Dialect)

	var gov tenant.Governance
	if a.db != nil {
		loaded, err := tenant.GetGovernance(r.Context(), a.db, dialect, tenantID)
		if err != nil && !errors.Is(err, tenant.ErrGovernanceNotFound) {
			api.WriteError(w, r, http.StatusInternalServerError, "failed to load workspace AI governance", nil)
			return
		}
		gov = loaded
		if !gov.AIEnabled {
			api.WriteError(w, r, http.StatusForbidden, "AI Copilot is disabled for this workspace", nil)
			return
		}
		if _, err := tenant.ConsumeAITokens(r.Context(), a.db, dialect, tenantID, 0); err != nil {
			if errors.Is(err, tenant.ErrAIQuotaExceeded) {
				api.WriteError(w, r, http.StatusTooManyRequests, "monthly AI token quota exceeded for this workspace", nil)
				return
			}
			if errors.Is(err, tenant.ErrAIDisabled) {
				api.WriteError(w, r, http.StatusForbidden, "AI Copilot is disabled for this workspace", nil)
				return
			}
			api.WriteError(w, r, http.StatusInternalServerError, "failed to check the workspace AI quota", nil)
			return
		}
	} else {
		gov = tenant.DefaultGovernance(tenantID)
	}

	defs := a.contentTypesForContext(r.Context())
	schemasSnapshot, recordsSnapshot := a.buildCopilotWorkspaceContext(r.Context(), defs)

	plan, tokensUsed, err := a.generateWorkspaceCopilotPlan(r.Context(), gov, prompt, defs, schemasSnapshot, recordsSnapshot)
	if err != nil {
		api.WriteError(w, r, http.StatusBadGateway, fmt.Sprintf("AI Copilot generation failed: %v", err), nil)
		return
	}
	if err := validateCopilotOperations(plan.Operations, defs); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if a.db != nil && tokensUsed > 0 {
		updatedGov, consumeErr := tenant.ConsumeAITokens(r.Context(), a.db, dialect, tenantID, tokensUsed)
		if consumeErr != nil {
			if errors.Is(consumeErr, tenant.ErrAIQuotaExceeded) {
				api.WriteError(w, r, http.StatusTooManyRequests, "monthly AI token quota exceeded for this workspace; nothing was changed", nil)
				return
			}
			api.WriteError(w, r, http.StatusInternalServerError, "failed to record AI token usage; nothing was changed", nil)
			return
		}
		gov = updatedGov
	}

	executedOps, execErr := a.executeCopilotOperations(r.Context(), plan.Operations)
	if execErr != nil {
		api.WriteError(w, r, http.StatusBadRequest, execErr.Error(), nil)
		return
	}

	pMeta, _ := prompts.Load("copilot_system")
	var pID, pVer, pHash string
	if pMeta != nil {
		pID = pMeta.ID
		pVer = pMeta.Version
		pHash = pMeta.Hash
	}
	plan.PromptID = pID
	plan.PromptVersion = pVer
	plan.PromptHash = pHash

	actionLog := aicore.CopilotActionLog{
		ID:            newCopilotActionID(),
		TenantID:      tenantID,
		Prompt:        prompt,
		Reply:         plan.Reply,
		Operations:    executedOps,
		Provider:      gov.AIProvider,
		Model:         gov.AIModel,
		TokensUsed:    tokensUsed,
		Undone:        false,
		PromptID:      pID,
		PromptVersion: pVer,
		PromptHash:    pHash,
		CreatedAt:     time.Now().UTC(),
	}
	a.prependCopilotActionLog(r.Context(), tenantID, actionLog)

	api.WriteData(w, http.StatusOK, map[string]any{
		"reply":      plan.Reply,
		"action_log": actionLog,
		"governance": gov,
	}, nil)
}

func (a *serveApp) adminAIActionsListHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	tenantID := a.currentWorkspaceTenantID(r)
	dialect := string(a.cfg.Database.Dialect)
	gov := tenant.DefaultGovernance(tenantID)
	if a.db != nil {
		if loaded, err := tenant.GetGovernance(r.Context(), a.db, dialect, tenantID); err == nil {
			gov = loaded
		}
	}
	logs := a.loadCopilotActionLogs(r.Context(), tenantID)
	api.WriteData(w, http.StatusOK, map[string]any{
		"actions":    logs,
		"governance": gov,
	}, nil)
}

func (a *serveApp) adminAIActionUndoHandler(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		api.WriteError(w, r, http.StatusUnauthorized, "unauthorized", nil)
		return
	}
	actionID := strings.TrimSpace(r.PathValue("id"))
	if actionID == "" {
		api.WriteError(w, r, http.StatusBadRequest, "action id is required", nil)
		return
	}
	tenantID := a.currentWorkspaceTenantID(r)
	logs := a.loadCopilotActionLogs(r.Context(), tenantID)
	targetIdx := -1
	for i := range logs {
		if logs[i].ID == actionID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		api.WriteError(w, r, http.StatusNotFound, "copilot action not found", nil)
		return
	}
	if logs[targetIdx].Undone {
		api.WriteError(w, r, http.StatusBadRequest, "copilot action has already been undone", nil)
		return
	}

	if err := a.undoCopilotOperations(r.Context(), logs[targetIdx].Operations); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, fmt.Sprintf("failed to undo copilot action: %v", err), nil)
		return
	}

	now := time.Now().UTC()
	logs[targetIdx].Undone = true
	logs[targetIdx].UndoneAt = &now
	a.saveCopilotActionLogs(r.Context(), tenantID, logs)

	api.WriteData(w, http.StatusOK, map[string]any{
		"action_log": logs[targetIdx],
	}, nil)
}

func (a *serveApp) buildCopilotWorkspaceContext(ctx context.Context, defs []contenttypes.Definition) (string, string) {
	type fieldDesc struct {
		ID       string `json:"id"`
		Kind     string `json:"kind"`
		Required bool   `json:"required,omitempty"`
	}
	type schemaDesc struct {
		ID     string      `json:"id"`
		Label  string      `json:"label"`
		Fields []fieldDesc `json:"fields"`
	}
	schemas := make([]schemaDesc, 0, len(defs))

	for _, def := range defs {
		fields := make([]fieldDesc, 0, len(def.Fields))
		for _, f := range def.Fields {
			fields = append(fields, fieldDesc{
				ID:       f.ID,
				Kind:     string(f.Kind),
				Required: f.Required,
			})
		}
		schemas = append(schemas, schemaDesc{
			ID:     def.ID,
			Label:  def.Label,
			Fields: fields,
		})
	}

	schemasJSON, _ := json.Marshal(schemas)
	// Live customer records, order notes, and phone numbers are excluded from the prompt
	// to protect customer privacy and prevent indirect prompt injection.
	return string(schemasJSON), "{}"
}

func (a *serveApp) generateWorkspaceCopilotPlan(
	ctx context.Context,
	gov tenant.Governance,
	prompt string,
	defs []contenttypes.Definition,
	schemasJSON string,
	recordsJSON string,
) (aicore.CopilotPlan, int64, error) {
	apiKey := ""
	if gov.HasCustomAPIKey && strings.TrimSpace(gov.AIEncryptedAPIKey) != "" {
		decrypted, err := aicore.DecryptAPIKey(gov.AIEncryptedAPIKey, a.aiMasterSecret())
		if err != nil {
			return aicore.CopilotPlan{}, 0, fmt.Errorf("failed to decrypt workspace AI API key: %w", err)
		}
		apiKey = strings.TrimSpace(decrypted)
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("HYDRO_AI_API_KEY"))
	}

	provider := strings.ToLower(strings.TrimSpace(gov.AIProvider))
	if provider == "" {
		provider = "gemini"
	}

	if apiKey != "" || provider == "ollama" {
		adapter := aicore.NewAdapter(aicore.AdapterConfig{
			Provider: provider,
			Model:    gov.AIModel,
			APIKey:   apiKey,
			Endpoint: strings.TrimSpace(os.Getenv("HYDRO_AI_BASE_URL")),
		})
		sysPrompt := aicore.BuildCopilotSystemPrompt(gov.DisplayName, schemasJSON, recordsJSON)
		resp, err := adapter.Generate(ctx, aicore.ChatRequest{
			SystemPrompt: sysPrompt,
			Messages: []aicore.ChatMessage{
				{Role: "user", Content: prompt},
			},
			JSONMode: true,
		})
		if err != nil {
			return aicore.CopilotPlan{}, 0, err
		}
		plan, err := aicore.ParseCopilotPlan(resp.Content)
		if err != nil {
			return aicore.CopilotPlan{}, resp.TokensUsed, err
		}
		tokens := resp.TokensUsed
		if tokens <= 0 {
			tokens = int64((len(sysPrompt) + len(prompt) + len(resp.Content)) / 4)
		}
		return plan, tokens, nil
	}

	// Local deterministic planner when no external LLM key is configured.
	plan := a.localHeuristicCopilotPlan(ctx, prompt, defs)
	tokens := int64((len(prompt) + len(plan.Reply)) / 3)
	if tokens < 25 {
		tokens = 25
	}
	return plan, tokens, nil
}

func (a *serveApp) localHeuristicCopilotPlan(ctx context.Context, prompt string, defs []contenttypes.Definition) aicore.CopilotPlan {
	// An explicit JSON plan passed as the prompt is a dev/test affordance only:
	// in production the prompt must never be executable input, or any
	// authenticated caller could write arbitrary records through it.
	if a.devMode && strings.HasPrefix(strings.TrimSpace(prompt), "{") {
		if plan, err := aicore.ParseCopilotPlan(prompt); err == nil && plan.Reply != "" {
			return plan
		}
	}

	lower := strings.ToLower(prompt)
	var ops []aicore.CopilotOperation

	// Check if there is a store_settings collection and the user asks for theme/color/whatsapp changes.
	for _, def := range defs {
		if def.ID == "store_settings" && a.collectionService.Repository != nil {
			recs, _ := a.collectionService.QueryRecords(ctx, "store_settings", collections.Query{Limit: 1})
			if len(recs) > 0 {
				rec := recs[0]
				updates := map[string]any{}
				if strings.Contains(lower, "koyu") || strings.Contains(lower, "dark") || strings.Contains(lower, "siyah") {
					updates["bg_tone"] = "#121212"
					updates["primary_color"] = "#F9F6F0"
				}
				if strings.Contains(lower, "altın") || strings.Contains(lower, "gold") {
					updates["accent_color"] = "#D4AF37"
				}
				if len(updates) > 0 {
					ops = append(ops, aicore.CopilotOperation{
						Op:           "update",
						CollectionID: "store_settings",
						RecordID:     rec.ID,
						Data:         updates,
					})
					return aicore.CopilotPlan{
						Reply:      "Mağaza görünüm ve tema ayarlarınız isteğinize göre güncellendi. Beğenmezseniz tek tıkla geri alabilirsiniz.",
						Operations: ops,
					}
				}
			}
		}
	}

	return aicore.CopilotPlan{
		Reply:      "İsteğiniz alındı ve çalışma alanı asistanı tarafından değerlendirildi.",
		Operations: ops,
	}
}

// validateCopilotOperations rejects plans that reference unknown collections or
// set fields outside the target schema, so a manipulated model response cannot
// write arbitrary columns.
func validateCopilotOperations(ops []aicore.CopilotOperation, defs []contenttypes.Definition) error {
	if len(ops) == 0 {
		return nil
	}
	known := make(map[string]map[string]bool, len(defs))
	for _, def := range defs {
		fields := make(map[string]bool, len(def.Fields))
		for _, f := range def.Fields {
			fields[f.ID] = true
		}
		known[def.ID] = fields
	}
	for _, op := range ops {
		colID := strings.TrimSpace(op.CollectionID)
		fields, ok := known[colID]
		if !ok {
			return fmt.Errorf("copilot operation targets unknown collection %q", colID)
		}
		for key := range op.Data {
			if !fields[key] {
				return fmt.Errorf("copilot operation tries to set unknown field %q on %s", key, colID)
			}
		}
	}
	return nil
}

func (a *serveApp) executeCopilotOperations(ctx context.Context, ops []aicore.CopilotOperation) ([]aicore.CopilotOperation, error) {
	if len(ops) == 0 || a.collectionService.Repository == nil {
		return ops, nil
	}
	executed := make([]aicore.CopilotOperation, 0, len(ops))
	for _, op := range ops {
		action := strings.ToLower(strings.TrimSpace(op.Op))
		colID := strings.TrimSpace(op.CollectionID)
		recID := strings.TrimSpace(op.RecordID)
		if colID == "" {
			return nil, errors.New("copilot operation is missing collection_id")
		}

		switch action {
		case "create":
			if op.Data == nil {
				op.Data = map[string]any{}
			}
			created, err := a.collectionService.CreateRecord(ctx, colID, op.Data)
			if err != nil {
				return nil, fmt.Errorf("create in %s failed: %w", colID, err)
			}
			op.Op = "create"
			op.RecordID = created.ID
			op.Data = cloneMap(created.Data)
			executed = append(executed, op)

		case "update":
			if recID == "" {
				return nil, fmt.Errorf("update in %s requires record_id", colID)
			}
			existing, err := a.collectionService.GetRecord(ctx, colID, recID)
			if err != nil {
				return nil, fmt.Errorf("record %s/%s not found: %w", colID, recID, err)
			}
			op.BeforeData = cloneMap(existing.Data)
			op.Existed = true
			merged := cloneMap(existing.Data)
			for k, v := range op.Data {
				merged[k] = v
			}
			updated, err := a.collectionService.UpdateRecord(ctx, colID, recID, merged)
			if err != nil {
				return nil, fmt.Errorf("update %s/%s failed: %w", colID, recID, err)
			}
			op.Op = "update"
			op.Data = cloneMap(updated.Data)
			executed = append(executed, op)

		case "delete":
			if recID == "" {
				return nil, fmt.Errorf("delete in %s requires record_id", colID)
			}
			existing, err := a.collectionService.GetRecord(ctx, colID, recID)
			if err != nil {
				return nil, fmt.Errorf("record %s/%s not found: %w", colID, recID, err)
			}
			op.BeforeData = cloneMap(existing.Data)
			op.Existed = true
			if err := a.collectionService.DeleteRecord(ctx, colID, recID); err != nil {
				return nil, fmt.Errorf("delete %s/%s failed: %w", colID, recID, err)
			}
			op.Op = "delete"
			executed = append(executed, op)

		default:
			return nil, fmt.Errorf("unsupported copilot operation %q", op.Op)
		}
	}
	return executed, nil
}

func (a *serveApp) undoCopilotOperations(ctx context.Context, ops []aicore.CopilotOperation) error {
	if len(ops) == 0 || a.collectionService.Repository == nil {
		return nil
	}
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.Op {
		case "create":
			if op.RecordID != "" {
				if err := a.collectionService.DeleteRecord(ctx, op.CollectionID, op.RecordID); err != nil {
					return err
				}
			}
		case "update":
			if op.RecordID != "" && op.BeforeData != nil {
				if _, err := a.collectionService.UpdateRecord(ctx, op.CollectionID, op.RecordID, op.BeforeData); err != nil {
					return err
				}
			}
		case "delete":
			if op.BeforeData != nil {
				if _, err := a.collectionService.Repository.SaveRecord(ctx, op.CollectionID, op.RecordID, op.BeforeData, nil, 0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *serveApp) loadCopilotActionLogs(ctx context.Context, tenantID string) []aicore.CopilotActionLog {
	if a.db != nil {
		if setting, found, err := a.siteStore.GetSetting(ctx, a.db, copilotActionsSettingKey); err == nil && found && strings.TrimSpace(setting.Value) != "" {
			var logs []aicore.CopilotActionLog
			if jsonErr := json.Unmarshal([]byte(setting.Value), &logs); jsonErr == nil {
				return logs
			}
		}
	}
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	if a.copilotLogs == nil {
		return []aicore.CopilotActionLog{}
	}
	out := make([]aicore.CopilotActionLog, len(a.copilotLogs[tenantID]))
	copy(out, a.copilotLogs[tenantID])
	return out
}

func (a *serveApp) saveCopilotActionLogs(ctx context.Context, tenantID string, logs []aicore.CopilotActionLog) {
	if len(logs) > maxStoredCopilotActions {
		logs = logs[:maxStoredCopilotActions]
	}
	a.stateMu.Lock()
	if a.copilotLogs == nil {
		a.copilotLogs = make(map[string][]aicore.CopilotActionLog)
	}
	cp := make([]aicore.CopilotActionLog, len(logs))
	copy(cp, logs)
	a.copilotLogs[tenantID] = cp
	a.stateMu.Unlock()

	if a.db != nil {
		if raw, err := json.Marshal(logs); err == nil {
			_ = a.siteStore.UpsertSetting(ctx, a.db, site.Setting{
				Key:       copilotActionsSettingKey,
				Value:     string(raw),
				Public:    false,
				UpdatedAt: time.Now().UTC(),
			})
		}
	}
}

func (a *serveApp) prependCopilotActionLog(ctx context.Context, tenantID string, entry aicore.CopilotActionLog) {
	existing := a.loadCopilotActionLogs(ctx, tenantID)
	updated := append([]aicore.CopilotActionLog{entry}, existing...)
	a.saveCopilotActionLogs(ctx, tenantID, updated)
}

func newCopilotActionID() string {
	var raw [8]byte
	_, _ = rand.Read(raw[:])
	return "act_" + hex.EncodeToString(raw[:])
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		out := make(map[string]any, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out
	}
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
