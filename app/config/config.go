package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/caarlos0/env/v6"
)

// Primary space search index does not contain `||`, however, the search index
// for secondary spaces are named `primary||secondary`.
var regexIndexName = regexp.MustCompile(`^SearchIndex_([a-zA-Z0-9-\|]+)\.sqlite$`)

type SearchIndex struct {
	SpaceID string
	name    string
	dir     string
}

func (si SearchIndex) Path() string {
	return filepath.Join(si.dir, si.name)
}

type Config struct {
	//nolint:lll
	IndexPathDir string `env:"INDEX_PATH_DIR" envDefault:"~/Library/Containers/com.lukilabs.lukiapp/Data/Library/Application Support/com.lukilabs.lukiapp/Search"`
	indexes      []SearchIndex
}

func (c *Config) SearchIndexes() []SearchIndex {
	return c.indexes
}

func NewConfig() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if strings.HasPrefix(config.IndexPathDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("user home dir: %w", err)
		}

		config.IndexPathDir = strings.Replace(config.IndexPathDir, "~", homeDir, 1)
	}

	entries, err := os.ReadDir(config.IndexPathDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if regexIndexName.MatchString(entry.Name()) {
			spaceID := strings.Split(regexIndexName.FindStringSubmatch(entry.Name())[1], "||")
			config.indexes = append(config.indexes, SearchIndex{
				SpaceID: spaceID[len(spaceID)-1], // Select the last entry in the slice (primary/secondary).
				name:    entry.Name(),
				dir:     config.IndexPathDir,
			})
		}
	}

	if len(config.indexes) == 0 {
		return nil, errors.New("no index files found")
	}

	return &config, nil
}
