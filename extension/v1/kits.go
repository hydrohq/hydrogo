package v1

import (
	"fmt"
	"sort"
)

type CapabilityKit string

const (
	KitLocalization CapabilityKit = "localization"
	KitMedia        CapabilityKit = "media"
	KitSEO          CapabilityKit = "seo"
	KitForms        CapabilityKit = "forms"
	KitRedirects    CapabilityKit = "redirects"
	KitSearch       CapabilityKit = "search"
	KitCommerce     CapabilityKit = "commerce"
)

type KitDescriptor struct {
	Name         CapabilityKit   `json:"name"`
	Dependencies []CapabilityKit `json:"dependencies,omitempty"`
	Optional     bool            `json:"optional"`
}

var kitDependencies = map[CapabilityKit][]CapabilityKit{KitCommerce: {KitForms}}

// ResolveCapabilityKits validates names, expands dependencies and returns a
// deterministic set. It does not activate routes, migrations or workers.
func ResolveCapabilityKits(kits []CapabilityKit) ([]CapabilityKit, error) {
	seen := map[CapabilityKit]struct{}{}
	for _, kit := range kits {
		if !knownCapabilityKit(kit) {
			return nil, fmt.Errorf("unknown capability kit %q", kit)
		}
		if _, ok := seen[kit]; ok {
			return nil, fmt.Errorf("duplicate capability kit %q", kit)
		}
		seen[kit] = struct{}{}
	}
	resolved := map[CapabilityKit]struct{}{}
	var add func(CapabilityKit) error
	add = func(kit CapabilityKit) error {
		if _, ok := resolved[kit]; ok {
			return nil
		}
		if !knownCapabilityKit(kit) {
			return fmt.Errorf("unknown capability kit %q", kit)
		}
		resolved[kit] = struct{}{}
		for _, dep := range kitDependencies[kit] {
			if err := add(dep); err != nil {
				return err
			}
		}
		return nil
	}
	for _, kit := range kits {
		if err := add(kit); err != nil {
			return nil, err
		}
	}
	result := make([]CapabilityKit, 0, len(resolved))
	for kit := range resolved {
		result = append(result, kit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func DescribeCapabilityKits(kits []CapabilityKit) ([]KitDescriptor, error) {
	resolved, err := ResolveCapabilityKits(kits)
	if err != nil {
		return nil, err
	}
	result := make([]KitDescriptor, 0, len(resolved))
	for _, kit := range resolved {
		result = append(result, KitDescriptor{Name: kit, Dependencies: append([]CapabilityKit(nil), kitDependencies[kit]...), Optional: true})
	}
	return result, nil
}

func knownCapabilityKit(kit CapabilityKit) bool {
	switch kit {
	case KitLocalization, KitMedia, KitSEO, KitForms, KitRedirects, KitSearch, KitCommerce:
		return true
	default:
		return false
	}
}
