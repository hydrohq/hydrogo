package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/ai/prompts"
)

// CopilotOperation represents a single collection record mutation planned by the Workspace Copilot.
type CopilotOperation struct {
	Op           string         `json:"op"`
	CollectionID string         `json:"collection_id"`
	RecordID     string         `json:"record_id,omitempty"`
	Data         map[string]any `json:"data,omitempty"`
	BeforeData   map[string]any `json:"before_data,omitempty"`
	Existed      bool           `json:"existed,omitempty"`
}

// CopilotPlan is the structured JSON response produced by the AI adapter for a workspace instruction.
type CopilotPlan struct {
	Reply         string             `json:"reply"`
	Operations    []CopilotOperation `json:"operations"`
	PromptID      string             `json:"prompt_id,omitempty"`
	PromptVersion string             `json:"prompt_version,omitempty"`
	PromptHash    string             `json:"prompt_hash,omitempty"`
}

// CopilotActionLog records an executed Copilot turn along with pre-mutation snapshots for one-click Undo.
type CopilotActionLog struct {
	ID            string             `json:"id"`
	TenantID      string             `json:"tenant_id"`
	Prompt        string             `json:"prompt"`
	Reply         string             `json:"reply"`
	Provider      string             `json:"provider"`
	Model         string             `json:"model"`
	TokensUsed    int64              `json:"tokens_used"`
	Operations    []CopilotOperation `json:"operations"`
	Undone        bool               `json:"undone"`
	PromptID      string             `json:"prompt_id,omitempty"`
	PromptVersion string             `json:"prompt_version,omitempty"`
	PromptHash    string             `json:"prompt_hash,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UndoneAt      *time.Time         `json:"undone_at,omitempty"`
}

// BuildCopilotSystemPrompt constructs the system instruction for the tenant's Workspace Copilot from the prompt registry.
func BuildCopilotSystemPrompt(tenantID string, collectionsSummary string, recordsSample string) string {
	return prompts.MustRender("copilot_system", map[string]string{
		"TENANT_ID":           tenantID,
		"COLLECTIONS_SUMMARY": collectionsSummary,
		"RECORDS_SAMPLE":      recordsSample,
	})
}

// ParseCopilotPlan extracts and validates a CopilotPlan from raw model output.
func ParseCopilotPlan(raw string) (CopilotPlan, error) {
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.IndexByte(cleaned, '\n'); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
		cleaned = strings.TrimSuffix(strings.TrimSpace(cleaned), "```")
		cleaned = strings.TrimSpace(cleaned)
	}
	var plan CopilotPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return CopilotPlan{}, errors.New("invalid copilot JSON plan")
	}
	for i := range plan.Operations {
		plan.Operations[i].Op = strings.ToLower(strings.TrimSpace(plan.Operations[i].Op))
		plan.Operations[i].CollectionID = strings.TrimSpace(plan.Operations[i].CollectionID)
		plan.Operations[i].RecordID = strings.TrimSpace(plan.Operations[i].RecordID)
	}
	return plan, nil
}
