package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	// crumbSeparator divides the steps of the trail from the file down
	// to the open bucket. An arrow rather than a slash, so the trail
	// reads as steps taken and not as one name that happens to hold
	// slashes, which bolt keys and bucket names are free to do.
	crumbSeparator = " › "
	// logoName is the logo at the right end of the header.
	logoName = "boltcutter"
	// sizeSeparator divides the size of the file from the logo.
	sizeSeparator = " · "
	// groupGap is the least room left between the trail and the logo,
	// which are otherwise held apart by the width of the terminal.
	groupGap = 4
)

// unknownSize stands in for a file we were given no size for.
const unknownSize = "unknown"

// header renders the bar above the columns: on the left the trail
// from the file down to the bucket that is open, and on the right the
// size of the file beside the logo.
//
// The trail starts at the file because the file is the root of
// everything below it: one line then says both where we are and in
// what, and neither needs a label to be told apart from the other.
// The size and the logo hold the other end, where they sit still
// while the trail moves.
//
// The bar is framed like the footer, so the screen reads as one set
// of boxes rather than as boxes with a line of text left above them.
//
// What the terminal is too narrow to hold is given up from the right
// end first: the logo, then the size. The trail is the last to go, and even then
// it keeps its two ends, the file and the open bucket, and gives up
// the buckets between them.
func (m Model) header() string {
	room := max(m.width-boxBorder-boxPadding, 0)
	crumb, logo := m.headerEnds(room)
	gap := max(room-lipgloss.Width(crumb)-lipgloss.Width(logo), 0)

	return m.styles.bar.Width(max(m.width-boxBorder, 0)).Render(crumb + strings.Repeat(" ", gap) + logo)
}

// headerEnds renders what the header says as its two ends: the trail,
// and the size with the logo. The trail takes the room it needs, and
// the other end comes back as much of itself as fits beside it, which
// may be nothing. The logo goes before the size: it is the one of the
// two that tells the reader nothing they did not know.
func (m Model) headerEnds(room int) (string, string) {
	crumb := m.crumb(room)
	left := room - lipgloss.Width(crumb) - groupGap

	size := m.styles.dbSize.Render(formatSize(m.tree.Size()))
	if end := size + m.styles.separator.Render(sizeSeparator) + m.styles.logoName.Render(logoName); lipgloss.Width(end) <= left {
		return crumb, end
	}
	if lipgloss.Width(size) <= left {
		return crumb, size
	}

	return crumb, ""
}

// crumb renders the trail from the file down to the open bucket at no
// more than width cells. The buckets in its middle give way first,
// nearest the file first, and stand behind an ellipsis, so the trail
// keeps the two steps that say what it is: the file it starts from and
// the bucket it ends in. When even those do not fit, the file gives
// way too, and last of all the open bucket is cut from the left.
func (m Model) crumb(width int) string {
	steps := []string{m.styles.crumbFile.Render(printable(m.tree.Name()))}
	plain := []string{printable(m.tree.Name())}
	for _, name := range m.path() {
		steps = append(steps, m.styles.crumb.Render(printable(name)))
		plain = append(plain, printable(name))
	}

	sep := m.styles.separator.Render(crumbSeparator)
	// Drop the buckets after the file one at a time, never the last:
	// it is the bucket that is open.
	for dropped := 0; dropped < max(len(steps)-1, 1); dropped++ {
		shown := []string{steps[0]}
		if dropped > 0 {
			shown = append(shown, m.styles.separator.Render(ellipsis))
		}
		shown = append(shown, steps[1+dropped:]...)
		if line := strings.Join(shown, sep); lipgloss.Width(line) <= width {
			return line
		}
	}

	last := len(steps) - 1
	if last > 0 {
		line := m.styles.separator.Render(ellipsis) + sep + steps[last]
		if lipgloss.Width(line) <= width {
			return line
		}
	}
	style := m.styles.crumb
	if last == 0 {
		style = m.styles.crumbFile
	}

	return style.Render(truncateLeft(plain[last], width))
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
