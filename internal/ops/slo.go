package ops

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SLOConfig is intentionally target-neutral: teams can declare the shape of
// their SLI/SLOs without HydroGo inventing availability or latency objectives.
type SLOConfig struct {
	Objectives []SLO `yaml:"objectives" json:"objectives"`
}

type SLO struct {
	Name      string   `yaml:"name" json:"name"`
	Indicator string   `yaml:"indicator" json:"indicator"`
	Window    string   `yaml:"window" json:"window"`
	Target    *float64 `yaml:"target,omitempty" json:"target,omitempty"`
}

var sloNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,63}$`)
var sloWindowPattern = regexp.MustCompile(`^[1-9][0-9]*(m|h|d|w)$`)

func ParseSLOConfig(data []byte) (SLOConfig, error) {
	var config SLOConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return SLOConfig{}, fmt.Errorf("parse SLO config: %w", err)
	}
	if err := ValidateSLOConfig(config); err != nil {
		return SLOConfig{}, err
	}
	return config, nil
}

func ValidateSLOConfig(config SLOConfig) error {
	seen := make(map[string]struct{}, len(config.Objectives))
	for index, objective := range config.Objectives {
		name := strings.TrimSpace(objective.Name)
		indicator := strings.TrimSpace(objective.Indicator)
		window := strings.TrimSpace(objective.Window)
		if !sloNamePattern.MatchString(name) {
			return fmt.Errorf("SLO objective %d has invalid name", index)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate SLO objective %q", name)
		}
		seen[name] = struct{}{}
		if indicator == "" || len(indicator) > 128 || strings.ContainsAny(indicator, "\r\n") {
			return fmt.Errorf("SLO %q has invalid indicator", name)
		}
		if !sloWindowPattern.MatchString(window) {
			return fmt.Errorf("SLO %q has invalid window", name)
		}
		if objective.Target != nil && (*objective.Target <= 0 || *objective.Target > 100) {
			return fmt.Errorf("SLO %q target must be greater than 0 and at most 100", name)
		}
	}
	return nil
}

// SortedObjectives returns a stable presentation order for dashboards.
func (config SLOConfig) SortedObjectives() []SLO {
	result := append([]SLO(nil), config.Objectives...)
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
