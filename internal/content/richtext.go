package content

import (
	"encoding/json"
	"errors"
	"strings"
)

const MaxRichTextBytes = 512 << 10

var ErrInvalidRichText = errors.New("invalid rich text")

// ValidateRichText accepts legacy plain text and the supported Quill Delta
// envelope, while bounding input before it reaches persistence.
func ValidateRichText(value string) error {
	if len([]byte(value)) > MaxRichTextBytes {
		return ErrInvalidRichText
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if !strings.HasPrefix(value, "{") {
		return nil
	}
	var envelope struct {
		Format        string          `json:"format"`
		FormatVersion int             `json:"format_version"`
		Document      json.RawMessage `json:"document"`
		Ops           json.RawMessage `json:"ops"`
	}
	if err := json.Unmarshal([]byte(value), &envelope); err != nil {
		return ErrInvalidRichText
	}
	raw := envelope.Document
	if len(raw) == 0 {
		raw = envelope.Ops
	}
	if len(raw) == 0 || (envelope.Format != "" && envelope.Format != "quill-delta") || (envelope.FormatVersion != 0 && envelope.FormatVersion != 1) {
		return ErrInvalidRichText
	}
	var ops []struct {
		Insert interface{} `json:"insert"`
	}
	if err := json.Unmarshal(raw, &ops); err != nil {
		return ErrInvalidRichText
	}
	for _, op := range ops {
		if op.Insert == nil {
			return ErrInvalidRichText
		}
	}
	return nil
}
