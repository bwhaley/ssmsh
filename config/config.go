package config

import (
	"os"
	"path/filepath"

	gcfg "gopkg.in/gcfg.v1"
)

const DefaultConfigFileName = ".ssmshrc"

// Config holds the default shell configuration
type Config struct {
	Default struct {
		Decrypt   bool
		Key       string
		Profile   string
		Region    string
		Overwrite bool
		Type      string
		Output    string
	}
}

// ReadConfig reads ssmsh configuration from a given file
func ReadConfig(cfgFile string) (Config, error) {
	if cfgFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		cfgFile = filepath.Join(homeDir, DefaultConfigFileName)
	}

	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		// Config file is not required, just return an empty config
		return Config{}, nil
	}

	var cfg Config
	err := gcfg.ReadFileInto(&cfg, cfgFile)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
