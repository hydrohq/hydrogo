package adminstudio

import (
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
