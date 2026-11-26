package helpers

// ToOptionValues flattens a product's options into "<option title>:<value>"
// entries, e.g. ["Size:S", "Color:Red"]. One field keeps the index simple;
// the storefront splits on the first ":" to group the facet by option name.
func ToOptionValues(options any) []any {
	out := []any{}
	// A value shared by several options would otherwise be counted twice.
	seen := map[string]bool{}
	for _, option := range Records(options) {
		title, ok := Trimmed(option, "title")
		if !ok {
			continue
		}
		for _, optionValue := range Records(Get(option, "values")) {
			value, ok := Trimmed(optionValue, "value")
			if !ok {
				continue
			}
			entry := title + ":" + value
			if !seen[entry] {
				seen[entry] = true
				out = append(out, entry)
			}
		}
	}
	return out
}
