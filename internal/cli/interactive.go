package cli

import (
	"bufio"
	"fmt"
	"os"

	"glyphraw/internal/logger"
	"glyphraw/pkg/util"
)

// CLI handles command-line user interactions.
type CLI struct {
	logger logger.Logger
	reader *bufio.Reader
}

// NewCLI creates a new CLI handler.
func NewCLI(l logger.Logger) *CLI {
	return &CLI{
		logger: l,
		reader: bufio.NewReader(os.Stdin),
	}
}

// DisplayHeader displays the application header.
func (c *CLI) DisplayHeader() {
	fmt.Println("================================================================")
	fmt.Println("  ____ _             _     ____                ")
	fmt.Println(" / ___| |_   _ _ __ | |__ |  _ \\ __ ___      __")
	fmt.Println("| |  _| | | | | '_ \\| '_ \\| |_) / _` \\ \\ /\\ / /")
	fmt.Println("| |_| | | |_| | |_) | | | |  _ < (_| |\\ V  V / ")
	fmt.Println(" \\____|_|\\__, | .__/|_| |_|_| \\_\\__,_| \\_/\\_/  ")
	fmt.Println("         |___/|_|                              ")
	fmt.Println("================================================================")
}

// PromptDockerCheck checks if Docker is running and prompts if needed.
func (c *CLI) PromptDockerCheck() bool {
	if util.CheckDocker() {
		return true
	}

	c.logger.Error("Docker is not installed or not running")
	return false
}

// PromptSetupConfirmation asks the user if they want to auto-setup.
func (c *CLI) PromptSetupConfirmation() bool {
	fmt.Print("\n[System] Missing components (Docker image or assets). Setup automatically? (Y/N): ")
	return util.ReadYesNo(c.reader)
}

// PromptStyleImagePath asks the user for the path to style images.
func (c *CLI) PromptStyleImagePath() string {
	fmt.Print("\nEnter path to handwritten image (file or folder): ")
	return util.ReadTrimmed(c.reader)
}

// DisplayError displays an error message.
func (c *CLI) DisplayError(err error) {
	c.logger.Error("%v", err)
}

// DisplaySuccess displays a success message.
func (c *CLI) DisplaySuccess(msg string) {
	c.logger.Info(msg)
}

// WaitExit waits for user to press Enter before exiting.
func (c *CLI) WaitExit() {
	fmt.Print("\nDone. Press Enter to exit.")
	c.reader.ReadString('\n')
}
