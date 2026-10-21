package contenttypes

import "github.com/ardaozyildirim/hydrogo/internal/adminstudio"

// HideUnreadableFields drops from the admin schema the fields that name a
// permission the caller lacks (can reports whether they hold it), so forms and
// lists do not show a label for a value the API never sends. Collections that
// do not come from definitions pass through; the slice is changed in place.
func HideUnreadableFields(collections []adminstudio.Collection, definitions []Definition, can func(permission string) bool) []adminstudio.Collection {
	restricted := map[string]map[string]string{}
	for _, definition := range definitions {
		for _, field := range definition.Fields {
			if field.Permission != "" {
				if restricted[definition.ID] == nil {
					restricted[definition.ID] = map[string]string{}
				}
				restricted[definition.ID][field.ID] = field.Permission
			}
		}
	}
	for index, collection := range collections {
		hidden := restricted[collection.ID]
		if len(hidden) == 0 {
			continue
		}
		kept := make([]adminstudio.Field, 0, len(collection.Fields))
		for _, field := range collection.Fields {
			if permission, ok := hidden[field.ID]; ok && !can(permission) {
				if collection.TitleField == field.ID {
					collections[index].TitleField = ""
				}
				continue
			}
			kept = append(kept, field)
		}
		collections[index].Fields = kept
	}
	return collections
}
