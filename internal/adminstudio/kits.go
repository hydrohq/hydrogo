package adminstudio

import "sort"

type Kit string

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

var knownKits = map[Kit]struct{}{KitLocalization: {}, KitMedia: {}, KitSEO: {}, KitForms: {}, KitRedirects: {}, KitSearch: {}, KitCommerce: {}}

func NewKitRegistry(kits ...Kit) KitRegistry {
	set := make(map[Kit]struct{}, len(kits))
	for _, kit := range kits {
		set[kit] = struct{}{}
	}
	return KitRegistry{kits: set}
}

func (r KitRegistry) Enabled(kit Kit) bool { _, ok := r.kits[kit]; return ok }

func IsKnownKit(kit Kit) bool { _, ok := knownKits[kit]; return ok }

func (r KitRegistry) Kits() []Kit {
	result := make([]Kit, 0, len(r.kits))
	for kit := range r.kits {
		result = append(result, kit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
