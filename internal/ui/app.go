package ui

import (
	"image/color"
	"log"

	"github.com/golang-gui/goui/style"
	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	wui "github.com/golang-gui/goui/widgets/ui"
)

// AppID is the stable application identity.
const AppID = "com.audiamus.unseal"

// appState holds the durable state of the application.
type appState struct {
	// onboarding is the first-run wizard's state.
	onboarding *onboardingState
	// profile is the registration dialog's state.
	profile *profileState
	// showProfile says whether the registration dialog is open.
	showProfile bool
}

// Run starts the desktop application.
//
// It must be called from main, never from a background goroutine.
func Run() error {
	state := &appState{
		onboarding: &onboardingState{step: 0, total: 3},
		profile:    newProfileState(),
	}

	err := ui.Run(AppID, func(app ui.App) ui.RootView {
		var windows []ui.WindowView

		// The registration dialog floats above whatever else is open.
		if state.showProfile {
			windows = append(windows, profileWindow(state.profile, app))
		}

		// The onboarding wizard takes over the whole application until it
		// is finished, so the two never overlap.
		if state.onboarding != nil && !state.onboarding.completed {
			windows = append(windows, wizardWindow(state, app))
		} else {
			windows = append(windows, mainWindow(app))
		}

		return ui.Root().
			StyleSheet(buildSheet(app)).
			Windows(windows...)
	})
	if err != nil {
		log.Printf("unseal: %v", err)
	}
	return err
}

// buildSheet builds the style sheet: the modern theme plus the styles this
// application adds.
func buildSheet(app ui.App) style.StyleSheet {
	settings := app.Settings()
	dark := settings.ColorScheme() == ui.ColorSchemeDark

	fg := color.NRGBA{R: 40, G: 45, B: 52, A: 255}
	muted := color.NRGBA{R: 115, G: 120, B: 130, A: 255}
	cardBg := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	cardBorder := color.NRGBA{R: 222, G: 224, B: 230, A: 255}
	barBg := color.NRGBA{R: 235, G: 236, B: 240, A: 255}

	if dark {
		fg = color.NRGBA{R: 234, G: 236, B: 239, A: 255}
		muted = color.NRGBA{R: 150, G: 155, B: 165, A: 255}
		cardBg = color.NRGBA{R: 42, G: 44, B: 50, A: 255}
		cardBorder = color.NRGBA{R: 60, G: 62, B: 70, A: 255}
		barBg = color.NRGBA{R: 36, G: 38, B: 43, A: 255}
	}

	rules := modern.Rules(modern.Options{
		Dark:        dark,
		AccentColor: settings.AccentColor(),
		FontFamily:  settings.FontFamily(),
		FontSize:    settings.FontSize(),
	})
	rules = append(rules,
		style.Name("app-title").ForegroundColor(fg).FontSize(18),
		style.Name("section-title").ForegroundColor(fg).FontSize(13),
		style.Name("muted").ForegroundColor(muted),
		style.Name("card").BackgroundColor(cardBg).BorderColor(cardBorder).BorderWidth(1).Radius(8),
		style.Name("status-bar").BackgroundColor(barBg),
	)

	return style.Sheet(rules...)
}

// card wraps content in a bordered, rounded surface.
//
// A plain box does not paint a background; only a widget does, which is why
// the cards are border boxes.
func card(content ui.View) *ui.BorderBoxView {
	return ui.BorderBox(content).Style("card")
}

// mainWindow is the application's main window.
func mainWindow(app ui.App) ui.WindowView {
	return ui.Window("main").
		Title("Unseal").
		Chrome(ui.WindowChromeIntegrated).
		Size(1120, 760).
		MinSize(820, 560).
		Center(true).
		Content(mainView(app))
}

// mainView is the content of the main window.
func mainView(app ui.App) ui.View {
	return ui.VBox(
		headerBar(app),
		settingsRow(app),
		downloadQueue(app),
		progressRow(app),
		statusBar(app),
	).Spacing(0)
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

// settingsRow holds the three settings cards side by side.
func settingsRow(app ui.App) ui.View {
	return ui.HBox(
		profileGroup(app),
		downloadGroup(app),
		exportGroup(app),
	).Spacing(8).Padding(12)
}

// profileGroup is the profile settings card.
func profileGroup(app ui.App) ui.View {
	return card(
		ui.VBox(
			ui.Label("Profile").Style("section-title"),
			ui.Label("No profile selected").Style("muted"),
			ui.Button("Profiles…").Style(modern.Flat),
		).Spacing(6).Padding(12),
	)
}

// downloadGroup is the download settings card.
func downloadGroup(app ui.App) ui.View {
	return card(
		ui.VBox(
			ui.Label("Download").Style("section-title"),
			ui.Label("Quality: High").Style("muted"),
			ui.Label("Directory: (not set)").Style("muted"),
		).Spacing(6).Padding(12),
	)
}

// exportGroup is the export settings card.
func exportGroup(app ui.App) ui.View {
	return card(
		ui.VBox(
			ui.Label("Export").Style("section-title"),
			ui.Label("Format: .aax").Style("muted"),
			ui.Label("Directory: (not set)").Style("muted"),
		).Spacing(6).Padding(12),
	)
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
		card(
			ui.VBox(
				ui.Label("No items staged for download.").Style("muted"),
			).Padding(12),
		).MainWeight(1),
	).Spacing(6).Padding(12)
}

// progressRow holds the three progress bars and the Run/Abort buttons.
func progressRow(app ui.App) ui.View {
	return card(
		ui.VBox(
			ui.HBox(
				ui.Label("Download").Style("muted").MinWidth(72),
				wui.ProgressBar(0),
			).Spacing(8),
			ui.HBox(
				ui.Label("Decrypt").Style("muted").MinWidth(72),
				wui.ProgressBar(0),
			).Spacing(8),
			ui.HBox(
				ui.Label("Export").Style("muted").MinWidth(72),
				wui.ProgressBar(0),
			).Spacing(8),
			ui.HBox(
				ui.Button("Run").Style(modern.Primary),
				ui.Button("Abort").Style(modern.Flat),
			).Spacing(8),
		).Spacing(6).Padding(12),
	)
}

// statusBar is the bottom bar.
func statusBar(app ui.App) ui.View {
	return ui.BorderBox(
		ui.HBox(
			ui.Label("Ready").Style("muted"),
		).Padding(6).MinHeight(24),
	).Style("status-bar")
}
