package font

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"glyphraw/internal/config"
	"glyphraw/internal/docker"
	"glyphraw/internal/logger"
)

// Generator orchestrates the font generation process.
type Generator struct {
	logger logger.Logger
	docker *docker.DockerRunner
	config *config.Config
	packer *Packer
}

// NewFontGenerator creates a new font generator.
func NewFontGenerator(l logger.Logger, d *docker.DockerRunner, cfg *config.Config) *Generator {
	return &Generator{
		logger: l,
		docker: d,
		config: cfg,
		packer: NewPacker(l, d, cfg.OutputDir),
	}
}

// GenerateFromImages generates fonts from style images.
func (fg *Generator) GenerateFromImages(styleInput string) error {
	// Collect style images
	styleImages, err := CollectStyleImages(styleInput)
	if err != nil {
		return fmt.Errorf("failed to collect images: %w", err)
	}

	fg.logger.Info("Found %d style image(s) to process", len(styleImages))

	// Get reference assets directory
	refsDir := filepath.Join(fg.config.AssetsDir, "content_refs")
	contentFiles, err := os.ReadDir(refsDir)
	if err != nil {
		return fmt.Errorf("reference assets missing at %s: %w", refsDir, err)
	}

	// Get absolute paths for Docker volume mounting
	absAssets, _ := filepath.Abs(fg.config.AssetsDir)
	absCheckpoints, _ := filepath.Abs(fg.config.CheckpointsDir)
	absOutput, _ := filepath.Abs(fg.config.OutputDir)
	absInputBase, _ := filepath.Abs(filepath.Dir(styleImages[0]))

	// Process each style image
	for _, stylePath := range styleImages {
		if err := fg.processStyle(stylePath, contentFiles, absAssets, absCheckpoints, absOutput, absInputBase); err != nil {
			return err
		}
	}

	return nil
}

// processStyle processes a single style image.
func (fg *Generator) processStyle(stylePath string, contentFiles []os.DirEntry, absAssets, absCheckpoints, absOutput, absInputBase string) error {
	styleName := strings.TrimSuffix(filepath.Base(stylePath), filepath.Ext(stylePath))
	styleOutputDir := filepath.Join(fg.config.OutputDir, styleName)

	if err := os.MkdirAll(styleOutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fg.logger.Info("Processing style: %s", styleName)
	successCount := 0

	// Process each content reference file
	for i, file := range contentFiles {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".png") {
			continue
		}

		charName := strings.TrimSuffix(file.Name(), ".png")
		charDir := filepath.Join(styleOutputDir, charName)

		if err := os.MkdirAll(charDir, 0755); err != nil {
			fg.logger.Error("Failed to create char directory for %s", charName)
			continue
		}

		fg.logger.Debug("[%d/%d] Generating: %s", i+1, len(contentFiles), charName)

		// Run inference for this character
		args := docker.InferenceArgs{
			StyleImagePath:   stylePath,
			ContentImagePath: fmt.Sprintf("/app/assets/content_refs/%s", file.Name()),
			OutputDir:        fmt.Sprintf("/output_data/%s/%s", styleName, charName),
			CheckpointDir:    fmt.Sprintf("%s/checkpoints", absAssets),
		}

		if err := fg.docker.RunInference(args); err != nil {
			fg.logger.Error("Failed to generate character %s: %v", charName, err)
			continue
		}

		// Rename output file on host
		src := filepath.Join(charDir, "out_single.png")
		dst := filepath.Join(charDir, charName+"_single.png")
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, dst); err != nil {
				fg.logger.Warn("Failed to rename output file for %s", charName)
			} else {
				successCount++
			}
		}
	}

	fg.logger.Info("Style %s: Generated %d images", styleName, successCount)

	// Pack to TTF
	ttfName := styleName + ".ttf"
	if err := fg.packer.PackToTTF(styleName, ttfName); err != nil {
		fg.logger.Error("Failed to pack font: %v", err)
	} else {
		ttfPath := filepath.Join(fg.config.OutputDir, ttfName)
		fg.logger.Info("Font saved: %s", ttfPath)
	}

	return nil
}
