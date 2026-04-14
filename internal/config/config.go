package config

import (
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	InstallDir     string
	AssetsDir      string
	CheckpointsDir string
	OutputDir      string
	Models         map[string]*ModelConfig
}

// LoadConfig creates a new config with paths based on the installation directory.
func LoadConfig(installDir string) *Config {
	return &Config{
		InstallDir:     installDir,
		AssetsDir:      filepath.Join(installDir, "assets"),
		CheckpointsDir: filepath.Join(installDir, "checkpoints"),
		OutputDir:      filepath.Join(installDir, "article_output"),
		Models: map[string]*ModelConfig{
			"fontdiffuser": FontDiffuserModel,
		},
	}
}

// GetModel returns the model config by name.
func (c *Config) GetModel(name string) *ModelConfig {
	if model, ok := c.Models[name]; ok {
		return model
	}
	return nil
}
