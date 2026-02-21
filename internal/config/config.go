package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

const AppVersion = "1.0.0"
const LegacyPrefixURL = "https://github.com/NichSchlagen/wemod-proton-launcher-go/releases/latest/download/prefix.zip"

type Config struct {
	Meta struct {
		ConfigPath string `toml:"-"`
	} `toml:"-"`
	General GeneralConfig `toml:"general"`
	Paths   PathsConfig   `toml:"paths"`
	Prefix  PrefixConfig  `toml:"prefix"`
}

type GeneralConfig struct {
	Interactive bool   `toml:"interactive"`
	LogLevel    string `toml:"log_level"`
	LogFile     string `toml:"log_file"`
}

type PathsConfig struct {
	WorkDir      string `toml:"work_dir"`
	WeModExePath string `toml:"wemod_exe_path"`
	PrefixDir    string `toml:"prefix_dir"`
	DownloadDir  string `toml:"download_dir"`
}

type PrefixConfig struct {
	DownloadURL string `toml:"download_url"`
}

func defaultConfigPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "wemod-launcher", "wemod.toml"), nil
}

func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}

	baseDir := filepath.Join(home, ".local", "share", "wemod-launcher")
	cfg := &Config{}
	cfg.General.Interactive = true
	cfg.General.LogLevel = "info"
	cfg.General.LogFile = filepath.Join(baseDir, "wemod-launcher.log")
	cfg.Paths.WorkDir = baseDir
	cfg.Paths.WeModExePath = filepath.Join(baseDir, "wemod_bin", "WeMod.exe")
	cfg.Paths.PrefixDir = filepath.Join(baseDir, "wemod_prefix")
	cfg.Paths.DownloadDir = filepath.Join(baseDir, "downloads")
	cfg.Prefix.DownloadURL = "auto"
	return cfg, nil
}

func LoadOrCreate(path string) (*Config, string, error) {
	var err error
	if path == "" {
		path, err = defaultConfigPath()
		if err != nil {
			return nil, "", err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, "", fmt.Errorf("create config dir: %w", err)
	}

	cfg, err := Default()
	if err != nil {
		return nil, "", err
	}

	if _, statErr := os.Stat(path); errors.Is(statErr, os.ErrNotExist) {
		if err := cfg.Save(path); err != nil {
			return nil, "", err
		}
		cfg.Meta.ConfigPath = path
		return cfg, path, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read config: %w", err)
	}
	if len(data) > 0 {
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, "", fmt.Errorf("parse config: %w", err)
		}
	}
	if cfg.Prefix.DownloadURL == "" || cfg.Prefix.DownloadURL == LegacyPrefixURL {
		cfg.Prefix.DownloadURL = "auto"
		_ = cfg.Save(path)
	}
	cfg.Meta.ConfigPath = path
	return cfg, path, nil
}

func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	content, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
