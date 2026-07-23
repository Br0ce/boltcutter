package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "boltcutter",
		Short: "A TUI for sifting through your bbolt data",
		Long: `BoltCutter is a terminal UI for browsing bbolt databases.

It opens a bbolt database file and lets you navigate its buckets and
key/value pairs interactively, so you can inspect the contents of a
database without writing ad-hoc scripts to peek inside.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Welcome to boltcutter!")
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
