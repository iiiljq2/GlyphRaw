package font

import (
	"fmt"
	"os"
	"path/filepath"

	"glyphraw/internal/docker"
	"glyphraw/internal/logger"
)

// Packer handles font packing to TTF/OTF formats.
type Packer struct {
	logger    logger.Logger
	docker    *docker.DockerRunner
	outputDir string
}

// NewPacker creates a new font packer.
func NewPacker(l logger.Logger, d *docker.DockerRunner, outputDir string) *Packer {
	return &Packer{
		logger:    l,
		docker:    d,
		outputDir: outputDir,
	}
}

// PackToTTF packs generated font images into a TTF file.
func (p *Packer) PackToTTF(styleName, ttfName string) error {
	p.logger.Info("Packing font to TTF: %s", ttfName)

	// Read the Python packing script from the filesystem
	// The script should be at scripts/pack_font.py relative to the install directory
	scriptPath := filepath.Join(p.outputDir, "..", "..", "scripts", "pack_font.py")
	script, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read packing script at %s: %w", scriptPath, err)
	}

	tempScript := filepath.Join(p.outputDir, "temp_pack.py")
	if err := os.WriteFile(tempScript, script, 0644); err != nil {
		return fmt.Errorf("failed to write temp script: %w", err)
	}
	defer os.Remove(tempScript)

	// Run Docker packing command
	if err := p.docker.RunPacking(styleName, ttfName); err != nil {
		return fmt.Errorf("packing failed: %w", err)
	}

	p.logger.Info("Font packed successfully: %s", ttfName)
	return nil
}
