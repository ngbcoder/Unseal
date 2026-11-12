package ui

import (
	"log"

	"image/color"

	"github.com/golang-gui/goui/layout"
	"github.com/golang-gui/goui/style"
	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	wui "github.com/golang-gui/goui/widgets/ui"
)

// AppID is the stable application identity.
const AppID = "com.audiamus.unseal"

// appState holds the durable state of the application.
type appState struct {
	onboarding *onboardingState
}

// Run starts the desktop application.
//
// It must be called from main, never from a background goroutine.
func Run() error {
	state := &appState{onboarding: &onboardingState{step: 0, total: 3}}

	err := ui.Run(AppID, func(app ui.App) ui.RootView {
		settings := app.Settings()
		dark := settings.ColorScheme() == ui.ColorSchemeDark
		fg := color.NRGBA{R: 40, G: 45, B: 52, A: 255}
		if dark {
			fg = color.NRGBA{R: 234, G: 236, B: 239, A: 255}
		}
		rules := modern.Rules(modern.Options{Dark: dark, AccentColor: settings.AccentColor(), FontFamily: settings.FontFamily(), FontSize: settings.FontSize()})
		rules = append(rules,
			style.Name("app-title").ForegroundColor(fg).FontSize(18),
			style.Name("section-title").ForegroundColor(fg).FontSize(13),
			style.Name("muted").ForegroundColor(color.NRGBA{R: 120, G: 125, B: 135, A: 255}),
			style.Name("card").BackgroundColor(color.NRGBA{R: 255, G: 255, B: 255, A: 255}).Radius(8).BorderWidth(1).BorderColor(color.NRGBA{R: 220, G: 222, B: 228, A: 255}),
			style.Name("status-bar").BackgroundColor(color.NRGBA{R: 235, G: 236, B: 240, A: 255}),
		)
		if dark {
			rules[len(rules)-2] = style.Name("card").BackgroundColor(color.NRGBA{R: 42, G: 44, B: 50, A: 255}).Radius(8).BorderWidth(1).BorderColor(color.NRGBA{R: 60, G: 62, B: 70, A: 255})
		}
		sheet := style.Sheet(rules...)

		return ui.Root().
			StyleSheet(sheet).
			Windows(
				ui.Window("main").
					Title("Unseal").
					Chrome(ui.WindowChromeIntegrated).
					Size(1120, 760).
					MinSize(820, 560).
					Center(true).
					Content(mainView(app, state)),
			)
	})
	if err != nil {
		log.Printf("unseal: %v", err)
	}
	return err
}

// mainView is the content of the main window.
func mainView(app ui.App, state *appState) ui.View {
	mainContent := ui.VBox(
		headerBar(app),
		settingsRow(app),
		downloadQueue(app),
		progressRow(app),
		statusBar(app),
	).Spacing(0).CrossAlign(layout.CrossStretch)

	// The wizard is a floating overlay. When onboarding is complete it
	// disappears and the main content is shown underneath.
	if state.onboarding != nil && !state.onboarding.completed && needsOnboarding() {
		if item := onboardingOverlay(state.onboarding); item != nil {
			return ui.Overlay(mainContent, item)
		}
	}
	return mainContent
}

// headerBar is the top bar with the app title.
func headerBar(app ui.App) ui.View {
	return ui.HeaderBar(
		ui.HBox(
			ui.Label("Unseal").Style("app-title"),
			ui.HBox(
				ui.Button("Profiles…").Style(modern.Flat),
				ui.Button("Settings…").Style(modern.Flat),
			).Spacing(4),
		).Spacing(12).Padding(4),
	).Padding(6)
}

// settingsRow holds the three settings group boxes side by side.
func settingsRow(app ui.App) ui.View {
	return ui.HBox(
		profileGroup(app),
		downloadGroup(app),
		exportGroup(app),
	).Spacing(8).Padding(12).CrossAlign(layout.CrossStretch)
}

// profileGroup is the profile settings card.
func profileGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Profile").Style("section-title"),
		ui.Label("No profile selected").Style(modern.MutedText),
		ui.Button("Profiles…").Style(modern.Flat),
	).Spacing(6).Padding(12).Style("card")
}

// downloadGroup is the download settings card.
func downloadGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Download").Style("section-title"),
		ui.Label("Quality: High").Style(modern.MutedText),
		ui.Label("Directory: (not set)").Style(modern.MutedText),
	).Spacing(6).Padding(12).Style("card")
}

// exportGroup is the export settings card.
func exportGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Export").Style("section-title"),
		ui.Label("Format: .aax").Style(modern.MutedText),
		ui.Label("Directory: (not set)").Style(modern.MutedText),
	).Spacing(6).Padding(12).Style("card")
}

// downloadQueue is the centre panel with the download queue.
func downloadQueue(app ui.App) ui.View {
	return ui.VBox(
		ui.HBox(
			ui.Label("Download queue").Style("section-title"),
			ui.HBox(
				ui.Button("Download…").Style(modern.Flat),
				ui.Button("Remove").Style(modern.Flat),
			).Spacing(4),
		).Spacing(8),
		ui.ScrollView(
			ui.Label("No items staged for download.").Style(modern.MutedText),
		).Style("border-box").MinHeight(200),
	).Spacing(6).Padding(12).MainWeight(1)
}

// progressRow holds the three progress bars and the Run/Abort buttons.
func progressRow(app ui.App) ui.View {
	return ui.VBox(
		ui.HBox(
			ui.Label("Download").Style(modern.MutedText).MinWidth(72),
			wui.ProgressBar(0).MainWeight(1),
		).Spacing(8),
		ui.HBox(
			ui.Label("Decrypt").Style(modern.MutedText).MinWidth(72),
			wui.ProgressBar(0).MainWeight(1),
		).Spacing(8),
		ui.HBox(
			ui.Label("Export").Style(modern.MutedText).MinWidth(72),
			wui.ProgressBar(0).MainWeight(1),
		).Spacing(8),
		ui.HBox(
			ui.Button("Run").Style(modern.Primary),
			ui.Button("Abort").Style(modern.Flat),
		).Spacing(8),
	).Spacing(6).Padding(12).Style("card")
}

// statusBar is the bottom bar.
func statusBar(app ui.App) ui.View {
	return ui.HBox(
		ui.Label("Ready").Style(modern.MutedText),
	).Padding(6).MinHeight(24).Style("status-bar")
}
