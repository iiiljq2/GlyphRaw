package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var pythonScripts embed.FS

const (
	condaDirName = ".env_ai"
	fdDirName    = "FontDiffuser"
	condaEnvName = "fontdiffuser"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	exeDir := getExeDir()

	fmt.Println("==============================================")
	fmt.Println("       FontDiffuser Font Generator v0.6       ")
	fmt.Println("==============================================")

	condaExe, _ := detectConda(exeDir)
	hasCondaEnv := checkCondaEnv(condaExe, condaEnvName)
	fdDir, _ := detectFontDiffuser(exeDir)
	hasFD := fdDir != ""

	// Guard: Environment setup
	if condaExe == "" || !hasCondaEnv || !hasFD {
		fmt.Print("\n[System] Missing components. Install automatically? (Y/N): ")
		if !readYesNo(reader) {
			fmt.Println("Please setup manually.")
			waitExit(reader)
			return
		}

		sm := NewSetupManager(exeDir)
		if !hasFD {
			sm.downloadFontDiffuser()
			sm.downloadCheckpoints()
			sm.syncContentRefs()
			fdDir = sm.FDDir
		}
		if condaExe == "" {
			sm.installMiniconda()
			sm.configureCondaEnv()
			condaExe = filepath.Join(sm.CondaDir, getCondaMarker())
		}
		fmt.Println("[Success] Setup complete!")
	}

	fmt.Print("\nEnter path to handwritten image (file/folder): ")
	styleInput := readTrimmed(reader)
	if styleInput == "" {
		return
	}

	outputBase := filepath.Join(exeDir, "article_output")
	_ = os.MkdirAll(outputBase, 0755)

	if err := runArticleInference(condaExe, fdDir, styleInput, outputBase); err != nil {
		fmt.Printf("\n[Error] Task failed: %v\n", err)
	} else {
		fmt.Printf("\n[Success] All tasks done. Saved to: %s\n", outputBase)
	}

	waitExit(reader)
}

/**
* runArticleInference: Iterates style images and generates characters
* - condaExe: Path to conda
* - fdDir: FontDiffuser root
* - styleInput: Source style path
* - outputBase: Output folder
 */
func runArticleInference(condaExe, fdDir, styleInput, outputBase string) error {
	pyExe := getPythonExe(condaExe)
	if _, err := os.Stat(pyExe); err != nil {
		return fmt.Errorf("python missing: %s", pyExe)
	}

	styleImages, _ := collectStyleImages(styleInput)
	if len(styleImages) == 0 {
		return fmt.Errorf("no images found")
	}

	refsDir := filepath.Join(fdDir, "assets", "content_refs")
	contentFiles, err := os.ReadDir(refsDir)
	if err != nil {
		return fmt.Errorf("reference assets missing")
	}

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

			cmd := exec.Command(pyExe, "sample.py",
				"--style_image_path", stylePath,
				"--content_image_path", filepath.Join("assets", "content_refs", file.Name()),
				"--save_image_dir", charDir,
				"--save_image",
				"--ckpt_dir", "checkpoints/fontdiffuser",
				"--device", "cuda",
			)
			cmd.Dir = fdDir
			if cmd.Run() != nil {
				continue
			}

			// Rename output
			src := filepath.Join(charDir, "out_single.png")
			dst := filepath.Join(charDir, charName+"_single.png")
			if _, err := os.Stat(src); err == nil {
				os.Rename(src, dst)
				successCount++
			}
		}

		fmt.Printf("\n[Success] %s: Generated %d images.\n", styleName, successCount)

		// Pack to TTF
		ttfPath := filepath.Join(outputBase, styleName+".ttf")
		if err := packFontToTTF(pyExe, styleOutputDir, ttfPath); err != nil {
			fmt.Printf("[Error] Pack TTF failed: %v\n", err)
		} else {
			fmt.Printf("[Success] Font saved: %s\n", ttfPath)
		}
	}
	return nil
}

/**
* packFontToTTF: Converts character images into a font file
 */
func packFontToTTF(pyExe, inputDir, outputTtf string) error {
	script, _ := pythonScripts.ReadFile("scripts/pack_font.py")
	temp := filepath.Join(os.TempDir(), "temp_pack.py")
	os.WriteFile(temp, script, 0644)
	defer os.Remove(temp)

	cmd := exec.Command(pyExe, temp, inputDir, outputTtf)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, errBuf.String())
	}
	return nil
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

func getPythonExe(condaExe string) string {
	root := filepath.Dir(filepath.Dir(condaExe))
	if runtime.GOOS == "windows" {
		return filepath.Join(root, "envs", condaEnvName, "python.exe")
	}
	return filepath.Join(root, "envs", condaEnvName, "bin", "python")
}

func detectConda(exeDir string) (string, string) {
	if p, err := exec.LookPath("conda"); err == nil {
		return p, "Global"
	}
	local := filepath.Join(exeDir, condaDirName, getCondaMarker())
	if _, err := os.Stat(local); err == nil {
		return local, "Local"
	}
	return "", ""
}

func checkCondaEnv(condaExe, envName string) bool {
	out, _ := exec.Command(condaExe, "env", "list").Output()
	return strings.Contains(string(out), envName)
}

func detectFontDiffuser(exeDir string) (string, string) {
	local := filepath.Join(exeDir, fdDirName)
	if _, err := os.Stat(filepath.Join(local, "README.md")); err == nil {
		return local, "Local"
	}
	return "", ""
}

func getExeDir() string {
	exe, _ := os.Executable()
	return filepath.Dir(exe)
}

func getCondaMarker() string {
	if runtime.GOOS == "windows" {
		return filepath.Join("condabin", "conda.bat")
	}
	return filepath.Join("bin", "conda")
}

func printStatus(name, path string, ok bool) {
	if ok {
		fmt.Printf(" [Success] %s: %s\n", name, path)
	} else {
		fmt.Printf(" [Failure] %s: Not found\n", name)
	}
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
	r.ReadString('\n')
}
