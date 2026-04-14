package docker

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"glyphraw/internal/config"
	"glyphraw/internal/logger"
)

// DockerRunner handles Docker operations.
type DockerRunner struct {
	logger logger.Logger
	config *config.Config
}

// NewDockerRunner creates a new Docker runner.
func NewDockerRunner(l logger.Logger, cfg *config.Config) *DockerRunner {
	return &DockerRunner{
		logger: l,
		config: cfg,
	}
}

// CheckRunning checks if Docker daemon is running.
func (d *DockerRunner) CheckRunning() bool {
	err := exec.Command("docker", "info").Run()
	return err == nil
}

// CheckImageExists checks if a Docker image exists locally.
func (d *DockerRunner) CheckImageExists(imageName string) bool {
	out, err := exec.Command("docker", "images", "-q", imageName).Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// BuildImage builds a Docker image from a Dockerfile.
func (d *DockerRunner) BuildImage(imageName string) error {
	d.logger.Info("Building Docker image: %s", imageName)

	cmd := exec.Command("docker", "build", "-t", imageName, d.config.InstallDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	d.logger.Info("Docker image built successfully")
	return nil
}

// InferenceArgs holds arguments for inference execution.
type InferenceArgs struct {
	StyleImagePath   string
	ContentImagePath string
	OutputDir        string
	CheckpointDir    string
}

// RunInference executes inference inside Docker container.
func (d *DockerRunner) RunInference(args InferenceArgs) error {
	absAssets, _ := filepath.Abs(d.config.AssetsDir)
	absCheckpoints, _ := filepath.Abs(d.config.CheckpointsDir)
	absOutput, _ := filepath.Abs(d.config.OutputDir)
	absInputBase, _ := filepath.Abs(filepath.Dir(args.StyleImagePath))

	model := d.config.GetModel("fontdiffuser")

	dockerArgs := []string{
		"run", "--rm",
		"--gpus", "all",
		"-v", fmt.Sprintf("%s:/app/assets", absAssets),
		"-v", fmt.Sprintf("%s:/app/checkpoints", absCheckpoints),
		"-v", fmt.Sprintf("%s:/input_data:ro", absInputBase),
		"-v", fmt.Sprintf("%s:/output_data", absOutput),
		model.DockerImage,
		"python", "sample.py",
		"--style_image_path", fmt.Sprintf("/input_data/%s", filepath.Base(args.StyleImagePath)),
		"--content_image_path", args.ContentImagePath,
		"--save_image_dir", args.OutputDir,
		"--save_image",
		"--ckpt_dir", "/app/checkpoints/fontdiffuser",
		"--device", "cuda",
	}

	cmd := exec.Command("docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker inference failed: %w", err)
	}

	return nil
}

// RunPacking executes font packing inside Docker container.
func (d *DockerRunner) RunPacking(inputDir, outputFile string) error {
	absOutput, _ := filepath.Abs(d.config.OutputDir)

	model := d.config.GetModel("fontdiffuser")

	dockerArgs := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/output_data", absOutput),
		model.DockerImage,
		"python", "/output_data/temp_pack.py",
		fmt.Sprintf("/output_data/%s", filepath.Base(inputDir)),
		fmt.Sprintf("/output_data/%s", filepath.Base(outputFile)),
	}

	cmd := exec.Command("docker", dockerArgs...)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker packing failed: %s", errBuf.String())
	}

	return nil
}
