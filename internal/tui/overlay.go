package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

func dimContent(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.TrimSpace(ansi.Strip(line)) == "" {
			continue
		}
		lines[i] = backdropDimStyle.Render(ansi.Strip(line))
	}
	return strings.Join(lines, "\n")
}

func renderModalOverlay(background, dialog string, screenWidth int) string {
	if screenWidth < 1 {
		screenWidth = 1
	}

	dimmed := dimContent(background)
	dialogWidth := lipgloss.Width(dialog)
	dialogHeight := lipgloss.Height(dialog)
	bgLines := strings.Split(dimmed, "\n")
	bgHeight := len(bgLines)

	startY := (bgHeight - dialogHeight) / 2
	if startY < 0 {
		startY = 0
	}
	startX := (screenWidth - dialogWidth) / 2
	if startX < 0 {
		startX = 0
	}

	fgLines := strings.Split(dialog, "\n")
	for i, fgLine := range fgLines {
		row := startY + i
		if row < 0 || row >= len(bgLines) {
			break
		}
		bgLines[row] = pasteLine(bgLines[row], fgLine, startX, screenWidth)
	}
	return strings.Join(bgLines, "\n")
}

func pasteLine(background, foreground string, x, width int) string {
	bgPlain := ansi.Strip(background)
	if runewidth.StringWidth(bgPlain) < width {
		bgPlain += strings.Repeat(" ", width-runewidth.StringWidth(bgPlain))
	}

	left := truncateVisual(bgPlain, 0, x)
	fgWidth := lipgloss.Width(foreground)
	right := truncateVisual(bgPlain, x+fgWidth, width-(x+fgWidth))

	leftStyled := backdropDimStyle.Render(left)
	if right != "" {
		return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, foreground, backdropDimStyle.Render(right))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, foreground)
}

func truncateVisual(s string, start, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if start < 0 {
		start = 0
	}
	seen := 0
	out := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if seen+rw <= start {
			seen += rw
			continue
		}
		if out+rw > maxWidth {
			break
		}
		b.WriteRune(r)
		seen += rw
		out += rw
	}
	return b.String()
}
