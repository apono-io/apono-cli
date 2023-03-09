package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/kirsle/configdir"
)

var (
	configDirPath  = configdir.LocalConfig("apono-cli")
	configFilePath = path.Join(configDirPath, "config.json")
)

func Get() (*Config, error) {
	cfg := new(Config)
	configFile, err := os.Open(filepath.Clean(configFilePath))
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}

		return nil, fmt.Errorf("failed to open config file: %w", err)
	}

	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Printf("failed to close config file: %s", err.Error())
		}
	}(configFile)

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	err = json.Unmarshal(configBytes, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configBytes, err := json.Marshal(*cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if _, err = os.Stat("/path/to/whatever"); os.IsNotExist(err) {
		err = configdir.MakePath(configDirPath) // Ensure it exists.
		if err != nil {
			return err
		}
	}

	configFile, err := os.Create(filepath.Clean(configFilePath))
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}

	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			log.Printf("failed to close config file: %s", err.Error())
		}
	}(configFile)

	_, err = io.Copy(configFile, bytes.NewBuffer(configBytes))
	if err != nil {
		return fmt.Errorf("failed write config to file: %w", err)
	}

	return nil
}
