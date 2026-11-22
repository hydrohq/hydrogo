package config

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Extension lets an optional add-on module own one top-level section of the config file.
//
// Core knows nothing about what a section contains. A module registers its section from an
// init function; Load then hands it the section's YAML, its HYDRO_* environment overrides,
// validation and production secret checks. Without the module the section name is unknown and
// Load rejects it like any other unknown key.
type Extension struct {
	// Section is the top-level YAML key the extension owns.
	Section string
	// Defaults returns a pointer to the section's value with its default settings applied.
	Defaults func() any
	// Env applies HYDRO_* environment overrides to the value returned by Defaults.
	Env func(lookup LookupEnv, value any) error
	// Validate checks the final value. cfg is the whole configuration, for cross-checks.
	Validate func(cfg Config, value any) error
	// ProductionIssues lists secrets or settings that must not reach production as they are.
	ProductionIssues func(cfg Config, lookup LookupEnv, value any) []string
}

var (
	extensionsMu sync.RWMutex
	extensions   []Extension
)

// RegisterExtension registers (or replaces) the extension owning e.Section.
func RegisterExtension(e Extension) {
	if strings.TrimSpace(e.Section) == "" || e.Defaults == nil {
		panic("config: extension needs a section name and Defaults")
	}
	extensionsMu.Lock()
	defer extensionsMu.Unlock()
	for i := range extensions {
		if extensions[i].Section == e.Section {
			extensions[i] = e
			return
		}
	}
	extensions = append(extensions, e)
	sort.Slice(extensions, func(i, j int) bool { return extensions[i].Section < extensions[j].Section })
}

// valueOf returns the extension's value in c, or a fresh default when c was built without Load.
func (e Extension) valueOf(c Config) any {
	if v, ok := c.Extensions[e.Section]; ok && v != nil {
		return v
	}
	return e.Defaults()
}

func registeredExtensions() []Extension {
	extensionsMu.RLock()
	defer extensionsMu.RUnlock()
	return append([]Extension(nil), extensions...)
}

// ExtensionValue returns the decoded value of an extension section as the pointer type its
// Defaults function returned, or false when the extension is not built in.
func ExtensionValue[T any](c Config, section string) (T, bool) {
	var zero T
	v, ok := c.Extensions[section]
	if !ok {
		return zero, false
	}
	typed, ok := v.(T)
	return typed, ok
}

// DecodeSection decodes an extension's YAML section into v, rejecting unknown keys.
func DecodeSection(node *yaml.Node, v any) error {
	raw, err := yaml.Marshal(node)
	if err != nil {
		return err
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	return dec.Decode(v)
}

// IsInsecureSecret reports whether a secret is empty or still a CHANGE_ME placeholder.
func IsInsecureSecret(val string) bool {
	v := strings.TrimSpace(val)
	return v == "" || strings.EqualFold(v, "CHANGE_ME") || strings.EqualFold(v, "CHANGEME")
}

func extensionDefaults() map[string]any {
	exts := registeredExtensions()
	if len(exts) == 0 {
		return nil
	}
	out := make(map[string]any, len(exts))
	for _, e := range exts {
		out[e.Section] = e.Defaults()
	}
	return out
}

// applyExtensionSections decodes the sections owned by registered extensions and rejects any
// other key core did not recognise, with the same wording the strict decoder uses.
func applyExtensionSections(c *Config) error {
	owned := map[string]Extension{}
	for _, e := range registeredExtensions() {
		owned[e.Section] = e
	}
	var unknown []string
	keys := make([]string, 0, len(c.ExtensionSections))
	for k := range c.ExtensionSections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		node := c.ExtensionSections[key]
		ext, ok := owned[key]
		if !ok {
			unknown = append(unknown, fmt.Sprintf("line %d: field %s not found in type config.Config", node.Line, key))
			continue
		}
		if c.Extensions == nil {
			c.Extensions = map[string]any{}
		}
		value := ext.valueOf(*c)
		c.Extensions[ext.Section] = value
		if err := DecodeSection(&node, value); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
	}
	c.ExtensionSections = nil
	if len(unknown) > 0 {
		return fmt.Errorf("yaml: unmarshal errors:\n  %s", strings.Join(unknown, "\n  "))
	}
	return nil
}

func applyExtensionEnv(c *Config, lookup LookupEnv) error {
	for _, e := range registeredExtensions() {
		if e.Env == nil {
			continue
		}
		if err := e.Env(lookup, e.valueOf(*c)); err != nil {
			return err
		}
	}
	return nil
}

func validateExtensions(c Config) error {
	for _, e := range registeredExtensions() {
		if e.Validate == nil {
			continue
		}
		if err := e.Validate(c, e.valueOf(c)); err != nil {
			return err
		}
	}
	return nil
}

// ExtensionProductionIssues collects the production secret problems of every registered extension.
func ExtensionProductionIssues(c Config, lookup LookupEnv) []string {
	var issues []string
	for _, e := range registeredExtensions() {
		if e.ProductionIssues == nil {
			continue
		}
		issues = append(issues, e.ProductionIssues(c, lookup, e.valueOf(c))...)
	}
	return issues
}
