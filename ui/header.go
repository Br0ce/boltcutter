package ui

import (
	"fmt"
	"path"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Labels naming the facts of the header, so neither has to be guessed
// at from its shape alone. The size carries no label of its own: no
// other fact looks anything like one.
const (
	bucketLabel = "at"
	dbLabel     = "db"
)

const (
	// factSeparator divides the facts of the header. It is wide because
	// the facts are short: the room around the dot is what keeps three
	// of them from reading as one sentence.
	factSeparator = "   ·   "
	// barPadding is the column of space a bar keeps at either end of
	// the terminal.
	barPadding = 1
	// groupGap is the least room left between the bucket and the file,
	// which are otherwise held apart by the width of the terminal.
	groupGap = 4
	// minPathWidth is the least a database path is worth showing at.
	// Below it the header drops the path rather than spend the room on
	// an ellipsis and a letter of a directory.
	minPathWidth = 8
	// minCrumbWidth is the least the bucket is worth showing at. The
	// size gives way rather than leave it any shorter.
	minCrumbWidth = 8
)

// unknownSize stands in for a file we were given no size for.
const unknownSize = "unknown"

// header renders the flat bar above the columns: the bucket that is
// open on the left, and on the right the file it is in and the size
// of that file.
//
// The bucket stands alone at one end because it is the fact that
// moves: a reader watching it browse looks at one place, not at
// wherever the file before it happened to end. The file and its size
// hold the other end, where they sit still. The program does not name
// itself here: the reader has just typed it, and the bar is for what
// changes.
//
// The bar carries no frame of its own. The columns below are framed,
// and a second row of boxes above them reads as a wall rather than as
// a bar: a header is chrome, and chrome is best kept quiet.
//
// What the terminal is too narrow to hold is given up in order of what
// it is worth: the database path first, then the size. The bucket is
// the last to go, because it is the one fact here that changes as the
// browser moves.
func (m Model) header() string {
	room := max(m.width-2*barPadding, 0)
	bucket, file := m.facts(room)
	gap := max(room-lipgloss.Width(bucket)-lipgloss.Width(file), 0)

	return m.styles.bar.Width(m.width).Render(bucket + strings.Repeat(" ", gap) + file)
}

// A fact is one item of the header bar: a muted label, and the value
// behind it in a style of the value's own.
type fact struct {
	label string
	value string
	style lipgloss.Style
}

// width is how many cells the fact takes, label and all.
func (f fact) width() int {
	if f.label == "" {
		return lipgloss.Width(f.value)
	}

	return lipgloss.Width(f.label) + 1 + lipgloss.Width(f.value)
}

// facts renders what the header says as its two ends: the bucket, and
// the file with its size. Each is shortened to what the room it is
// given leaves it, and an end with nothing left to say comes back
// empty.
func (m Model) facts(room int) (string, string) {
	bucket := fact{label: bucketLabel, value: m.crumb(), style: m.styles.crumb}
	size := fact{value: formatSize(m.db.Size), style: m.styles.dbSize}
	file := fact{label: dbLabel, value: m.db.Path, style: m.styles.dbPath}

	sep := lipgloss.Width(factSeparator)
	// What is left for the path once the bucket, the size, the label
	// of the path and the gaps between them have taken theirs. The
	// path is cut from the left: the file name is what tells two
	// databases apart, and it sits at the end.
	if left := room - bucket.width() - size.width() - sep - groupGap - lipgloss.Width(dbLabel) - 1; left >= minPathWidth {
		file.value = truncateLeft(file.value, left)

		return m.renderFacts(bucket), m.renderFacts(file, size)
	}
	// Without the path, the size is worth keeping whatever it costs
	// the bucket: seven cells buy a fact of their own, and the bucket
	// reads much the same one directory shorter.
	if left := room - size.width() - groupGap - lipgloss.Width(bucketLabel) - 1; left >= minCrumbWidth {
		bucket.value = truncateLeft(bucket.value, left)

		return m.renderFacts(bucket), m.renderFacts(size)
	}
	bucket.value = truncateLeft(bucket.value, max(room-lipgloss.Width(bucketLabel)-1, 0))

	return m.renderFacts(bucket), ""
}

// renderFacts styles the facts and sets them out along the bar.
func (m Model) renderFacts(facts ...fact) string {
	parts := make([]string, 0, len(facts))
	for _, f := range facts {
		text := f.style.Render(f.value)
		if f.label != "" {
			text = m.styles.label.Render(f.label) + " " + text
		}
		parts = append(parts, text)
	}

	return strings.Join(parts, m.styles.separator.Render(factSeparator))
}

// crumb names the bucket that is open, by the whole path from the root
// down to it. The root itself is spelled as a separator rather than
// left blank, so the bar says where we are instead of saying nothing.
func (m Model) crumb() string {
	if !m.hasParent() {
		return "/"
	}

	return printable(path.Join(m.path()...))
}

// formatSize spells a size in the largest unit that leaves a number
// worth reading. A size we were not given is said to be unknown, which
// is worth more than a confident nought.
func formatSize(n int64) string {
	if n < 0 {
		return unknownSize
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB"}
	size := float64(n) / unit
	i := 0
	for size >= unit && i < len(units)-1 {
		size /= unit
		i++
	}

	return fmt.Sprintf("%.1f %s", size, units[i])
}
