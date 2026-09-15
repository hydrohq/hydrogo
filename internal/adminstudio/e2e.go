package adminstudio

import "strings"

// E2EManifest is the safe, presentation-only projection consumed by the
// metadata-driven browser test generator. It contains no secrets or policy
// decisions; authorization remains enforced by the route itself.
type E2EManifest struct {
	Version int              `json:"version"`
	Admin   E2EAdminManifest `json:"admin"`
}

type E2EAdminManifest struct {
	LoginPath   string          `json:"login_path"`
	Collections []E2ECollection `json:"collections"`
}

type E2ECollection struct {
	ID     string     `json:"id"`
	Route  string     `json:"route"`
	Fields []E2EField `json:"fields"`
	Kits   []Kit      `json:"kits,omitempty"`
}

type E2EField struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
	ReadOnly bool   `json:"read_only,omitempty"`
	TestID   string `json:"test_id"`
	Widget   string `json:"widget,omitempty"`
}

func (r Registry) E2EManifest() E2EManifest {
	return E2EManifestForCollections(r.Collections())
}

// E2EManifestForCollections builds the browser-test projection for an already
// filtered active-module collection set.
func E2EManifestForCollections(collections []Collection) E2EManifest {
	result := E2EManifest{Version: 1, Admin: E2EAdminManifest{LoginPath: "/login", Collections: make([]E2ECollection, 0, len(collections))}}
	for _, collection := range collections {
		if collection.ID == "commerce_variants" {
			continue
		}
		route := ""
		if len(collection.Actions) > 0 {
			route = collection.Actions[0].Href
		}
		if route != "" && !strings.HasSuffix(route, "/new") {
			route += "/new"
		}
		item := E2ECollection{ID: collection.ID, Route: route, Fields: make([]E2EField, 0, len(collection.Fields)), Kits: append([]Kit(nil), collection.Kits...)}
		for _, field := range collection.Fields {
			item.Fields = append(item.Fields, E2EField{ID: field.ID, Required: field.Required, ReadOnly: field.ReadOnly, Widget: field.Widget, TestID: "admin-field-" + collection.ID + "-" + field.ID})
		}
		result.Admin.Collections = append(result.Admin.Collections, item)
	}
	return result
}
