package mixin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Mixin is a single selectable config fragment within a category.
type Mixin struct {
	ID       string
	Category string
	Tags     map[string]string
	Requires []Expr
	Excludes []Expr
	Env      map[string]string
	Script   *ScriptSpec
}

// ScriptSpec describes an external script a mixin runs to generate
// additional key/value pairs at generate-time.
type ScriptSpec struct {
	Path   string
	Format string // "dotenv", "json", or "yaml"
}

type rawMixin struct {
	Tags     map[string]string `yaml:"tags"`
	Requires []string          `yaml:"requires"`
	Excludes []string          `yaml:"excludes"`
	Env      map[string]string `yaml:"env"`
	Script   *rawScript        `yaml:"script"`
}

type rawScript struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
}

// Catalog holds all mixins grouped by category.
type Catalog struct {
	Categories map[string][]Mixin
}

// CategoryNames returns the category names in sorted order.
func (c *Catalog) CategoryNames() []string {
	names := make([]string, 0, len(c.Categories))
	for n := range c.Categories {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// OrderedCategories returns category names topologically sorted so that a
// category produces tags another category's mixins require/exclude is
// ordered before that dependent category. Categories with no dependency
// relationship are ordered alphabetically relative to each other. Returns an
// error if the categories form a dependency cycle.
func (c *Catalog) OrderedCategories() ([]string, error) {
	produces := map[string]map[string]bool{}
	requires := map[string]map[string]bool{}
	for category, mixins := range c.Categories {
		produces[category] = map[string]bool{}
		requires[category] = map[string]bool{}
		for _, m := range mixins {
			for k := range m.Tags {
				produces[category][k] = true
			}
			for _, e := range m.Requires {
				for _, k := range e.Keys() {
					requires[category][k] = true
				}
			}
			for _, e := range m.Excludes {
				for _, k := range e.Keys() {
					requires[category][k] = true
				}
			}
		}
	}

	// deps[category] is the set of categories that must be ordered first.
	deps := map[string]map[string]bool{}
	for category := range c.Categories {
		deps[category] = map[string]bool{}
		for other := range c.Categories {
			if other == category {
				continue
			}
			for k := range requires[category] {
				if produces[other][k] {
					deps[category][other] = true
				}
			}
		}
	}

	var order []string
	remaining := c.CategoryNames()
	for len(remaining) > 0 {
		var ready []string
		for _, category := range remaining {
			if len(deps[category]) == 0 {
				ready = append(ready, category)
			}
		}
		if len(ready) == 0 {
			sort.Strings(remaining)
			return nil, fmt.Errorf("dependency cycle detected among categories: %s", strings.Join(remaining, ", "))
		}
		sort.Strings(ready)
		order = append(order, ready...)

		readySet := map[string]bool{}
		for _, category := range ready {
			readySet[category] = true
		}
		var next []string
		for _, category := range remaining {
			if readySet[category] {
				continue
			}
			for dep := range deps[category] {
				if readySet[dep] {
					delete(deps[category], dep)
				}
			}
			next = append(next, category)
		}
		remaining = next
	}
	return order, nil
}

// Find returns the mixin with the given id within a category.
func (c *Catalog) Find(category, id string) (Mixin, bool) {
	for _, m := range c.Categories[category] {
		if m.ID == id {
			return m, true
		}
	}
	return Mixin{}, false
}

// LoadAll walks one level of subdirectories under root; each subdirectory is a
// category, and each *.yaml/*.yml file within it is a mixin.
func LoadAll(root string) (*Catalog, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading mixins dir %q: %w", root, err)
	}

	cat := &Catalog{Categories: map[string][]Mixin{}}
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		category := entry.Name()
		dir := filepath.Join(root, category)
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading category dir %q: %w", dir, err)
		}
		var mixins []Mixin
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			id := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			m, err := loadMixin(filepath.Join(dir, f.Name()), category, id)
			if err != nil {
				return nil, err
			}
			mixins = append(mixins, m)
		}
		if len(mixins) > 0 {
			sort.Slice(mixins, func(i, j int) bool { return mixins[i].ID < mixins[j].ID })
			cat.Categories[category] = mixins
		}
	}
	if len(cat.Categories) == 0 {
		return nil, fmt.Errorf("no mixin categories found under %q", root)
	}
	return cat, nil
}

func loadMixin(path, category, id string) (Mixin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Mixin{}, fmt.Errorf("reading mixin %q: %w", path, err)
	}
	var raw rawMixin
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Mixin{}, fmt.Errorf("parsing mixin %q: %w", path, err)
	}

	m := Mixin{
		ID:       id,
		Category: category,
		Tags:     raw.Tags,
		Env:      raw.Env,
	}
	for _, s := range raw.Requires {
		e, err := ParseExpr(s)
		if err != nil {
			return Mixin{}, fmt.Errorf("mixin %q requires clause: %w", path, err)
		}
		m.Requires = append(m.Requires, e)
	}
	for _, s := range raw.Excludes {
		e, err := ParseExpr(s)
		if err != nil {
			return Mixin{}, fmt.Errorf("mixin %q excludes clause: %w", path, err)
		}
		m.Excludes = append(m.Excludes, e)
	}
	if raw.Script != nil {
		format := raw.Script.Format
		if format == "" {
			format = "dotenv"
		}
		switch format {
		case "dotenv", "json", "yaml":
		default:
			return Mixin{}, fmt.Errorf("mixin %q script.format %q: must be dotenv, json, or yaml", path, format)
		}
		if raw.Script.Path == "" {
			return Mixin{}, fmt.Errorf("mixin %q: script.path must not be empty", path)
		}
		scriptPath := raw.Script.Path
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(filepath.Dir(path), scriptPath)
		}
		m.Script = &ScriptSpec{Path: scriptPath, Format: format}
	}
	return m, nil
}
