package command

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"

	"github.com/Br0ce/boltcutter/store"
	"github.com/Br0ce/boltcutter/ui"
)

// openTimeout bounds the wait for the database file lock, so a database
// held by another process fails fast instead of hanging the terminal.
const openTimeout = 2 * time.Second

var (
	rootCmd = &cobra.Command{
		Use:   "boltcutter <database>",
		Short: "A TUI for sifting through your bbolt data",
		Long: `BoltCutter is a terminal UI for browsing bbolt databases.

It opens a bbolt database file and lets you navigate its buckets and
key/value pairs interactively, so you can inspect the contents of a
database without writing ad-hoc scripts to peek inside.`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return browse(args[0])
		},
	}
)

// browse opens the database read-only and hands it to the UI.
func browse(path string) error {
	db, err := bolt.Open(path, 0o600, &bolt.Options{
		ReadOnly: true,
		Timeout:  openTimeout,
	})
	if err != nil {
		return fmt.Errorf("open database %q: %w", path, err)
	}
	defer db.Close()

	return ui.Run(store.NewStore(db), path)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
