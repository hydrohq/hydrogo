package pages

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	MaxBlocks       = 64
	MaxBlockPayload = 64 << 10
)

type Block struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

var ErrInvalidBlocks = errors.New("invalid page blocks")

func ValidateBlocks(blocks []Block) error {
	if len(blocks) > MaxBlocks {
		return fmt.Errorf("%w: too many blocks", ErrInvalidBlocks)
	}
	for _, block := range blocks {
		if strings.TrimSpace(block.Type) == "" || len(block.Payload) == 0 || len(block.Payload) > MaxBlockPayload {
			return ErrInvalidBlocks
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(block.Payload, &object); err != nil || object == nil {
			return ErrInvalidBlocks
		}
		switch block.Type {
		case "hero", "cta":
			if err := requireString(object, "title"); err != nil {
				return err
			}
		case "faq":
			var items []struct {
				Question string `json:"question"`
				Answer   string `json:"answer"`
			}
			raw, ok := object["items"]
			if !ok || json.Unmarshal(raw, &items) != nil || len(items) == 0 || len(items) > 32 {
				return ErrInvalidBlocks
			}
			for _, item := range items {
				if strings.TrimSpace(item.Question) == "" || strings.TrimSpace(item.Answer) == "" {
					return ErrInvalidBlocks
				}
			}
		default:
			return fmt.Errorf("%w: unsupported block type %q", ErrInvalidBlocks, block.Type)
		}
		if raw, ok := object["button_url"]; ok {
			var value string
			if json.Unmarshal(raw, &value) != nil || !safeBlockURL(value) {
				return ErrInvalidBlocks
			}
		}
	}
	return nil
}

func requireString(object map[string]json.RawMessage, key string) error {
	raw, ok := object[key]
	var value string
	if !ok || json.Unmarshal(raw, &value) != nil || strings.TrimSpace(value) == "" {
		return ErrInvalidBlocks
	}
	return nil
}

func safeBlockURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	u, err := url.Parse(value)
	if err != nil || strings.EqualFold(u.Scheme, "javascript") || strings.EqualFold(u.Scheme, "data") {
		return false
	}
	return u.Scheme == "" || strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")
}

func EncodeBlocksForStore(blocks []Block) (string, error) {
	if err := ValidateBlocks(blocks); err != nil {
		return "", err
	}
	if blocks == nil {
		return "[]", nil
	}
	b, err := json.Marshal(blocks)
	return string(b), err
}

func DecodeBlocksForStore(value any) ([]Block, error) {
	if value == nil {
		return nil, nil
	}
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, ErrInvalidBlocks
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	var blocks []Block
	if err := json.Unmarshal(data, &blocks); err != nil || ValidateBlocks(blocks) != nil {
		return nil, ErrInvalidBlocks
	}
	return blocks, nil
}
