package mixin

import "fmt"

// Merge combines the env vars of every mixin in the selection. It is an error
// for the same key to appear in more than one mixin.
func Merge(sel Selection) (map[string]string, error) {
	result := map[string]string{}
	owner := map[string]string{} // key -> "category/id" that set it

	categories := make([]string, 0, len(sel))
	for c := range sel {
		categories = append(categories, c)
	}

	for _, category := range categories {
		m := sel[category]
		for k, v := range m.Env {
			if prev, ok := owner[k]; ok {
				return nil, fmt.Errorf("duplicate key %q: set by both %s and %s/%s", k, prev, category, m.ID)
			}
			result[k] = v
			owner[k] = fmt.Sprintf("%s/%s", category, m.ID)
		}
	}
	return result, nil
}
