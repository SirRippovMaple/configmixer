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
	// SelectorScripts maps category name to a script that picks the mixin
	// for it. A selector script takes priority over the category's default;
	// the default is used if the script declines or picks an invalid mixin.
	SelectorScripts map[string]SelectorScriptSpec
}

// SelectorScriptSpec describes an external script that selects a mixin ID
// for a category. See RunSelectorScript for the protocol.
type SelectorScriptSpec struct {
	Path string
}

type rawConfig struct {
	Defaults        map[string]string            `yaml:"defaults"`
	SelectorScripts map[string]rawSelectorScript `yaml:"selector-scripts"`
}

type rawSelectorScript struct {
	Path string `yaml:"path"`
}

// LoadConfig reads <root>/config.yaml. A missing file is not an error; it
// yields an empty Config so the defaults and selector-scripts features are
// fully optional. Entries in Defaults are validated against cat: unknown
// categories or mixin IDs cause an error. Selector script paths are resolved
// relative to root unless absolute.
func LoadConfig(root string, cat *Catalog) (*Config, error) {
	path := filepath.Join(root, "config.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Defaults: map[string]string{}, SelectorScripts: map[string]SelectorScriptSpec{}}, nil
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

	selectorScripts := make(map[string]SelectorScriptSpec, len(raw.SelectorScripts))
	for category, s := range raw.SelectorScripts {
		if _, exists := cat.Categories[category]; !exists {
			return nil, fmt.Errorf("config %q: unknown category %q in selector-scripts", path, category)
		}
		if s.Path == "" {
			return nil, fmt.Errorf("config %q: selector-scripts.%s.path must not be empty", path, category)
		}
		scriptPath := s.Path
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(root, scriptPath)
		}
		selectorScripts[category] = SelectorScriptSpec{Path: scriptPath}
	}

	return &Config{Defaults: raw.Defaults, SelectorScripts: selectorScripts}, nil
}
