package main

import (
	"glyphraw/internal/cli"
	"glyphraw/internal/config"
	"glyphraw/internal/docker"
	"glyphraw/internal/font"
	"glyphraw/internal/logger"
	"glyphraw/internal/setup"
	"glyphraw/pkg/util"
)

func main() {
	// Initialize logger
	log := logger.NewStdLogger()

	// Get executable directory
	exeDir, err := util.GetExeDir()
	if err != nil {
		log.Error("Failed to get executable directory: %v", err)
		return
	}

	// Load configuration
	cfg := config.LoadConfig(exeDir)

	// Initialize CLI
	cliApp := cli.NewCLI(log)

	// Display header
	cliApp.DisplayHeader()

	// Check Docker
	if !cliApp.PromptDockerCheck() {
		cliApp.WaitExit()
		return
	}

	// Initialize Docker runner
	dockerRunner := docker.NewDockerRunner(log, cfg)

	// Check if setup is needed
	setupMgr := setup.NewManager(log, cfg)
	if !setupMgr.IsReady() {
		if !cliApp.PromptSetupConfirmation() {
			log.Info("Setup skipped. Please setup manually.")
			cliApp.WaitExit()
			return
		}

		if err := setupMgr.SetupAll(); err != nil {
			cliApp.DisplayError(err)
			cliApp.WaitExit()
			return
		}

		cliApp.DisplaySuccess("Setup complete!")
	}

	// Ensure output directory exists
	if err := util.EnsureDir(cfg.OutputDir); err != nil {
		cliApp.DisplayError(err)
		cliApp.WaitExit()
		return
	}

	// Get user input for style images
	styleInput := cliApp.PromptStyleImagePath()
	if styleInput == "" {
		return
	}

	// Create font generator
	generator := font.NewFontGenerator(log, dockerRunner, cfg)

	// Generate fonts
	if err := generator.GenerateFromImages(styleInput); err != nil {
		cliApp.DisplayError(err)
	} else {
		cliApp.DisplaySuccess("All tasks done. Saved to: " + cfg.OutputDir)
	}

	cliApp.WaitExit()
}
