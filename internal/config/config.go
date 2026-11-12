// Package config loads lct.yaml, the repo-level configuration of the lct tool.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the name of the config file that marks the repo root.
const FileName = "lct.yaml"

// ErrRootNotFound is returned by FindRoot when no directory up the tree holds
// FileName.
var ErrRootNotFound = errors.New(FileName + " not found")

// Config is the content of lct.yaml.
type Config struct {
	LangDir     string    `yaml:"lang_dir"`
	Frozen      []string  `yaml:"frozen"`
	CDNTemplate string    `yaml:"cdn_template"`
	Worktrees   Worktrees `yaml:"worktrees"`
	Batch       Batch     `yaml:"batch"`
	Build       Build     `yaml:"build"`
	Names       Names     `yaml:"names"`
	Font        Font      `yaml:"font"`
	Bootstrap   Bootstrap `yaml:"bootstrap"`
}

// Worktrees holds the worktree directories, relative to the repo root.
type Worktrees struct {
	Raw string `yaml:"raw"`
	TW  string `yaml:"tw"`
}

// Batch holds the limits applied when cutting translation batches.
type Batch struct {
	MaxUnits      int `yaml:"max_units"`
	StoryMaxUnits int `yaml:"story_max_units"`
	MaxNames      int `yaml:"max_names"`
	MaxAttempts   int `yaml:"max_attempts"`
	LeaseMinutes  int `yaml:"lease_minutes"`
}

// Build holds the limits applied when building the language pack.
type Build struct {
	// MaxInconsistent is how many "same KR, different renderings" groups the
	// build record keeps, besides those with units of the update.
	MaxInconsistent int `yaml:"max_inconsistent"`
}

// Names lists the fields whose current translations make up the table of
// existing names.
type Names struct {
	Fields []string `yaml:"fields"`
}

// Font pins the font file bundled into the language pack.
type Font struct {
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
	SHA256  string `yaml:"sha256"`
}

// Bootstrap pins the one-off sources used to seed the first translation.
type Bootstrap struct {
	RawToken     string   `yaml:"raw_token"`
	RawCreatedAt string   `yaml:"raw_created_at"`
	Paratranz    Upstream `yaml:"paratranz"`
	LTM          Upstream `yaml:"ltm"`
	LLC          Upstream `yaml:"llc"`
}

// Upstream names a release asset of a GitHub repository and pins its
// content: SHA256 is the hex sha256 of the asset file.
type Upstream struct {
	Repo   string `yaml:"repo"`
	Tag    string `yaml:"tag"`
	Asset  string `yaml:"asset"`
	SHA256 string `yaml:"sha256"`
}

// Load reads and parses the lct.yaml in root. Unknown keys are an error so
// that typos do not silently fall back to zero values.
func Load(root string) (*Config, error) {
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// FindRoot returns the nearest directory at or above start that contains
// lct.yaml.
func FindRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		info, err := os.Stat(filepath.Join(dir, FileName))
		switch {
		case err == nil && info.Mode().IsRegular():
			return dir, nil
		case err != nil && !errors.Is(err, fs.ErrNotExist):
			// Not being allowed to look, or a broken path, is not the same
			// as there being no lct.yaml here: a parent's file would be the
			// wrong repo.
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w in %s or any parent directory", ErrRootNotFound, start)
		}
		dir = parent
	}
}
