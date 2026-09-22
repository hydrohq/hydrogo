package adminstudio

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// DefaultBulkLimit is the maximum number of records a single Admin Studio
// bulk request may select. Modules may choose a smaller limit for their own
// capacity profile, but must not raise it implicitly at the transport layer.
const DefaultBulkLimit = 500

var (
	ErrInvalidBulkRequest = errors.New("invalid bulk request")
	bulkActionPattern     = regexp.MustCompile(`^[a-z][a-z0-9_.:-]{1,63}$`)
)

// BulkRequest is a provider-neutral selection contract. It contains no
// authorization or mutation semantics; the owning module validates the action
// and executes it inside its own policy and transaction boundary.
type BulkRequest struct {
	Action string   `json:"action"`
	IDs    []string `json:"ids"`
}

// BulkItemResult is the public, deliberately non-sensitive outcome for one
// selected record. Modules may map their own domain error to Code; raw error
// strings never cross this boundary.
type BulkItemResult struct {
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Code    string `json:"code,omitempty"`
}

// BulkExecutionResult is deterministic because IDs are normalized before the
// executor is called. Completed is false only when the context is cancelled
// before all selected IDs have been attempted.
type BulkExecutionResult struct {
	Action    string           `json:"action"`
	Results   []BulkItemResult `json:"results"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Completed bool             `json:"completed"`
}

// ExecuteBulk provides the bounded orchestration primitive for module-owned
// mutations. The callback must perform authorization, idempotency, auditing,
// and transaction handling; this function is intentionally sequential so the
// caller can supply one transaction per item or a module-specific batch
// transaction without hidden concurrency or resource amplification.
func ExecuteBulk(ctx context.Context, request BulkRequest, limit int, execute func(context.Context, string) error) (BulkExecutionResult, error) {
	if execute == nil {
		return BulkExecutionResult{}, ErrInvalidBulkRequest
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 || limit > DefaultBulkLimit {
		return BulkExecutionResult{}, ErrInvalidBulkRequest
	}
	normalized, err := NormalizeBulkIDs(request.IDs, limit)
	if err != nil {
		return BulkExecutionResult{}, err
	}
	action := strings.TrimSpace(request.Action)
	if !bulkActionPattern.MatchString(action) {
		return BulkExecutionResult{}, ErrInvalidBulkRequest
	}
	result := BulkExecutionResult{Action: action, Results: make([]BulkItemResult, 0, len(normalized))}
	for _, id := range normalized {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		item := BulkItemResult{ID: id}
		if err := execute(ctx, id); err != nil {
			item.Code = "failed"
			result.Failed++
		} else {
			item.Success = true
			result.Succeeded++
		}
		result.Results = append(result.Results, item)
	}
	result.Completed = true
	return result, nil
}

// NormalizeBulkIDs trims and deterministically sorts record IDs while
// rejecting empty, duplicate, malformed, or unbounded input.
func NormalizeBulkIDs(ids []string, limit int) ([]string, error) {
	if limit <= 0 || limit > DefaultBulkLimit || len(ids) == 0 || len(ids) > limit {
		return nil, ErrInvalidBulkRequest
	}
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, raw := range ids {
		for _, r := range raw {
			if r < 0x20 && r != '\t' || r == 0x7f {
				return nil, ErrInvalidBulkRequest
			}
		}
		id := strings.TrimSpace(raw)
		if id == "" || !utf8.ValidString(id) || utf8.RuneCountInString(id) > 128 {
			return nil, ErrInvalidBulkRequest
		}
		for _, r := range id {
			if r < 0x20 || r == 0x7f {
				return nil, ErrInvalidBulkRequest
			}
		}
		if _, ok := seen[id]; ok {
			return nil, ErrInvalidBulkRequest
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

// ValidateBulkRequest validates the shared envelope and applies the default
// bound. It intentionally does not decide whether the action is permitted.
func ValidateBulkRequest(request BulkRequest) (BulkRequest, error) {
	action := strings.TrimSpace(request.Action)
	if !bulkActionPattern.MatchString(action) {
		return BulkRequest{}, ErrInvalidBulkRequest
	}
	ids, err := NormalizeBulkIDs(request.IDs, DefaultBulkLimit)
	if err != nil {
		return BulkRequest{}, err
	}
	return BulkRequest{Action: action, IDs: ids}, nil
}
