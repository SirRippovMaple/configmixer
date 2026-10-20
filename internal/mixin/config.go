package mixin

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds mixins-dir-wide settings, loaded from <root>/config.yaml.
type Config struct {
	// Defaults maps category name to the mixin ID that should be selected
	// for it when no flag or embedded selection resolves the category.
	Defaults map[string]string
}

type rawConfig struct {
	Defaults map[string]string `yaml:"defaults"`
}

// LoadConfig reads <root>/config.yaml. A missing file is not an error; it
// yields an empty Config so the defaults feature is fully optional. Entries
// in Defaults are validated against cat: unknown categories or mixin IDs
// cause an error.
func LoadConfig(root string, cat *Catalog) (*Config, error) {
	path := filepath.Join(root, "config.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Defaults: map[string]string{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}

	for category, id := range raw.Defaults {
		if _, exists := cat.Categories[category]; !exists {
			return nil, fmt.Errorf("config %q: unknown category %q in defaults", path, category)
		}
		if _, ok := cat.Find(category, id); !ok {
			return nil, fmt.Errorf("config %q: unknown mixin %q for category %q in defaults", path, id, category)
		}
	}

	return &Config{Defaults: raw.Defaults}, nil
}
