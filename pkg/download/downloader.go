package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"glyphraw/internal/logger"
)

// Downloader handles file downloads with retry logic.
type Downloader struct {
	maxRetries int
	timeout    time.Duration
	logger     logger.Logger
}

// NewDownloader creates a new downloader with default settings.
func NewDownloader(logger logger.Logger) *Downloader {
	return &Downloader{
		maxRetries: 3,
		timeout:    30 * time.Second,
		logger:     logger,
	}
}

// DownloadFile downloads a file from URL to destination with retry logic.
func (d *Downloader) DownloadFile(url, dest string) error {
	var lastErr error

	for attempt := 1; attempt <= d.maxRetries; attempt++ {
		if attempt > 1 {
			d.logger.Warn("Retrying download (attempt %d/%d): %s", attempt, d.maxRetries, url)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		err := d.downloadFileOnce(url, dest)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed to download after %d attempts: %v", d.maxRetries, lastErr)
}

// downloadFileOnce attempts to download a file once.
func (d *Downloader) downloadFileOnce(url, dest string) error {
	client := &http.Client{
		Timeout: d.timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(dest)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
