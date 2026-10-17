package mixin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"

	"gopkg.in/yaml.v3"

	"configmixer/internal/dotenv"
)

// ResolvedEnv returns this mixin's full set of key/value pairs: its static
// Env entries merged with the output of its Script (if any). An overlapping
// key between the two sources is an error.
func (m Mixin) ResolvedEnv() (map[string]string, error) {
	if m.Script == nil {
		return m.Env, nil
	}

	scripted, err := runScript(m.Script)
	if err != nil {
		return nil, fmt.Errorf("mixin %s/%s: %w", m.Category, m.ID, err)
	}

	result := make(map[string]string, len(m.Env)+len(scripted))
	maps.Copy(result, m.Env)
	for k, v := range scripted {
		if _, exists := result[k]; exists {
			return nil, fmt.Errorf("mixin %s/%s: duplicate key %q set by both static env and script output", m.Category, m.ID, k)
		}
		result[k] = v
	}
	return result, nil
}

// runScript executes spec.Path and transforms its stdout into a flat
// map[string]string, regardless of which format it was written in -- dotenv
// is the common currency the rest of the app works with.
func runScript(spec *ScriptSpec) (map[string]string, error) {
	cmd := exec.Command(spec.Path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running script %q: %w (stderr: %s)", spec.Path, err, stderr.String())
	}

	switch spec.Format {
	case "dotenv":
		kv, err := dotenv.Parse(&stdout)
		if err != nil {
			return nil, fmt.Errorf("parsing dotenv output of script %q: %w", spec.Path, err)
		}
		return kv, nil
	case "json":
		var raw map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
			return nil, fmt.Errorf("parsing json output of script %q: %w", spec.Path, err)
		}
		return toDotenvValues(spec.Path, raw)
	case "yaml":
		var raw map[string]any
		if err := yaml.Unmarshal(stdout.Bytes(), &raw); err != nil {
			return nil, fmt.Errorf("parsing yaml output of script %q: %w", spec.Path, err)
		}
		return toDotenvValues(spec.Path, raw)
	default:
		return nil, fmt.Errorf("script %q: unknown format %q", spec.Path, spec.Format)
	}
}

// toDotenvValues converts a decoded JSON/YAML object into the flat
// map[string]string dotenv values are represented as, stringifying scalars
// and rejecting nested objects/arrays that dotenv has no way to express.
func toDotenvValues(scriptPath string, raw map[string]any) (map[string]string, error) {
	kv := make(map[string]string, len(raw))
	for k, v := range raw {
		s, err := scalarToString(v)
		if err != nil {
			return nil, fmt.Errorf("script %q: key %q: %w", scriptPath, k, err)
		}
		kv[k] = s
	}
	return kv, nil
}

func scalarToString(v any) (string, error) {
	switch val := v.(type) {
	case nil:
		return "", nil
	case string:
		return val, nil
	case bool, int, int64, float64:
		return fmt.Sprint(val), nil
	default:
		return "", fmt.Errorf("cannot represent a nested value as a dotenv entry")
	}
}
