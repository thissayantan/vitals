package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFieldHelpers(t *testing.T) {
	if got := field("go version go1.24.0 linux/amd64", 2); got != "go1.24.0" {
		t.Errorf("field 2 = %q, want go1.24.0", got)
	}
	if got := field("one two", 5); got != "" {
		t.Errorf("out-of-range field = %q, want empty", got)
	}
	if got := field("x", -1); got != "" {
		t.Errorf("negative field = %q, want empty", got)
	}
	if got := firstWord("  hello world"); got != "hello" {
		t.Errorf("firstWord = %q, want hello", got)
	}
	if got := firstLine("first\nsecond"); got != "first" {
		t.Errorf("firstLine = %q, want first", got)
	}
	if got := firstLine("only"); got != "only" {
		t.Errorf("firstLine single = %q, want only", got)
	}
}

func TestParseJavaVersion(t *testing.T) {
	in := `openjdk version "21.0.2" 2024-01-16` + "\n" + `OpenJDK Runtime Environment`
	if got := parseJavaVersion(in); got != "v21.0.2" {
		t.Errorf("parseJavaVersion = %q, want v21.0.2", got)
	}
	if got := parseJavaVersion("no quotes here"); got != "" {
		t.Errorf("parseJavaVersion(no quotes) = %q, want empty", got)
	}
}

func TestParseElixirVersion(t *testing.T) {
	in := "Erlang/OTP 26\n\nElixir 1.16.2 (compiled with Erlang/OTP 26)"
	if got := parseElixirVersion(in); got != "v1.16.2" {
		t.Errorf("parseElixirVersion = %q, want v1.16.2", got)
	}
	if got := parseElixirVersion("no elixir line"); got != "" {
		t.Errorf("parseElixirVersion(absent) = %q, want empty", got)
	}
}

func TestParseSwiftVersion(t *testing.T) {
	in := "Apple Swift version 5.9.2 (swiftlang-5.9.2.2.56)\nTarget: arm64-apple-macosx14.0"
	if got := parseSwiftVersion(in); got != "v5.9.2" {
		t.Errorf("parseSwiftVersion = %q, want v5.9.2", got)
	}
	if got := parseSwiftVersion("unrelated"); got != "" {
		t.Errorf("parseSwiftVersion(absent) = %q, want empty", got)
	}
}

func TestDetectMatchesGoMod(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// detect() runs the version command; we only assert the language is matched,
	// since the resolved version depends on the local toolchain.
	if info := detect(dir); info.Language != "go" {
		t.Errorf("detect(go.mod dir).Language = %q, want go", info.Language)
	}
}

func TestDetectPinFileNoExec(t *testing.T) {
	// A pinned-version file is read before any exec, so the result is
	// deterministic without a node toolchain installed.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), "{}")
	mustWrite(t, filepath.Join(dir, ".nvmrc"), "v20.11.1\n")

	info := detect(dir)
	if info.Language != "node" {
		t.Fatalf("Language = %q, want node", info.Language)
	}
	if info.Version != "v20.11.1" {
		t.Errorf("Version = %q, want v20.11.1 (from .nvmrc)", info.Version)
	}
}

func TestDetectNoMatch(t *testing.T) {
	if info := detect(t.TempDir()); info.Language != "" {
		t.Errorf("empty dir Language = %q, want empty", info.Language)
	}
}

func TestDetectGlobMatch(t *testing.T) {
	// dotnet matches by *.csproj glob, not an exact filename.
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "App.csproj"), "<Project/>")
	if info := detect(dir); info.Language != "dotnet" {
		t.Errorf("csproj dir Language = %q, want dotnet", info.Language)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
