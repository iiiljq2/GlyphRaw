package font

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CollectStyleImages collects style images from a file or directory.
// Returns a list of image file paths.
func CollectStyleImages(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, fmt.Errorf("failed to access path: %w", err)
	}

	var images []string

	// If input is a file, return it directly
	if !info.IsDir() {
		if isImageFile(input) {
			return []string{input}, nil
		}
		return nil, fmt.Errorf("file is not a supported image format: %s", input)
	}

	// If input is a directory, collect all image files
	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isImageFile(entry.Name()) {
			images = append(images, filepath.Join(input, entry.Name()))
		}
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no image files found in directory: %s", input)
	}

	return images, nil
}

// isImageFile checks if a file has a supported image extension.
func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supported := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".bmp":  true,
		".tiff": true,
	}
	return supported[ext]
}
