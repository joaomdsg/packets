package fabric

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// IndexEntry is one fabric registered in the fabrics.yaml index.
type IndexEntry struct {
	Slug     string `yaml:"slug"`
	Remote   string `yaml:"remote"`
	RepoPath string `yaml:"repo_path"`
}

// Index is the ~/.config/packets/fabrics.yaml document: every registered
// fabric, keyed by remote URL and resolved by slug or by cwd containment.
type Index struct {
	Fabrics []IndexEntry `yaml:"fabrics"`
}

// LoadIndex reads fabrics.yaml at path. A missing file is not an error: it
// means no fabric has been registered yet (registration is packets init's
// job, Phase 3).
func LoadIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Index{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("fabric: read index %s: %s", path, err)
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("fabric: parse index %s: %s", path, err)
	}
	return &idx, nil
}

// SaveIndex writes idx as YAML to path with file mode 0600.
func SaveIndex(path string, idx *Index) error {
	data, err := yaml.Marshal(idx)
	if err != nil {
		return fmt.Errorf("fabric: marshal index: %s", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("fabric: write index %s: %s", path, err)
	}
	return nil
}

// BySlug finds the registered fabric with the given slug.
func (idx *Index) BySlug(slug string) (IndexEntry, bool) {
	for _, e := range idx.Fabrics {
		if e.Slug == slug {
			return e, true
		}
	}
	return IndexEntry{}, false
}

// ByCWD finds the fabric whose repo_path contains cwd, per the default-
// fabric rule: no --fabric flag means the fabric whose repo_path contains
// the current working directory.
func (idx *Index) ByCWD(cwd string) (IndexEntry, error) {
	for _, e := range idx.Fabrics {
		rel, err := filepath.Rel(e.RepoPath, cwd)
		if err != nil {
			continue
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return e, nil
		}
	}
	return IndexEntry{}, fmt.Errorf("fabric: no registered fabric contains %s; pass --fabric or run packets init", cwd)
}
