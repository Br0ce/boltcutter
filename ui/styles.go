package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Br0ce/boltcutter/value"
)

// Colors are adaptive so the UI stays readable on light and dark
// terminals alike.
var (
	colorAccent = lipgloss.AdaptiveColor{Light: "#5A3EBF", Dark: "#B694FF"}
	colorBucket = lipgloss.AdaptiveColor{Light: "#1A6B57", Dark: "#5FD7B0"}
	colorMuted  = lipgloss.AdaptiveColor{Light: "#6C6C6C", Dark: "#8A8A8A"}
	colorError  = lipgloss.AdaptiveColor{Light: "#B3261E", Dark: "#FF8A80"}

	// Colors of the value preview.
	colorKey     = lipgloss.AdaptiveColor{Light: "#1F5FBF", Dark: "#7FB2FF"}
	colorString  = lipgloss.AdaptiveColor{Light: "#2E7D32", Dark: "#A5D6A7"}
	colorNumber  = lipgloss.AdaptiveColor{Light: "#B26A00", Dark: "#FFC46B"}
	colorLiteral = lipgloss.AdaptiveColor{Light: "#8E24AA", Dark: "#E39BF0"}
)

type styles struct {
	box       lipgloss.Style
	activeBox lipgloss.Style

	item       lipgloss.Style
	bucketItem lipgloss.Style
	activeItem lipgloss.Style
	cursorItem lipgloss.Style
	empty      lipgloss.Style

	// Token styles of the value preview, one per value.Kind.
	tokenKey     lipgloss.Style
	tokenString  lipgloss.Style
	tokenNumber  lipgloss.Style
	tokenLiteral lipgloss.Style
	tokenPunct   lipgloss.Style

	header  lipgloss.Style
	appName lipgloss.Style
	label   lipgloss.Style
	dbPath  lipgloss.Style
	crumb   lipgloss.Style

	footer    lipgloss.Style
	footerKey lipgloss.Style
	errorText lipgloss.Style
}

func newStyles() styles {
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(0, 1)

	// The header and footer are single-line boxes framed like a column.
	bar := box.Height(1)

	// List rows carry no padding of their own: the box already pads,
	// and rows mark the cursor with a prefix instead.
	item := lipgloss.NewStyle()

	return styles{
		box:       box,
		activeBox: box.BorderForeground(colorAccent),

		item:       item,
		bucketItem: item.Foreground(colorBucket),
		// The selected row of an unfocused column stays marked, but
		// dimmed, so the reader keeps their place while moving on.
		activeItem: item.Foreground(colorMuted).Bold(true),
		cursorItem: item.Foreground(colorAccent).Bold(true).Reverse(true),
		empty:      item.Foreground(colorMuted).Italic(true),

		tokenKey:     item.Foreground(colorKey),
		tokenString:  item.Foreground(colorString),
		tokenNumber:  item.Foreground(colorNumber),
		tokenLiteral: item.Foreground(colorLiteral),
		tokenPunct:   item.Foreground(colorMuted),

		// The header and the footer are framed like the columns, so the
		// screen reads as one set of boxes.
		header:  bar,
		appName: lipgloss.NewStyle().Foreground(colorAccent).Bold(true),
		label:   lipgloss.NewStyle().Foreground(colorMuted),
		dbPath:  lipgloss.NewStyle().Foreground(colorMuted),
		crumb:   lipgloss.NewStyle().Foreground(colorBucket).Bold(true),

		footer:    bar.Foreground(colorMuted),
		footerKey: lipgloss.NewStyle().Foreground(colorAccent).Bold(true),
		errorText: lipgloss.NewStyle().Foreground(colorError),
	}
}

// token returns the style of one kind of value token. Kinds a format
// does not use simply never turn up.
func (s styles) token(kind value.Kind) lipgloss.Style {
	switch kind {
	case value.KindKey:
		return s.tokenKey
	case value.KindString:
		return s.tokenString
	case value.KindNumber:
		return s.tokenNumber
	case value.KindLiteral:
		return s.tokenLiteral
	case value.KindPunct:
		return s.tokenPunct
	case value.KindPlain:
		return s.item
	default:
		return s.item
	}
}
