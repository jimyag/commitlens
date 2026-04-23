package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	webMode    bool
	webPort    int
	configFile string
)

var rootCmd = &cobra.Command{
	Use:   "commitlens",
	Short: "GitHub contribution stats viewer",
	RunE:  run,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.Flags().BoolVar(&webMode, "web", false, "Start web UI mode")
	rootCmd.Flags().IntVar(&webPort, "port", 8080, "Web UI port")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Config file path (default: ~/.commitlens/config.yaml)")
}

func run(cmd *cobra.Command, args []string) error {
	if webMode {
		fmt.Printf("Starting web UI on http://localhost:%d\n", webPort)
		return nil
	}
	fmt.Fprintln(os.Stderr, "TUI mode not yet implemented")
	return nil
}
