package main

import (
	"fmt"
	"os"

	"github.com/lmarques/efx-face-manager/internal/tui"
	"github.com/spf13/cobra"
)

var version = "1.0.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "efx-face",
		Short:   "MLX Hugging Face Model Manager",
		Long:    `efx-face is a TUI tool for managing MLX Hugging Face models and launching mlx-openai-server instances.`,
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run()
		},
	}

	// Run command - launch model directly
	runCmd := &cobra.Command{
		Use:   "run [model]",
		Short: "Run a model directly",
		RunE: func(cmd *cobra.Command, args []string) error {
			model := ""
			if len(args) > 0 {
				model = args[0]
			}
			return tui.RunModel(model)
		},
	}

	// List command - list installed models
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed models",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunList()
		},
	}

	// Search command - search HuggingFace models
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search HuggingFace models",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return tui.RunSearch(query)
		},
	}

	// Servers command - manage running servers
	serversCmd := &cobra.Command{
		Use:   "servers",
		Short: "Manage running servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunServerManager()
		},
	}

	// Config command - configure storage path
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configure storage path and settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunConfig()
		},
	}

	// Install command - install a model from HuggingFace
	installCmd := &cobra.Command{
		Use:   "install <repo-id>",
		Short: "Install a model from HuggingFace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunInstall(args[0])
		},
	}

	// Uninstall command - uninstall a model
	uninstallCmd := &cobra.Command{
		Use:   "uninstall <model>",
		Short: "Uninstall a model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.RunUninstall(args[0])
		},
	}

	rootCmd.AddCommand(runCmd, listCmd, searchCmd, serversCmd, configCmd, installCmd, uninstallCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
