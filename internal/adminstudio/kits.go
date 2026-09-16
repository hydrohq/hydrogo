package adminstudio

import (
	"errors"
	"sort"
)

type Kit string

// KitDescriptor is provider-neutral capability metadata. Implementations and
// runtime wiring remain owned by the module/Client Edition.
type KitDescriptor struct {
	Name         Kit   `json:"name"`
	Dependencies []Kit `json:"dependencies,omitempty"`
	Optional     bool  `json:"optional"`
}

const (
	KitLocalization Kit = "localization"
	KitMedia        Kit = "media"
	KitSEO          Kit = "seo"
	KitForms        Kit = "forms"
	KitRedirects    Kit = "redirects"
	KitSearch       Kit = "search"
	KitCommerce     Kit = "commerce"
)

// KitRegistry describes optional product capabilities without importing their
// implementations. Client Editions can enable only the kits they use.
type KitRegistry struct{ kits map[Kit]struct{} }

var ErrInvalidKitSet = errors.New("invalid capability kit set")

var knownKits = map[Kit]struct{}{KitLocalization: {}, KitMedia: {}, KitSEO: {}, KitForms: {}, KitRedirects: {}, KitSearch: {}, KitCommerce: {}}

var kitDependencies = map[Kit][]Kit{
	KitCommerce: {KitForms},
}

func NewKitRegistry(kits ...Kit) KitRegistry {
	set := make(map[Kit]struct{}, len(kits))
	for _, kit := range kits {
		set[kit] = struct{}{}
	}
	return KitRegistry{kits: set}
}

// NewResolvedKitRegistry validates the declared set, expands its allowlisted
// dependencies, and returns a deterministic registry for a composition root.
// The legacy NewKitRegistry constructor remains available for callers that
// intentionally build a raw registry during migration.
func NewResolvedKitRegistry(kits ...Kit) (KitRegistry, error) {
	resolved, err := ResolveKits(kits)
	if err != nil {
		return KitRegistry{}, err
	}
	return NewKitRegistry(resolved...), nil
}

func ValidateKits(kits []Kit) error {
	seen := make(map[Kit]struct{}, len(kits))
	for _, kit := range kits {
		if !IsKnownKit(kit) {
			return ErrInvalidKitSet
		}
		if _, ok := seen[kit]; ok {
			return ErrInvalidKitSet
		}
		seen[kit] = struct{}{}
	}
	return nil
}

// ResolveKits expands declared kit dependencies and returns a deterministic
// set. Dependencies are metadata only; implementations remain module-owned.
func ResolveKits(kits []Kit) ([]Kit, error) {
	if err := ValidateKits(kits); err != nil {
		return nil, err
	}
	resolved := make(map[Kit]struct{}, len(kits))
	var add func(Kit) error
	add = func(kit Kit) error {
		if _, ok := resolved[kit]; ok {
			return nil
		}
		if !IsKnownKit(kit) {
			return ErrInvalidKitSet
		}
		resolved[kit] = struct{}{}
		for _, dependency := range kitDependencies[kit] {
			if err := add(dependency); err != nil {
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
	result := make([]Kit, 0, len(resolved))
	for kit := range resolved {
		result = append(result, kit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func (r KitRegistry) Enabled(kit Kit) bool { _, ok := r.kits[kit]; return ok }

func IsKnownKit(kit Kit) bool { _, ok := knownKits[kit]; return ok }

// DescribeKits returns deterministic metadata for the resolved kit set.
func DescribeKits(kits []Kit) ([]KitDescriptor, error) {
	resolved, err := ResolveKits(kits)
	if err != nil {
		return nil, err
	}
	result := make([]KitDescriptor, 0, len(resolved))
	for _, kit := range resolved {
		deps := append([]Kit(nil), kitDependencies[kit]...)
		sort.Slice(deps, func(i, j int) bool { return deps[i] < deps[j] })
		result = append(result, KitDescriptor{Name: kit, Dependencies: deps, Optional: true})
	}
	return result, nil
}

func (r KitRegistry) Kits() []Kit {
	result := make([]Kit, 0, len(r.kits))
	for kit := range r.kits {
		result = append(result, kit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
