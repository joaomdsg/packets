package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/xdgpath"
	"github.com/spf13/cobra"
)

// resolvedFabric is what emit needs to locate a packet: the registered
// fabric entry plus the packets dir under its data directory.
type resolvedFabric struct {
	fabric.IndexEntry
	PacketsDir string
}

// resolveFabric applies the §7 default-fabric rule: --fabric selects a
// fabric by slug; with no flag, it's the fabric whose repo_path contains
// the current working directory.
func resolveFabric(cmd *cobra.Command) (resolvedFabric, error) {
	fabricSlug, err := cmd.Flags().GetString("fabric")
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("emit: %s", err)
	}

	configDir, err := xdgpath.ConfigDir()
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("emit: %s", err)
	}
	idx, err := fabric.LoadIndex(filepath.Join(configDir, "fabrics.yaml"))
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("emit: %s", err)
	}

	var entry fabric.IndexEntry
	if fabricSlug != "" {
		var ok bool
		entry, ok = idx.BySlug(fabricSlug)
		if !ok {
			return resolvedFabric{}, fmt.Errorf("emit: fabric %q is not registered", fabricSlug)
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return resolvedFabric{}, fmt.Errorf("emit: %s", err)
		}
		entry, err = idx.ByCWD(cwd)
		if err != nil {
			return resolvedFabric{}, fmt.Errorf("emit: %s", err)
		}
	}

	dataDir, err := xdgpath.DataDir()
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("emit: %s", err)
	}
	return resolvedFabric{
		IndexEntry: entry,
		PacketsDir: filepath.Join(dataDir, entry.Slug, "packets"),
	}, nil
}
