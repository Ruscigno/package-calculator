package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	port    int
	dbPath  string
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "packcalc",
	Short: "Pack Calculator - Optimize order fulfillment",
	Long: `Pack Calculator is a service that calculates the optimal combination
of pack sizes to fulfill an order with minimum waste.

It provides both an API and a web UI for calculating pack distributions.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 8080, "Server port")
	rootCmd.PersistentFlags().StringVarP(&dbPath, "db", "d", "./data/packcalc.db", "Database file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose logging")
}
