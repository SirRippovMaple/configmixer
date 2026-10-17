package dotenv

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Parse reads KEY=value lines (one per line, blank lines and '#' comments
// ignored) from r, the inverse of Write/formatValue: a value wrapped in
// double quotes has its surrounding quotes stripped and \" / \\ unescaped.
func Parse(r io.Reader) (map[string]string, error) {
	kv := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return nil, fmt.Errorf("invalid line %q: missing '='", line)
		}
		key := strings.TrimSpace(line[:idx])
		value, err := unquote(strings.TrimSpace(line[idx+1:]))
		if err != nil {
			return nil, fmt.Errorf("invalid value for key %q: %w", key, err)
		}
		kv[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return kv, nil
}

func unquote(v string) (string, error) {
	if len(v) < 2 || v[0] != '"' {
		return v, nil
	}
	if v[len(v)-1] != '"' {
		return "", fmt.Errorf("unterminated quoted value %q", v)
	}
	return strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(v[1 : len(v)-1]), nil
}
