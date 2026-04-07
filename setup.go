package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// SetupManager handles the preparation of external resources on the host machine
type SetupManager struct {
	InstallDir     string
	AssetsDir      string
	CheckpointsDir string
}

// NewSetupManager initializes a new setup manager with the given installation directory
func NewSetupManager(installDir string) *SetupManager {
	return &SetupManager{
		InstallDir:     installDir,
		AssetsDir:      filepath.Join(installDir, "assets"),
		CheckpointsDir: filepath.Join(installDir, "checkpoints"),
	}
}

// IsReady checks if the Docker image and necessary model weights exist
func (s *SetupManager) IsReady() bool {
	// Check if Docker image exists
	out, err := exec.Command("docker", "images", "-q", dockerImageName).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return false
	}

	// Check for core model weights
	checkpointFile := filepath.Join(s.CheckpointsDir, "fontdiffuser", "unet", "diffusion_pytorch_model.bin")
	if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
		return false
	}

	// Check for reference assets
	if _, err := os.Stat(filepath.Join(s.AssetsDir, "content_refs")); os.IsNotExist(err) {
		return false
	}

	return true
}

// SetupAll runs the full initialization sequence
func (sm *SetupManager) SetupAll() error {
	fmt.Println("[System] Downloading required model weights and assets...")

	if err := sm.downloadCheckpoints(); err != nil {
		return fmt.Errorf("[Error] failed to download checkpoints: %v", err)
	}

	if err := sm.syncContentRefs(); err != nil {
		return fmt.Errorf("[Error] failed to sync content refs: %v", err)
	}

	pkgDir := filepath.Join(sm.InstallDir, "pip_packages")
	os.RemoveAll(pkgDir)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return fmt.Errorf("[Error] failed to create directory: %v", err)
	}
	defer os.RemoveAll(pkgDir)

	fmt.Println("[System] Pre-fetching dependencies to ensure stable installation...")

	pythonExe := sm.getPythonExe()
	if pythonExe == "" {
		fmt.Println("[Warning] Python not found in PATH. Skipping local pre-fetch.")
		return sm.buildDockerImage()
	}

	args := []string{
		"-m", "pip", "download",
		"--dest", pkgDir,
		"--platform", "manylinux_2_17_x86_64",
		"--platform", "manylinux2014_x86_64",
		"--platform", "any",
		"--python-version", "3.9",
		"--abi", "cp39",
		"--only-binary=:all:",
	}

	content, err := os.ReadFile(filepath.Join(sm.InstallDir, "pyproject.toml"))
	if err != nil {
		fmt.Println("[Warning] pyproject.toml not found. Only downloading default core.")
		args = append(args, "huggingface_hub==0.19.4")
		goto EXECUTE
	}

	{
		fileStr := string(content)
		startTag := "dependencies = ["
		startIdx := strings.Index(fileStr, startTag)
		if startIdx == -1 {
			fmt.Println("[Warning] No 'dependencies' section found in toml.")
			args = append(args, "huggingface_hub==0.19.4")
			goto EXECUTE
		}

		remaining := fileStr[startIdx:]
		endIdx := strings.Index(remaining, "]")
		if endIdx == -1 {
			fmt.Println("[Warning] Malformed dependencies section (missing ']').")
			goto EXECUTE
		}

		depZone := remaining[:endIdx]
		re := regexp.MustCompile(`"(.*?)"`)
		matches := re.FindAllStringSubmatch(depZone, -1)

		for _, match := range matches {
			if len(match) <= 1 {
				continue
			}
			args = append(args, match[1])
		}
	}

EXECUTE:
	cmd := exec.Command(pythonExe, args...)
	cmd.Dir = sm.InstallDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("[Warning] Local pip download failed (%v), will fallback to Docker.\n", err)
	}

	return sm.buildDockerImage()
}

func (s *SetupManager) buildDockerImage() error {
	fmt.Println("\n[Setup] Building Docker Image (using local cache)...")

	dockerfilePath := filepath.Join(s.InstallDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found in %s", s.InstallDir)
	}

	cmd := exec.Command("docker", "build", "-t", dockerImageName, s.InstallDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *SetupManager) downloadCheckpoints() error {
	fmt.Println("\n[Setup] Downloading FontDiffuser weights...")
	ckptDir := filepath.Join(s.CheckpointsDir, "fontdiffuser")
	_ = os.MkdirAll(ckptDir, 0755)

	baseURL := "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/"
	files := []string{"content_encoder.pth", "scr_210000.pth", "style_encoder.pth", "unet.zip"}

	for _, file := range files {
		dest := filepath.Join(ckptDir, file)
		finalPath := filepath.Join(ckptDir, strings.TrimSuffix(file, ".zip")+".pth")
		if _, err := os.Stat(finalPath); err == nil {
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			continue
		}

		fmt.Printf("  - Downloading %s...\n", file)
		if err := downloadFile(baseURL+file, dest); err != nil {
			return fmt.Errorf("failed to download %s: %v", file, err)
		}
	}

	zipPath := filepath.Join(ckptDir, "unet.zip")
	if _, err := os.Stat(zipPath); err != nil {
		fmt.Println("  [Success] Model weights ready.")
		return nil
	}

	fmt.Println("  - unet.zip detected, extracting...")
	if err := unzipFile(zipPath, ckptDir); err != nil {
		return fmt.Errorf("unzip unet.zip failed: %v", err)
	}
	_ = os.Remove(zipPath)

	fmt.Println("  [Success] Model weights ready.")
	return nil
}

/**
 * syncContentRefs: Syncs the standard character reference library
 */
func (s *SetupManager) syncContentRefs() error {
	fmt.Println("\n[Sync] Syncing content reference library...")
	targetZip := filepath.Join(s.AssetsDir, "content_refs.zip")
	extractTo := filepath.Join(s.AssetsDir, "content_refs")
	_ = os.MkdirAll(s.AssetsDir, 0755)

	if files, _ := os.ReadDir(extractTo); len(files) > 100 {
		fmt.Println("  - [Ready] Content library already exists.")
		return nil
	}

	url := "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/content_refs.zip"
	if err := downloadFile(url, targetZip); err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	fmt.Println("  - Extracting content references...")
	if err := smartUnzipContentRefs(targetZip, extractTo); err != nil {
		return fmt.Errorf("unzip failed: %v", err)
	}
	_ = os.Remove(targetZip)
	return nil
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
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
			_ = os.MkdirAll(fpath, f.Mode())
			continue
		}
		_ = os.MkdirAll(filepath.Dir(fpath), 0755)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		_, _ = io.Copy(outFile, rc)
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

	_ = os.RemoveAll(finalDir)
	_ = os.MkdirAll(finalDir, 0755)

	for _, f := range r.File {
		rel := strings.Replace(f.Name, "content_refs/", "", 1)
		rel = strings.Replace(rel, "content_refs\\", "", 1)
		if rel == "" {
			continue
		}

		target := filepath.Join(finalDir, rel)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, f.Mode())
			continue
		}

		_ = os.MkdirAll(filepath.Dir(target), 0755)
		outFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			continue
		}
		_, _ = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
	}
	return nil
}

func (sm *SetupManager) getPythonExe() string {
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
