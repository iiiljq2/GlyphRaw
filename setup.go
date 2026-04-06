package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SetupManager handles environment and source code installation.
// - InstallDir: Root path for the tool
// - CondaDir: Path to miniconda
// - FDDir: Path to FontDiffuser source
// - EnvName: Name of the conda environment
type SetupManager struct {
	InstallDir string
	CondaDir   string
	FDDir      string
	EnvName    string
}

func NewSetupManager(installDir string) *SetupManager {
	return &SetupManager{
		InstallDir: installDir,
		CondaDir:   filepath.Join(installDir, condaDirName),
		FDDir:      filepath.Join(installDir, fdDirName),
		EnvName:    condaEnvName,
	}
}

/**
 * downloadCheckpoints: Downloads necessary model weights from R2
 */
func (s *SetupManager) downloadCheckpoints() error {
	fmt.Println("\n[Setup] Downloading FontDiffuser weights...")

	ckptDir := filepath.Join(s.FDDir, "checkpoints", "fontdiffuser")
	_ = os.MkdirAll(ckptDir, 0755)

	baseURL := "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/"
	files := []string{"content_encoder.pth", "scr_210000.pth", "style_encoder.pth", "unet.zip"}

	for _, file := range files {
		url := baseURL + file
		dest := filepath.Join(ckptDir, file)
		fmt.Printf("  - Downloading %s...\n", file)

		err := downloadFile(url, dest)
		if err != nil {
			return fmt.Errorf("[Error] Failed to download %s: %v", file, err)
		}
	}

	zipPath := filepath.Join(ckptDir, "unet.zip")
	_, err := os.Stat(zipPath)
	if err != nil {
		fmt.Println("  [Success] Model weights ready.")
		return nil
	}

	fmt.Println("  - unet.zip detected, extracting...")
	if err := unzipFile(zipPath, ckptDir); err != nil {
		return fmt.Errorf("[Error] Unzip unet.zip failed: %v", err)
	}
	os.Remove(zipPath)

	fmt.Println("  [Success] Model weights ready.")
	return nil
}

/**
 * syncContentRefs: Syncs the standard character reference library
 */
func (s *SetupManager) syncContentRefs() error {
	fmt.Println("\n[Sync] Syncing content reference library...")

	assetsDir := filepath.Join(s.FDDir, "assets")
	targetZip := filepath.Join(assetsDir, "content_refs.zip")
	extractTo := filepath.Join(assetsDir, "content_refs")

	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("[Error] Create assets dir failed: %v", err)
	}

	if files, _ := os.ReadDir(extractTo); len(files) > 100 {
		fmt.Println("  - [Ready] Content library already exists.")
		return nil
	}

	url := "https://pub-3372efe59a304a619bc7bc0eec1c9817.r2.dev/content_refs.zip"
	if err := downloadFile(url, targetZip); err != nil {
		return fmt.Errorf("[Error] Download failed: %v", err)
	}

	fmt.Println("  - Extracting content references...")
	if err := smartUnzipContentRefs(targetZip, extractTo); err != nil {
		return fmt.Errorf("[Error] Unzip failed: %v", err)
	}
	_ = os.Remove(targetZip)

	files, _ := os.ReadDir(extractTo)
	if len(files) <= 100 {
		return fmt.Errorf("[Error] Abnormal file count after extraction")
	}

	fmt.Printf("  - [Success] Library synced: %d files found\n", len(files))
	return nil
}

/**
 * installMiniconda: Installs Miniconda runtime based on OS
 */
func (s *SetupManager) installMiniconda() error {
	fmt.Println("\n[Setup] Installing Miniconda...")
	_ = os.MkdirAll(s.InstallDir, 0755)

	url := "https://repo.anaconda.com/miniconda/Miniconda3-latest-Windows-x86_64.exe"
	installer := filepath.Join(os.TempDir(), "conda_installer.exe")
	if runtime.GOOS != "windows" {
		url = "https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh"
		installer = filepath.Join(os.TempDir(), "conda_installer.sh")
	}

	if err := downloadFile(url, installer); err != nil {
		return err
	}
	defer os.Remove(installer)

	cmd := exec.Command("bash", installer, "-b", "-p", s.CondaDir)
	if runtime.GOOS == "windows" {
		cmd = exec.Command(installer, "/S", "/InstallationType=JustMe", "/RegisterPython=0", "/AddToPath=0", "/D="+s.CondaDir)
	}
	return cmd.Run()
}

/**
 * downloadFontDiffuser: Clones the latest code from GitHub
 */
func (s *SetupManager) downloadFontDiffuser() error {
	if _, err := os.Stat(s.FDDir); err == nil {
		fmt.Println("  - Source code exists, skipping clone.")
		return nil
	}

	fmt.Println("\n[Setup] Cloning FontDiffuser...")
	cmd := exec.Command("git", "clone", "--depth=1", "https://github.com/yeungchenwa/FontDiffuser.git", s.FDDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

/**
 * configureCondaEnv: Creates Python env and installs AI dependencies
 */
func (s *SetupManager) configureCondaEnv() error {
	fmt.Println("\n[Config] Setting up environment...")
	condaExe := filepath.Join(s.CondaDir, getCondaMarker())

	createCmd := exec.Command(condaExe, "create", "-n", s.EnvName, "python=3.9", "-y")
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("[Error] Create env failed: %v", err)
	}

	pipExe := filepath.Join(s.CondaDir, "envs", s.EnvName, "Scripts", "pip.exe")
	if runtime.GOOS != "windows" {
		pipExe = filepath.Join(s.CondaDir, "envs", s.EnvName, "bin", "pip")
	}

	fmt.Println("  - Installing PyTorch...")
	torchCmd := exec.Command(pipExe, "install", "torch==2.0.1", "torchvision==0.15.2", "torchaudio==2.0.2", "--index-url", "https://download.pytorch.org/whl/cu118")
	if err := torchCmd.Run(); err != nil {
		return fmt.Errorf("[Error] Torch install failed: %v", err)
	}

	fmt.Println("  - Installing requirements...")
	reqPath := filepath.Join(s.FDDir, "requirements.txt")
	if err := exec.Command(pipExe, "install", "-r", reqPath).Run(); err != nil {
		return fmt.Errorf("[Error] Pip requirements failed: %v", err)
	}

	exec.Command(pipExe, "install", "huggingface_hub==0.19.4").Run()
	return nil
}

func downloadFile(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
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
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(fpath), 0755)
		outFile, _ := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		rc, _ := f.Open()
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

	_ = os.RemoveAll(finalDir)
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
		outFile, _ := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		rc, _ := f.Open()
		io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
	}
	return nil
}
