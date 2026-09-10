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
// the current working directory. name is the calling command's name, used
// to prefix every error so each command reports its own errors rather than
// borrowing another command's label.
func resolveFabric(cmd *cobra.Command, name string) (resolvedFabric, error) {
	fabricSlug, err := cmd.Flags().GetString("fabric")
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
	}
	if cmd.Flags().Changed("fabric") && fabricSlug == "" {
		return resolvedFabric{}, fmt.Errorf("%s: --fabric requires a non-empty slug", name)
	}

	configDir, err := xdgpath.ConfigDir()
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
	}
	idx, err := fabric.LoadIndex(filepath.Join(configDir, "fabrics.yaml"))
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
	}

	var entry fabric.IndexEntry
	if fabricSlug != "" {
		var ok bool
		entry, ok = idx.BySlug(fabricSlug)
		if !ok {
			return resolvedFabric{}, fmt.Errorf("%s: fabric %q is not registered", name, fabricSlug)
		}
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
		}
		entry, err = idx.ByCWD(cwd)
		if err != nil {
			return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
		}
	}

	dataDir, err := xdgpath.DataDir()
	if err != nil {
		return resolvedFabric{}, fmt.Errorf("%s: %s", name, err)
	}
	return resolvedFabric{
		IndexEntry: entry,
		PacketsDir: filepath.Join(dataDir, entry.Slug, "packets"),
	}, nil
}
