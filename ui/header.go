package ui

import (
	"path"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// appName is what the header calls the program.
const appName = "boltcutter"

// Labels naming what the header shows, so neither string has to be
// guessed at from its shape alone.
const (
	bucketLabel = "bucket"
	dbLabel     = "db"
)

// header renders the bar above the panes: the program name and the bucket
// that is open on the left, the database file on the right. The file
// gives way first when the terminal is too narrow for both.
func (m Model) header() string {
	// The header style adds a border and a column of padding on either
	// side of its content.
	width := max(m.width-paneBorder-panePadding, 0)

	left := m.styles.appName.Render(appName) + "  " +
		m.styles.label.Render(bucketLabel+":") + " " +
		m.styles.crumb.Render(path.Join(m.path()...))

	// Whatever room the breadcrumb leaves goes to the database file,
	// which is truncated from the right to fit.
	room := width - lipgloss.Width(left) - len(dbLabel) - 3
	right := ""
	if path := truncate(m.dbPath, room); path != "" {
		right = m.styles.label.Render(dbLabel+":") + " " + m.styles.dbPath.Render(path)
	}

	gap := max(width-lipgloss.Width(left)-lipgloss.Width(right), 0)

	return m.styles.header.
		Width(max(m.width-paneBorder, 0)).
		Render(left + strings.Repeat(" ", gap) + right)
}
