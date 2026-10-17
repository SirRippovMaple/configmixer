package mixin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatalf("writing script %q: %v", path, err)
	}
	return path
}

func TestResolvedEnvNoScript(t *testing.T) {
	m := Mixin{Env: map[string]string{"KEY": "value"}}
	got, err := m.ResolvedEnv()
	if err != nil {
		t.Fatalf("ResolvedEnv: %v", err)
	}
	if got["KEY"] != "value" {
		t.Errorf("ResolvedEnv() = %v", got)
	}
}

func TestResolvedEnvDotenvFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo 'SCRIPTED_KEY=scripted value'`)

	m := Mixin{Category: "cat", ID: "id", Script: &ScriptSpec{Path: path, Format: "dotenv"}}
	got, err := m.ResolvedEnv()
	if err != nil {
		t.Fatalf("ResolvedEnv: %v", err)
	}
	if got["SCRIPTED_KEY"] != "scripted value" {
		t.Errorf("ResolvedEnv() = %v", got)
	}
}

func TestResolvedEnvJSONFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo '{"PORT": 8200, "ENABLED": true, "NAME": "vault"}'`)

	m := Mixin{Category: "cat", ID: "id", Script: &ScriptSpec{Path: path, Format: "json"}}
	got, err := m.ResolvedEnv()
	if err != nil {
		t.Fatalf("ResolvedEnv: %v", err)
	}
	if got["PORT"] != "8200" || got["ENABLED"] != "true" || got["NAME"] != "vault" {
		t.Errorf("ResolvedEnv() = %v", got)
	}
}

func TestResolvedEnvYAMLFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `printf 'port: 8200\nenabled: true\nname: vault\n'`)

	m := Mixin{Category: "cat", ID: "id", Script: &ScriptSpec{Path: path, Format: "yaml"}}
	got, err := m.ResolvedEnv()
	if err != nil {
		t.Fatalf("ResolvedEnv: %v", err)
	}
	if got["port"] != "8200" || got["enabled"] != "true" || got["name"] != "vault" {
		t.Errorf("ResolvedEnv() = %v", got)
	}
}

func TestResolvedEnvNestedValueRejected(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo '{"NESTED": {"a": 1}}'`)

	m := Mixin{Category: "cat", ID: "id", Script: &ScriptSpec{Path: path, Format: "json"}}
	if _, err := m.ResolvedEnv(); err == nil {
		t.Error("expected error for nested JSON value, got nil")
	}
}

func TestResolvedEnvMergesStaticAndScript(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo 'SCRIPTED_KEY=from-script'`)

	m := Mixin{
		Category: "cat",
		ID:       "id",
		Env:      map[string]string{"STATIC_KEY": "from-static"},
		Script:   &ScriptSpec{Path: path, Format: "dotenv"},
	}
	got, err := m.ResolvedEnv()
	if err != nil {
		t.Fatalf("ResolvedEnv: %v", err)
	}
	if got["STATIC_KEY"] != "from-static" || got["SCRIPTED_KEY"] != "from-script" {
		t.Errorf("ResolvedEnv() = %v", got)
	}
}

func TestResolvedEnvStaticScriptKeyCollision(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo 'SHARED_KEY=from-script'`)

	m := Mixin{
		Category: "cat",
		ID:       "id",
		Env:      map[string]string{"SHARED_KEY": "from-static"},
		Script:   &ScriptSpec{Path: path, Format: "dotenv"},
	}
	if _, err := m.ResolvedEnv(); err == nil {
		t.Error("expected error for static/script key collision, got nil")
	}
}

func TestResolvedEnvScriptFailure(t *testing.T) {
	dir := t.TempDir()
	path := writeScript(t, dir, "gen.sh", `echo 'boom' >&2; exit 1`)

	m := Mixin{Category: "cat", ID: "id", Script: &ScriptSpec{Path: path, Format: "dotenv"}}
	_, err := m.ResolvedEnv()
	if err == nil {
		t.Fatal("expected error for non-zero exit script, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q should contain script stderr", err.Error())
	}
}

func TestLoadMixinUnknownScriptFormat(t *testing.T) {
	dir := t.TempDir()
	catDir := filepath.Join(dir, "cat")
	if err := os.MkdirAll(catDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mixinPath := filepath.Join(catDir, "bad.yaml")
	body := "script:\n  path: ./gen.sh\n  format: xml\n"
	if err := os.WriteFile(mixinPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadAll(dir); err == nil {
		t.Error("expected error for unknown script format, got nil")
	}
}
