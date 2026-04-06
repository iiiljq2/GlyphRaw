package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var pythonScripts embed.FS

const (
	dockerImageName = "fontdiffuser-env:latest"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	exeDir := getExeDir()

	fmt.Println("================================================================")
	fmt.Println("  ____ _             _     ____                ")
	fmt.Println(" / ___| |_   _ _ __ | |__ |  _ \\ __ ___      __")
	fmt.Println("| |  _| | | | | '_ \\| '_ \\| |_) / _` \\ \\ /\\ / /")
	fmt.Println("| |_| | | |_| | |_) | | | |  _ < (_| |\\ V  V / ")
	fmt.Println(" \\____|_|\\__, | .__/|_| |_|_| \\_\\__,_| \\_/\\_/  ")
	fmt.Println("         |___/|_|                              ")
	fmt.Println("================================================================")

	if !checkDocker() {
		fmt.Println("[Error] Docker is not installed or not running. Please install/start Docker Desktop.")
		waitExit(reader)
		return
	}

	sm := NewSetupManager(exeDir)

	if !sm.IsReady() {
		fmt.Print("\n[System] Missing components (Docker image or assets). Setup automatically? (Y/N): ")
		if !readYesNo(reader) {
			fmt.Println("Please setup manually.")
			waitExit(reader)
			return
		}

		if err := sm.SetupAll(); err != nil {
			fmt.Printf("\n[Error] Setup failed: %v\n", err)
			waitExit(reader)
			return
		}
		fmt.Println("[Success] Setup complete!")
	}

	fmt.Print("\nEnter path to handwritten image (file or folder): ")
	styleInput := readTrimmed(reader)
	if styleInput == "" {
		return
	}

	outputBase := filepath.Join(exeDir, "article_output")
	_ = os.MkdirAll(outputBase, 0755)

	if err := runArticleInferenceDocker(sm, styleInput, outputBase); err != nil {
		fmt.Printf("\n[Error] Task failed: %v\n", err)
	} else {
		fmt.Printf("\n[Success] All tasks done. Saved to: %s\n", outputBase)
	}

	waitExit(reader)
}

// Docker Inference Logic
func runArticleInferenceDocker(sm *SetupManager, styleInput, outputBase string) error {
	styleImages, err := collectStyleImages(styleInput)
	if err != nil || len(styleImages) == 0 {
		return fmt.Errorf("no valid style images found in %s", styleInput)
	}

	refsDir := filepath.Join(sm.AssetsDir, "content_refs")
	contentFiles, err := os.ReadDir(refsDir)
	if err != nil {
		return fmt.Errorf("reference assets missing in %s", refsDir)
	}

	absAssets, _ := filepath.Abs(sm.AssetsDir)
	absCheckpoints, _ := filepath.Abs(sm.CheckpointsDir)
	absOutput, _ := filepath.Abs(outputBase)
	absInputBase, _ := filepath.Abs(filepath.Dir(styleImages[0]))

	for _, stylePath := range styleImages {
		styleName := strings.TrimSuffix(filepath.Base(stylePath), filepath.Ext(stylePath))
		styleOutputDir := filepath.Join(outputBase, styleName)
		_ = os.MkdirAll(styleOutputDir, 0755)

		fmt.Printf("\n--- Processing: %s ---\n", styleName)
		successCount := 0

		for i, file := range contentFiles {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".png") {
				continue
			}

			charName := strings.TrimSuffix(file.Name(), ".png")
			charDir := filepath.Join(styleOutputDir, charName)
			_ = os.MkdirAll(charDir, 0755)

			fmt.Printf("[%d/%d] Generating: %s\r", i+1, len(contentFiles), charName)

			// Container path mappings
			containerStylePath := fmt.Sprintf("/input_data/%s", filepath.Base(stylePath))
			containerContentPath := fmt.Sprintf("/app/assets/content_refs/%s", file.Name())
			containerSaveDir := fmt.Sprintf("/output_data/%s/%s", styleName, charName)

			// Build Docker command
			dockerArgs := []string{
				"run", "--rm",
				"--gpus", "all",
				"-v", fmt.Sprintf("%s:/app/assets", absAssets),
				"-v", fmt.Sprintf("%s:/app/checkpoints", absCheckpoints),
				"-v", fmt.Sprintf("%s:/input_data:ro", absInputBase),
				"-v", fmt.Sprintf("%s:/output_data", absOutput),
			}

			dockerArgs = append(dockerArgs,
				dockerImageName,
				"python", "sample.py",
				"--style_image_path", containerStylePath,
				"--content_image_path", containerContentPath,
				"--save_image_dir", containerSaveDir,
				"--save_image",
				"--ckpt_dir", "/app/checkpoints/fontdiffuser",
				"--device", "cuda",
			)

			cmd := exec.Command("docker", dockerArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				fmt.Printf("\n[Error] Docker failed for character %s: %v\n", charName, err)
				return err
			}

			// Rename output file on host
			src := filepath.Join(charDir, "out_single.png")
			dst := filepath.Join(charDir, charName+"_single.png")
			if _, err := os.Stat(src); err == nil {
				_ = os.Rename(src, dst)
				successCount++
			}
		}

		fmt.Printf("\n[Success] %s: Generated %d images.\n", styleName, successCount)

		// Pack to TTF via Docker
		ttfPath := filepath.Join(outputBase, styleName+".ttf")
		if err := packFontToTTFDocker(absOutput, styleName, styleName+".ttf"); err != nil {
			fmt.Printf("[Error] Pack TTF failed: %v\n", err)
		} else {
			fmt.Printf("[Success] Font saved: %s\n", ttfPath)
		}
	}
	return nil
}

func packFontToTTFDocker(absOutput, styleName, ttfName string) error {
	// Extract the embedded python script to the output directory so Docker can access it
	script, _ := pythonScripts.ReadFile("scripts/pack_font.py")
	tempScript := filepath.Join(absOutput, "temp_pack.py")
	_ = os.WriteFile(tempScript, script, 0644)
	defer os.Remove(tempScript)

	containerInputDir := fmt.Sprintf("/output_data/%s", styleName)
	containerOutputFile := fmt.Sprintf("/output_data/%s", ttfName)

	cmd := exec.Command("docker", "run", "--rm",
		"-v", fmt.Sprintf("%s:/output_data", absOutput),
		dockerImageName,
		"python", "/output_data/temp_pack.py", containerInputDir, containerOutputFile,
	)

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, errBuf.String())
	}
	return nil
}

// --- Utility Functions ---

func checkDocker() bool {
	_, err := exec.LookPath("docker")
	if err != nil {
		return false
	}
	// Check if Docker daemon is running
	err = exec.Command("docker", "info").Run()
	return err == nil
}

func collectStyleImages(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	var imgs []string
	if !info.IsDir() {
		imgs = append(imgs, input)
		return imgs, nil
	}
	files, _ := os.ReadDir(input)
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext == ".jpg" || ext == ".png" {
			imgs = append(imgs, filepath.Join(input, f.Name()))
		}
	}
	return imgs, nil
}

func getExeDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func readYesNo(r *bufio.Reader) bool {
	txt, _ := r.ReadString('\n')
	s := strings.ToUpper(strings.TrimSpace(txt))
	return s == "Y" || s == ""
}

func readTrimmed(r *bufio.Reader) string {
	txt, _ := r.ReadString('\n')
	return strings.TrimSpace(txt)
}

func waitExit(r *bufio.Reader) {
	fmt.Print("\nDone. Press Enter to exit.")
	_, _ = r.ReadString('\n')
}
