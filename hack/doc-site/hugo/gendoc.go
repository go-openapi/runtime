//go:build ignore

// Local development script for Hugo documentation.
//
// Usage: go run gendoc.go
//
// Requires:
// * hugo
// * git
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

//nolint:forbidigo,dogsled
func main() {
	ctx := context.Background()

	// Change to the directory containing this script.
	_, thisFile, _, _ := runtime.Caller(0)
	scriptDir := filepath.Dir(thisFile)
	if err := os.Chdir(scriptDir); err != nil {
		fatalf("chdir: %v", err)
	}

	fmt.Println("==> Preparing Hugo documentation site...")

	latestRelease := gitLatestRelease(ctx)
	requiredGoVersion := goVersionFromMod()
	buildTime := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	versionMessage := "Documentation test for latest master"

	fmt.Printf("    Latest release: %s\n", latestRelease)
	fmt.Printf("    Go version: %s\n", requiredGoVersion)
	fmt.Printf("    Build time: %s\n", buildTime)

	// Generate dynamic config from template.
	generateRuntimeYAML(requiredGoVersion, latestRelease, versionMessage, buildTime)
	fmt.Println("==> Generated runtime.yaml")

	// Check if theme exists.
	if _, err := os.Stat("themes/hugo-relearn"); os.IsNotExist(err) {
		fatalf("Relearn theme not found at themes/hugo-relearn\n" +
			"Run: unzip hugo-theme-relearn-main.zip -d themes/ && mv themes/hugo-theme-relearn-main themes/hugo-relearn")
	}

	// Check if generated docs exist.
	if _, err := os.Stat("../../../docs/doc-site"); os.IsNotExist(err) {
		fmt.Println("WARNING: Generated docs not found at ../../../docs/doc-site")
		fmt.Println("You may need to run: go generate ./...")
		fmt.Println()
		fmt.Println("Creating placeholder content directory...")
		os.MkdirAll("content", 0o755) //nolint:errcheck,mnd
	}

	fmt.Println("==> Starting Hugo development server...")
	fmt.Println("    Visit: http://localhost:1313/runtime/")
	fmt.Println()

	// Start Hugo server with both configs.
	cmd := exec.CommandContext(ctx, "hugo", "server",
		"--config", "hugo.yaml,runtime.yaml",
		"--buildDrafts",
		"--disableFastRender",
		"--navigateToChanged",
		"--bind", "0.0.0.0",
		"--port", "1313",
		"--baseURL", "http://localhost:1313/runtime/",
		"--appendPort=false",
		"--logLevel", "info",
		"--cleanDestinationDir",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fatalf("hugo: %v", err)
	}
}

// gitLatestRelease returns the latest semver tag, or "dev" if none found.
func gitLatestRelease(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "git", "tag", "--list", "--sort", "-version:refname", "v*").Output()
	if err != nil || len(out) == 0 {
		return "dev"
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	if sc.Scan() {
		return strings.TrimSpace(sc.Text())
	}
	return "dev"
}

// goVersionFromMod extracts the go version from the root go.mod.
func goVersionFromMod() string {
	data, err := os.ReadFile("../../../go.mod")
	if err != nil {
		fatalf("reading go.mod: %v", err)
	}
	re := regexp.MustCompile(`(?m)^go\s+(\S+)`)
	m := re.FindSubmatch(data)
	if m == nil {
		fatalf("could not find go version in go.mod")
	}
	return string(m[1])
}

// generateRuntimeYAML reads the template and writes runtime.yaml with substitutions.
func generateRuntimeYAML(goVersion, latestRelease, versionMessage, buildTime string) {
	tmpl, err := os.ReadFile("runtime.yaml.template")
	if err != nil {
		fatalf("reading template: %v", err)
	}

	out := string(tmpl)
	out = strings.ReplaceAll(out, "{{ GO_VERSION }}", goVersion)
	out = strings.ReplaceAll(out, "{{ LATEST_RELEASE }}", latestRelease)
	out = strings.ReplaceAll(out, "{{ VERSION_MESSAGE }}", versionMessage)
	out = strings.ReplaceAll(out, "{{ BUILD_TIME }}", buildTime)

	if err := os.WriteFile("runtime.yaml", []byte(out), 0o600); err != nil { //nolint:mnd
		fatalf("writing runtime.yaml: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
