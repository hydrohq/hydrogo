package prompts

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed files/*
var embeddedFS embed.FS

// Prompt represents a versioned, hash-verified prompt instruction.
type Prompt struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body"`
	Hash        string `json:"hash"`
	Source      string `json:"source"`
}

var (
	cacheMu sync.RWMutex
	cache   = map[string]*Prompt{}
)

// ErrNotFound is returned when a prompt cannot be found.
var ErrNotFound = errors.New("prompt not found")

// Load retrieves a prompt by its identifier.
// It checks the directory specified by HYDRO_PROMPTS_DIR first (if set).
// If not found or if the override directory is not configured, it loads from embedded assets.
func Load(id string) (*Prompt, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrNotFound
	}

	overrideDir := os.Getenv("HYDRO_PROMPTS_DIR")
	if overrideDir != "" {
		if p, err := loadFromDir(overrideDir, id); err == nil && p != nil {
			return p, nil
		}
	}

	return loadEmbedded(id)
}

// MustLoad retrieves a prompt by ID or panics if not found.
func MustLoad(id string) *Prompt {
	p, err := Load(id)
	if err != nil {
		panic(fmt.Sprintf("prompts.MustLoad: %v", err))
	}
	return p
}

// Render replaces {KEY} and {{KEY}} placeholders in the prompt body with corresponding values.
func (p *Prompt) Render(vars map[string]string) string {
	res := p.Body
	for k, v := range vars {
		res = strings.ReplaceAll(res, "{{"+k+"}}", v)
		res = strings.ReplaceAll(res, "{"+k+"}", v)
	}
	return res
}

// Render loads the prompt by ID and replaces variables.
func Render(id string, vars map[string]string) (string, *Prompt, error) {
	p, err := Load(id)
	if err != nil {
		return "", nil, err
	}
	return p.Render(vars), p, nil
}

// MustRender loads the prompt by ID, replaces variables, and returns the rendered string.
func MustRender(id string, vars map[string]string) string {
	p := MustLoad(id)
	return p.Render(vars)
}

// List returns all available embedded and override prompts sorted by ID.
func List() ([]*Prompt, error) {
	seen := map[string]*Prompt{}

	// Embedded
	entries, err := fs.ReadDir(embeddedFS, "files")
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			base := e.Name()
			id := strings.TrimSuffix(base, filepath.Ext(base))
			if p, err := loadEmbedded(id); err == nil {
				seen[p.ID] = p
			}
		}
	}

	// Override dir
	overrideDir := os.Getenv("HYDRO_PROMPTS_DIR")
	if overrideDir != "" {
		if dirEntries, err := os.ReadDir(overrideDir); err == nil {
			for _, e := range dirEntries {
				if e.IsDir() {
					continue
				}
				base := e.Name()
				id := strings.TrimSuffix(base, filepath.Ext(base))
				if p, err := loadFromDir(overrideDir, id); err == nil {
					seen[p.ID] = p
				}
			}
		}
	}

	result := make([]*Prompt, 0, len(seen))
	for _, p := range seen {
		result = append(result, p)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func parsePrompt(raw []byte, defaultID, source string) (*Prompt, error) {
	text := string(raw)
	lines := strings.Split(text, "\n")

	p := &Prompt{
		ID:      defaultID,
		Version: "1.0.0",
		Source:  source,
	}

	// Check if first line is JSON
	trimmedFirst := strings.TrimSpace(lines[0])
	bodyStartIndex := 0

	if strings.HasPrefix(trimmedFirst, "{") && strings.HasSuffix(trimmedFirst, "}") {
		var header struct {
			ID          string `json:"id"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal([]byte(trimmedFirst), &header); err == nil {
			if header.ID != "" {
				p.ID = header.ID
			}
			if header.Version != "" {
				p.Version = header.Version
			}
			p.Description = header.Description
			bodyStartIndex = 1
		}
	} else {
		// Key-Value header parsing (e.g. ID: ..., VERSION: ...)
		inHeader := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || trimmed == "---" {
				if inHeader || i > 0 {
					bodyStartIndex = i + 1
					break
				}
			}
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				k := strings.ToUpper(strings.TrimSpace(parts[0]))
				v := strings.TrimSpace(parts[1])
				switch k {
				case "ID":
					p.ID = v
					inHeader = true
				case "VERSION":
					p.Version = v
					inHeader = true
				case "DESCRIPTION":
					p.Description = v
					inHeader = true
				default:
					if inHeader {
						// unrecognized header key
					} else {
						// not a header line
						bodyStartIndex = 0
						goto parseBody
					}
				}
			} else {
				if inHeader {
					bodyStartIndex = i
					break
				}
				bodyStartIndex = 0
				break
			}
		}
	}

parseBody:
	body := strings.Join(lines[bodyStartIndex:], "\n")
	p.Body = strings.TrimSpace(body)

	sum := sha256.Sum256([]byte(p.Body))
	p.Hash = hex.EncodeToString(sum[:])

	return p, nil
}

func loadFromDir(dir, id string) (*Prompt, error) {
	candidates := []string{
		filepath.Join(dir, id+".prompt"),
		filepath.Join(dir, id+".json"),
		filepath.Join(dir, id),
	}
	for _, cand := range candidates {
		data, err := os.ReadFile(cand)
		if err == nil {
			return parsePrompt(data, id, cand)
		}
	}
	return nil, ErrNotFound
}

func loadEmbedded(id string) (*Prompt, error) {
	candidates := []string{
		"files/" + id + ".prompt",
		"files/" + id + ".json",
		"files/" + id,
		id + ".prompt",
	}
	for _, cand := range candidates {
		data, err := embeddedFS.ReadFile(cand)
		if err == nil {
			return parsePrompt(data, id, "embedded:"+cand)
		}
	}
	return nil, ErrNotFound
}
