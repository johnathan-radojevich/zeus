package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Adaptive palette — foreground colors only; terminal provides backgrounds.
var (
	colorAccent     = lipgloss.AdaptiveColor{Light: "30", Dark: "86"}
	colorAccentSoft = lipgloss.AdaptiveColor{Light: "37", Dark: "73"}
	colorText       = lipgloss.AdaptiveColor{Light: "236", Dark: "252"}
	colorMuted      = lipgloss.AdaptiveColor{Light: "245", Dark: "243"}
	colorSubtle     = lipgloss.AdaptiveColor{Light: "250", Dark: "238"}
	colorBorder     = lipgloss.AdaptiveColor{Light: "248", Dark: "239"}
	colorStatus     = lipgloss.AdaptiveColor{Light: "130", Dark: "214"}
	colorKey        = lipgloss.AdaptiveColor{Light: "27", Dark: "117"}
	colorOnAccent   = lipgloss.AdaptiveColor{Light: "255", Dark: "235"}
)

var (
	appStyle = lipgloss.NewStyle().Padding(0, 1)

	labelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccentSoft)

	activeSidebarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent).
				Foreground(colorText)

	inactiveSidebarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSubtle).
				Foreground(colorMuted)

	mainPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Foreground(colorText).
			Padding(1, 2)

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	footerSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorKey).
			Bold(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorStatus).
			Italic(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	emptyIconStyle = lipgloss.NewStyle().
			Foreground(colorAccentSoft)

	pathLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	pathBarStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	pathSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	pathCrumbStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	pathCurrentStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	policyDetailBarStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.NormalBorder(), false, false, true, true).
			BorderForeground(colorAccentSoft)

	policyDetailTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	policyDetailLabelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	policyDetailValueStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true)

	policyDetailSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)

	chevronStyle = lipgloss.NewStyle().
			Foreground(colorAccentSoft)

	loadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			Padding(1, 2)

	listNormalTitleStyle = lipgloss.NewStyle().
				Foreground(colorText)

	listDimmedTitleStyle = lipgloss.NewStyle().
				Foreground(colorSubtle)

	listSelectedTitleStyle = lipgloss.NewStyle().
				Foreground(colorOnAccent).
				Background(colorAccent).
				Bold(true)

	listFilterMatchStyle = lipgloss.NewStyle().
				Foreground(colorAccentSoft).
				Underline(true)

	requestSectionStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorSubtle).
				Padding(0, 1)

	requestSectionActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorAccent).
				Padding(0, 1)

	requestMetaStyle = lipgloss.NewStyle().
				Foreground(colorText)

	requestMethodStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	statusBadgeOKStyle = lipgloss.NewStyle().
				Foreground(colorOnAccent).
				Background(colorAccent).
				Bold(true).
				Padding(0, 1)

	statusBadgeWarnStyle = lipgloss.NewStyle().
				Foreground(colorOnAccent).
				Background(colorStatus).
				Bold(true).
				Padding(0, 1)

	responseBlockStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(colorSubtle).
				Foreground(colorMuted).
				Padding(0, 1)

	commandMenuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccentSoft).
			Foreground(colorText).
			Padding(1, 2)

	modalDialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Background(lipgloss.AdaptiveColor{Light: "255", Dark: "236"}).
			Foreground(colorText).
			Padding(2, 3)

	backdropDimStyle = lipgloss.NewStyle().
			Faint(true).
			Foreground(colorSubtle)
)

// initTheme queries the terminal so adaptive colors match system/terminal appearance.
func initTheme() {
	_ = termenv.HasDarkBackground()
	_ = lipgloss.HasDarkBackground()
}

// IsDarkMode reports whether the UI is rendering for a dark background.
func IsDarkMode() bool {
	return lipgloss.HasDarkBackground()
}
