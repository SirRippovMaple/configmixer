package dotenv

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Write renders kv as a .env file, one KEY=value line per entry, sorted by
// key for stable output. Values containing whitespace, '#', or '"' are
// double-quoted with internal quotes/backslashes escaped.
func Write(w io.Writer, kv map[string]string) error {
	keys := make([]string, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, formatValue(kv[k])); err != nil {
			return err
		}
	}
	return nil
}

func needsQuoting(v string) bool {
	return strings.ContainsAny(v, " \t#\"'\n$`\\") || v == ""
}

func formatValue(v string) string {
	if !needsQuoting(v) {
		return v
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(v)
	return `"` + escaped + `"`
}
