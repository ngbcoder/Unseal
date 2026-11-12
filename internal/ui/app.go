package ui

import (
	"image/color"
	"log"

	"github.com/golang-gui/goui/style"
	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	wui "github.com/golang-gui/goui/widgets/ui"

	"github.com/ngbcoder/Unseal/internal/audible"
)

// AppID is the stable application identity.
const AppID = "com.audiamus.unseal"

// appState holds the durable state of the application.
type appState struct {
	// onboarding is the first-run wizard's state.
	onboarding *onboardingState
	// profiles are the registered devices.
	profiles []*audible.Profile
}

// Run starts the desktop application.
//
// It must be called from main, never from a background goroutine.
func Run() error {
	state := &appState{onboarding: newOnboardingState()}

	err := ui.Run(AppID, func(app ui.App) ui.RootView {
		// The region is detected from the system on the first frame, once
		// the interface language is available.
		state.onboarding.detectRegion(app)

		// There is one window. The wizard replaces the main content until
		// the setup is finished, so nothing ever overlaps and the window
		// keeps its place in the taskbar and the alt-tab list.
		var content ui.View
		if state.onboarding.needsOnboarding() {
			content = wizardView(state.onboarding, app, state.onRegistered)
		} else {
			content = mainView(app, state)
		}

		return ui.Root().
			StyleSheet(buildSheet(app)).
			Windows(
				ui.Window("main").
					Title("Unseal").
					Chrome(ui.WindowChromeNative).
					Size(1120, 760).
					MinSize(760, 560).
					Center(true).
					Content(content),
			)
	})
	if err != nil {
		log.Printf("unseal: %v", err)
	}
	return err
}

// onRegistered is called when a device registration succeeds.
func (s *appState) onRegistered(p *audible.Profile) {
	s.profiles = append(s.profiles, p)
	if s.onboarding != nil && s.onboarding.step == 0 {
		s.onboarding.step = 1
	}
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

// mainView is the content of the main window.
func mainView(app ui.App, state *appState) ui.View {
	return ui.VBox(
		headerBar(app),
		settingsRow(app, state),
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
func settingsRow(app ui.App, state *appState) ui.View {
	return ui.HBox(
		profileGroup(app, state),
		downloadGroup(app),
		exportGroup(app),
	).Spacing(8).Padding(12)
}

// profileGroup is the profile settings card.
func profileGroup(app ui.App, state *appState) ui.View {
	name := "No profile selected"
	if len(state.profiles) > 0 {
		p := state.profiles[len(state.profiles)-1]
		name = p.DeviceInfo.Name
		if name == "" {
			name = p.CustomerInfo.AccountID
		}
	}

	return card(
		ui.VBox(
			ui.Label("Profile").Style("section-title"),
			ui.Label(name).Style("muted"),
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
				ui.Label("Download").Style("muted").MinWidth(80),
				wui.ProgressBar(0).MainWeight(1),
			).Spacing(8),
			ui.HBox(
				ui.Label("Decrypt").Style("muted").MinWidth(80),
				wui.ProgressBar(0).MainWeight(1),
			).Spacing(8),
			ui.HBox(
				ui.Label("Export").Style("muted").MinWidth(80),
				wui.ProgressBar(0).MainWeight(1),
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
