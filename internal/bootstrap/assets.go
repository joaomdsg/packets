package bootstrap

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets
var assetsFS embed.FS

// imageAssets maps the paths written into the fabric's data dir to their
// embedded source; hooks get 0700 so they're executable inside the image
// build, everything else 0600.
var imageAssets = map[string]string{
	"Dockerfile":    "assets/Dockerfile",
	"settings.json": "assets/settings.json",
	"entrypoint.sh": "assets/entrypoint.sh",
	"hooks/pre.sh":  "assets/hooks/pre.sh",
	"hooks/post.sh": "assets/hooks/post.sh",
	"hooks/stop.sh": "assets/hooks/stop.sh",
}

// writeImageAssets writes the Dockerfile (with base substituted for
// {{BASE}}), settings.json, entrypoint.sh, and hooks/ into dataDir (§13.1
// -13.5).
func writeImageAssets(dataDir, base string) error {
	if err := os.MkdirAll(filepath.Join(dataDir, "hooks"), 0o700); err != nil {
		return fmt.Errorf("bootstrap: create %s: %s", filepath.Join(dataDir, "hooks"), err)
	}

	for rel, embedded := range imageAssets {
		data, err := assetsFS.ReadFile(embedded)
		if err != nil {
			return fmt.Errorf("bootstrap: read embedded %s: %s", embedded, err)
		}
		if rel == "Dockerfile" {
			data = []byte(strings.ReplaceAll(string(data), "{{BASE}}", base))
		}
		mode := os.FileMode(0o600)
		if strings.HasSuffix(rel, ".sh") {
			mode = 0o700
		}
		dest := filepath.Join(dataDir, rel)
		if err := os.WriteFile(dest, data, mode); err != nil {
			return fmt.Errorf("bootstrap: write %s: %s", dest, err)
		}
	}
	return nil
}

// workflowContent renders the CI workflow (§13.9) with toolchainSetup
// substituted for {{TOOLCHAIN_SETUP}}.
func workflowContent(toolchainSetup string) (string, error) {
	data, err := assetsFS.ReadFile("assets/workflow.yml")
	if err != nil {
		return "", fmt.Errorf("bootstrap: read embedded workflow.yml: %s", err)
	}
	return strings.ReplaceAll(string(data), "{{TOOLCHAIN_SETUP}}", toolchainSetup), nil
}
