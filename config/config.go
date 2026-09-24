package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultConfigFileName = ".ssmshrc"

// Config holds the default shell configuration.
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

// ReadConfig reads the [default] section of an ssmsh configuration file.
func ReadConfig(filename string) (Config, error) {
	explicit := filename != ""
	if filename == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		filename = filepath.Join(homeDir, DefaultConfigFileName)
	}

	file, err := os.Open(filename)
	if os.IsNotExist(err) && !explicit {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	section := ""
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section != "default" {
				return Config{}, fmt.Errorf("%s:%d: unsupported section %q", filename, lineNumber, section)
			}
			continue
		}
		if section != "default" {
			return Config{}, fmt.Errorf("%s:%d: setting outside [default]", filename, lineNumber)
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("%s:%d: expected key=value", filename, lineNumber)
		}
		key, value = strings.ToLower(strings.TrimSpace(key)), strings.TrimSpace(value)
		switch key {
		case "decrypt":
			cfg.Default.Decrypt, err = strconv.ParseBool(value)
		case "overwrite":
			cfg.Default.Overwrite, err = strconv.ParseBool(value)
		case "key":
			cfg.Default.Key = value
		case "profile":
			cfg.Default.Profile = value
		case "region":
			cfg.Default.Region = value
		case "type":
			cfg.Default.Type = value
		case "output":
			cfg.Default.Output = strings.ToLower(value)
		default:
			return Config{}, fmt.Errorf("%s:%d: unknown setting %q", filename, lineNumber, key)
		}
		if err != nil {
			return Config{}, fmt.Errorf("%s:%d: invalid %s value: %w", filename, lineNumber, key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
