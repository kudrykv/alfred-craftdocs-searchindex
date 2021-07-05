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

var regexIndexName = regexp.MustCompile(`^SearchIndex_([a-zA-Z0-9-]+)\.sqlite$`)

type Config struct {
	//nolint:lll
	IndexPathDir string `env:"INDEX_PATH_DIR" envDefault:"~/Library/Containers/com.lukilabs.lukiapp/Data/Library/Application Support/com.lukilabs.lukiapp/Search"`
	IndexName    string
}

func (c Config) PathToIndex() string {
	return filepath.Join(c.IndexPathDir, c.IndexName)
}

func NewConfig() (Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return config, fmt.Errorf("parse: %w", err)
	}

	if !strings.HasPrefix(config.IndexPathDir, "~/") {
		return config, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config, fmt.Errorf("user home dir: %w", err)
	}

	config.IndexPathDir = strings.Replace(config.IndexPathDir, "~", homeDir, 1)

	entries, err := os.ReadDir(config.IndexPathDir)
	if err != nil {
		return config, fmt.Errorf("read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if regexIndexName.MatchString(entry.Name()) {
			config.IndexName = entry.Name()

			break
		}
	}

	if len(config.IndexName) == 0 {
		return config, errors.New("did not find index file")
	}

	return config, nil
}
