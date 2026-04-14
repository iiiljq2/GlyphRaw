package setup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"glyphraw/internal/config"
	"glyphraw/internal/logger"
	"glyphraw/pkg/download"
)

// Manager handles setup and initialization of required resources.
type Manager struct {
	logger     logger.Logger
	config     *config.Config
	downloader *download.Downloader
}

// NewManager creates a new setup manager.
func NewManager(l logger.Logger, cfg *config.Config) *Manager {
	return &Manager{
		logger:     l,
		config:     cfg,
		downloader: download.NewDownloader(l),
	}
}

// IsReady checks if the system is ready for font generation.
func (sm *Manager) IsReady() bool {
	// Check if Docker image exists
	imageName := sm.config.GetModel("fontdiffuser").DockerImage
	out, err := exec.Command("docker", "images", "-q", imageName).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return false
	}

	// Check for core model weights
	checkpointPath := sm.config.GetModel("fontdiffuser").CheckpointPath
	checkpointFile := filepath.Join(sm.config.CheckpointsDir, checkpointPath)
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		return false
	}

	// Check for reference assets
	if _, err := os.Stat(filepath.Join(sm.config.AssetsDir, "content_refs")); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetupAll runs the full initialization sequence.
func (sm *Manager) SetupAll() error {
	sm.logger.Info("Starting setup process...")

	if err := sm.downloadCheckpoints(); err != nil {
		return fmt.Errorf("failed to download checkpoints: %w", err)
	}

	if err := sm.syncContentRefs(); err != nil {
		return fmt.Errorf("failed to sync content refs: %w", err)
	}

	if err := sm.buildDockerImage(); err != nil {
		return fmt.Errorf("failed to build Docker image: %w", err)
	}

	sm.logger.Info("Setup complete!")
	return nil
}

// downloadCheckpoints downloads model weights from remote storage.
func (sm *Manager) downloadCheckpoints() error {
	sm.logger.Info("Downloading model weights...")

	model := sm.config.GetModel("fontdiffuser")
	ckptDir := filepath.Join(sm.config.CheckpointsDir, "fontdiffuser")

	if err := os.MkdirAll(ckptDir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	for _, file := range model.Files {
		dest := filepath.Join(ckptDir, file.Name)
		finalPath := filepath.Join(ckptDir, strings.TrimSuffix(file.Name, ".zip")+".pth")

		// Skip if file already exists
		if _, err := os.Stat(finalPath); err == nil {
			sm.logger.Debug("Model file already exists: %s", file.Name)
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			sm.logger.Debug("Model file already exists: %s", file.Name)
			continue
		}

		sm.logger.Info("Downloading: %s", file.Name)
		if err := sm.downloader.DownloadFile(file.URL, dest); err != nil {
			return fmt.Errorf("failed to download %s: %w", file.Name, err)
		}
	}

	// Extract unet.zip if it exists
	zipPath := filepath.Join(ckptDir, "unet.zip")
	if _, err := os.Stat(zipPath); err == nil {
		sm.logger.Info("Extracting: unet.zip")
		if err := unzipFile(zipPath, ckptDir); err != nil {
			return fmt.Errorf("failed to unzip: %w", err)
		}
		os.Remove(zipPath)
	}

	sm.logger.Info("Model weights ready")
	return nil
}

// syncContentRefs downloads and extracts content reference library.
func (sm *Manager) syncContentRefs() error {
	sm.logger.Info("Syncing content reference library...")

	targetZip := filepath.Join(sm.config.AssetsDir, "content_refs.zip")
	extractTo := filepath.Join(sm.config.AssetsDir, "content_refs")

	if err := os.MkdirAll(sm.config.AssetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Check if content refs already exist
	if files, _ := os.ReadDir(extractTo); len(files) > 100 {
		sm.logger.Info("Content library already exists")
		return nil
	}

	sm.logger.Info("Downloading content references...")
	if err := sm.downloader.DownloadFile(config.ContentRefsURL, targetZip); err != nil {
		return fmt.Errorf("failed to download content refs: %w", err)
	}

	sm.logger.Info("Extracting content references...")
	if err := smartUnzipContentRefs(targetZip, extractTo); err != nil {
		return fmt.Errorf("failed to extract content refs: %w", err)
	}

	os.Remove(targetZip)
	sm.logger.Info("Content references synced")
	return nil
}

// buildDockerImage builds the FontDiffuser Docker image.
func (sm *Manager) buildDockerImage() error {
	imageName := sm.config.GetModel("fontdiffuser").DockerImage
	sm.logger.Info("Building Docker image: %s", imageName)

	dockerfilePath := filepath.Join(sm.config.InstallDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found at %s", dockerfilePath)
	}

	// Pre-fetch pip packages if Python is available
	if pythonExe := getPythonExe(); pythonExe != "" {
		sm.logger.Info("Pre-fetching Python dependencies...")
		if err := sm.preFetchDependencies(pythonExe); err != nil {
			sm.logger.Warn("Pre-fetch failed, will fallback to Docker: %v", err)
		}
	}

	cmd := exec.Command("docker", "build", "-t", imageName, sm.config.InstallDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build failed: %w", err)
	}

	sm.logger.Info("Docker image built successfully")
	return nil
}

// preFetchDependencies pre-fetches Python dependencies.
func (sm *Manager) preFetchDependencies(pythonExe string) error {
	pkgDir := filepath.Join(sm.config.InstallDir, "pip_packages")
	os.RemoveAll(pkgDir)

	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("failed to create pip_packages directory: %w", err)
	}
	defer os.RemoveAll(pkgDir)

	args := []string{
		"-m", "pip", "download",
		"--dest", pkgDir,
		"--platform", "manylinux_2_17_x86_64",
		"--platform", "manylinux2014_x86_64",
		"--platform", "any",
		"--python-version", "3.10",
		"--only-binary=:all:",
	}

	// Parse pyproject.toml for dependencies
	projPath := filepath.Join(sm.config.InstallDir, "pyproject.toml")
	content, err := os.ReadFile(projPath)
	if err != nil {
		sm.logger.Debug("pyproject.toml not found, using default dependencies")
		args = append(args, "huggingface_hub==0.19.4")
	} else {
		fileStr := string(content)
		startTag := "dependencies = ["
		startIdx := strings.Index(fileStr, startTag)

		if startIdx == -1 {
			args = append(args, "huggingface_hub==0.19.4")
		} else {
			remaining := fileStr[startIdx:]
			endIdx := strings.Index(remaining, "]")

			if endIdx != -1 {
				depZone := remaining[:endIdx]
				re := regexp.MustCompile(`"(.*?)"`)
				matches := re.FindAllStringSubmatch(depZone, -1)

				for _, match := range matches {
					if len(match) > 1 {
						args = append(args, match[1])
					}
				}
			}
		}
	}

	cmd := exec.Command(pythonExe, args...)
	cmd.Dir = sm.config.InstallDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// Utility functions

func getPythonExe() string {
	candidates := []string{"python", "python3"}
	if runtime.GOOS == "windows" {
		candidates = []string{"python", "py"}
	}

	for _, name := range candidates {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func unzipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		os.MkdirAll(filepath.Dir(fpath), 0755)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
	}

	return nil
}

func smartUnzipContentRefs(zipPath, finalDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	os.RemoveAll(finalDir)
	os.MkdirAll(finalDir, 0755)

	for _, f := range r.File {
		rel := strings.Replace(f.Name, "content_refs/", "", 1)
		rel = strings.Replace(rel, "content_refs\\", "", 1)

		if rel == "" {
			continue
		}

		target := filepath.Join(finalDir, rel)
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, f.Mode())
			continue
		}

		os.MkdirAll(filepath.Dir(target), 0755)
		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			continue
		}

		io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
	}

	return nil
}
