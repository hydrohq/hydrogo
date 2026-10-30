package ai

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
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
	Reply      string             `json:"reply"`
	Operations []CopilotOperation `json:"operations"`
}

// CopilotActionLog records an executed Copilot turn along with pre-mutation snapshots for one-click Undo.
type CopilotActionLog struct {
	ID         string             `json:"id"`
	TenantID   string             `json:"tenant_id"`
	Prompt     string             `json:"prompt"`
	Reply      string             `json:"reply"`
	Provider   string             `json:"provider"`
	Model      string             `json:"model"`
	TokensUsed int64              `json:"tokens_used"`
	Operations []CopilotOperation `json:"operations"`
	Undone     bool               `json:"undone"`
	CreatedAt  time.Time          `json:"created_at"`
	UndoneAt   *time.Time         `json:"undone_at,omitempty"`
}

// BuildCopilotSystemPrompt constructs the system instruction for the tenant's Workspace Copilot.
func BuildCopilotSystemPrompt(tenantID string, collectionsSummary string, recordsSample string) string {
	return `Sen "` + tenantID + `" çalışma alanının (workspace) özel yapay zeka mağaza ve içerik asistanısın (HydroGo AI Copilot).
Kullanıcının doğal dildeki isteklerini analiz edip çalışma alanındaki koleksiyon kayıtlarını oluşturabilir, güncelleyebilir veya silebilirsin.
Yalnızca aşağıdaki JSON formatında yanıt ver (başka hiçbir açıklama veya markdown ekleme):
{
  "reply": "Kullanıcıya yapılan işlemi açıklayan nazik Türkçe yanıt",
  "operations": [
    {
      "op": "create" | "update" | "delete",
      "collection_id": "koleksiyon_id",
      "record_id": "kayit_id (update/delete için zorunlu, create için opsiyonel)",
      "data": { "alan": "değer" }
    }
  ]
}
Kurallar:
1. Sadece kullanıcının istediği değişiklikleri yap. Eğer kullanıcı soru soruyorsa "operations": [] döndür ve "reply" içinde yanıtla.
2. "update" işleminde mevcut kaydın tüm alanlarını koruyarak sadece değişen alanları güncelle.
3. Aktif Koleksiyon Şemaları:
` + collectionsSummary + `
4. Mevcut Kayıt Örnekleri:
` + recordsSample
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
