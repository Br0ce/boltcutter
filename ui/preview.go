package ui

import (
	"fmt"

	"github.com/Br0ce/boltcutter/value"
)

// previewLimit is how much of a value the preview column reads. It
// shows a screenful at a time and a stored value may be megabytes, so
// reading one whole would be paid for on every press of a cursor key.
const previewLimit = 64 << 10

// preview is what the preview column shows: the listing of the node
// under the cursor, or the value of the leaf under it. At most one of
// the two is ever set, and neither is while the cursor is on nothing.
type preview struct {
	// column lists the node under the cursor. It is the column the
	// browser descends into, handed over rather than opened a second
	// time.
	column *column

	// lines are the decoded leaf under the cursor, and offset is how
	// far the column is scrolled through them.
	lines  []value.Line
	offset int

	// note stands in for the lines when there is nothing to show:
	// a value in a format we cannot read, or one too long to have
	// been read whole.
	note string
}

// close releases what the preview holds.
func (p *preview) close() error {
	return p.column.Close()
}

// setValue takes the bytes read for a leaf, which are the first size
// bytes of it, and works out what the column has to show.
func (p *preview) setValue(raw []byte, size int) {
	if decoded, ok := value.Decode(raw); ok {
		p.lines = decoded.Lines()

		return
	}
	// A value read short is unlikely to decode, so say that it was cut
	// rather than call a format unreadable on the strength of its
	// first page.
	if len(raw) < size {
		p.note = fmt.Sprintf("%s, too long to preview", humanSize(size))

		return
	}
	p.note = fmt.Sprintf("%s, in no format we can read", humanSize(size))
}

// humanSize spells a count of bytes the way a file listing does.
func humanSize(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := unit, 0
	for size := n / unit; size >= unit; size /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
