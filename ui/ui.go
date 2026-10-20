package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Br0ce/boltcutter/tree"
)

// New returns a Model browsing t, with the root listing open. path and
// size describe the file it reads and are what the header shows of it;
// a size below zero is one the caller could not find out.
//
// Only a root that will not open is an error: anything that goes wrong
// further down is shown in the browser rather than raised, so a reader
// keeps what they were looking at.
//
// The Model holds listings open for as long as it lives, so the caller
// closes it when it is done.
func New(t tree.Tree, path string, size int64) (Model, error) {
	m := Model{
		tree:   t,
		dbPath: path,
		dbSize: size,
		styles: newStyles(),
		focus:  focusListing,
	}
	root, err := newColumn(t, nil, m.listHeight())
	if err != nil {
		return Model{}, err
	}
	m.columns = []*column{root}
	m.loadPreview()

	return m, nil
}

// Run starts the browser on t and blocks until the user quits.
func Run(t tree.Tree, path string, size int64) error {
	m, err := New(t, path, size)
	if err != nil {
		return err
	}

	// bubbletea hands the model back as it was left, which is the one
	// holding the listings of however deep the browsing went. The
	// model given to it is a copy, and closing that would leave every
	// listing opened since behind.
	final, runErr := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if closer, ok := final.(Model); ok {
		closer.Close()
	} else {
		m.Close()
	}

	return runErr
}
