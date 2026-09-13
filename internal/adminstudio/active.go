package adminstudio

// FilterActiveCollections returns only collections whose owning module is
// active. Variant metadata follows the Commerce module lifecycle.
func FilterActiveCollections(collections []Collection, active map[string]bool) []Collection {
	result := make([]Collection, 0, len(collections))
	for _, collection := range collections {
		module := collection.ID
		if module == "commerce_variants" {
			module = "commerce"
		}
		if active[module] {
			result = append(result, cloneCollection(collection))
		}
	}
	return result
}
