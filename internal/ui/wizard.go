package ui

import (
	"github.com/golang-gui/goui/gui"
	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	wui "github.com/golang-gui/goui/widgets/ui"
)

// onboardingState holds the state of the first-run wizard.
type onboardingState struct {
	// step is the current step (0-based).
	step int
	// total is the number of steps.
	total int
	// completed says whether onboarding is finished.
	completed bool
}

// needsOnboarding reports whether the first-run wizard should be shown.
func needsOnboarding() bool {
	// TODO: check the actual settings (profiles, download directory,
	// export decision) once they are wired up.
	return true
}

// wizardWindow is the first-run wizard.
//
// It is a window of its own, so it never overlaps the main window.
func wizardWindow(state *appState, app ui.App) ui.WindowView {
	return ui.Window("wizard").
		Title("Unseal — Setup").
		Chrome(ui.WindowChromeIntegrated).
		Size(620, 480).
		MinSize(480, 400).
		Center(true).
		SkipTaskbar(true).
		Content(wizardContent(state, app))
}

// wizardContent is the body of the wizard.
func wizardContent(state *appState, app ui.App) ui.View {
	wiz := state.onboarding
	steps := []string{"Profile", "Download directory", "Export"}
	total := len(steps)
	if wiz.step >= total {
		wiz.step = total - 1
	}

	return ui.VBox(
		ui.Label("Welcome to Unseal").Style("app-title"),
		ui.Label("Step "+itoa(wiz.step+1)+" of "+itoa(total)+": "+steps[wiz.step]).Style("section-title"),
		ui.Label("Set up Unseal in a few steps. You can change all of this later.").Style("muted").WrapMode(gui.WrapWordChar),
		wui.ProgressBar(float32(wiz.step+1)/float32(total)),
		card(wizardStepContent(state, app)).MainWeight(1),
		ui.HBox(
			wizardBackButton(state, app),
			wizardNextButton(state, app, total),
		).Spacing(8),
	).Spacing(10).Padding(20)
}

// wizardStepContent returns the content for the current step.
func wizardStepContent(state *appState, app ui.App) ui.View {
	switch state.onboarding.step {
	case 0:
		return ui.VBox(
			ui.Label("Register an Audible device").Style("section-title"),
			ui.Label("Unseal signs you in through your browser and registers itself as an Audible device. Your password never reaches the application.").Style("muted").WrapMode(gui.WrapWordChar),
			ui.Button("Register a profile…").Style(modern.Primary).OnClick(func() {
				state.showProfile = true
				app.RequestUpdate()
			}),
		).Spacing(8).Padding(16)
	case 1:
		return ui.VBox(
			ui.Label("Choose where downloaded files are written").Style("section-title"),
			ui.Label("The decrypted .m4b files and the encrypted .aaxc payloads are written here.").Style("muted").WrapMode(gui.WrapWordChar),
			ui.Button("Choose download directory…").Style(modern.Flat).OnClick(func() {
				app.FileDialog().OpenDirectory(ui.DialogOptions{Title: "Choose download directory"}, func(accesses []ui.FileAccess, err error) {
					if err != nil || len(accesses) == 0 {
						return
					}
					path, err := accesses[0].LocalPath()
					if err != nil {
						return
					}
					// TODO: store the download directory in the settings.
					_ = path
					app.RequestUpdate()
				})
			}),
			ui.Label("Directory: (not set)").Style("muted"),
		).Spacing(8).Padding(16)
	case 2:
		return ui.VBox(
			ui.Label("Choose an export format").Style("section-title"),
			ui.Label("Exporting writes an .aax file plus sidecar JSON, which AAX Audio Converter turns into .mp3 or .m4a.").Style("muted").WrapMode(gui.WrapWordChar),
			ui.HBox(
				ui.Button("Yes, export to .aax").Style(modern.Primary).OnClick(func() {
					// TODO: set exportToAax to true.
					app.RequestUpdate()
				}),
				ui.Button("No, keep .m4b only").Style(modern.Flat).OnClick(func() {
					// TODO: set exportToAax to false.
					app.RequestUpdate()
				}),
			).Spacing(8),
		).Spacing(8).Padding(16)
	default:
		return ui.Label("Setup complete.").Style("muted")
	}
}

// wizardBackButton returns the Back button, or a spacer when on the first step.
func wizardBackButton(state *appState, app ui.App) ui.View {
	if state.onboarding.step == 0 {
		return ui.Label("").MainWeight(1)
	}
	return ui.Button("Back").Style(modern.Flat).OnClick(func() {
		if state.onboarding.step > 0 {
			state.onboarding.step--
			app.RequestUpdate()
		}
	})
}

// wizardNextButton returns the Next/Finish button.
func wizardNextButton(state *appState, app ui.App, total int) ui.View {
	label := "Next"
	if state.onboarding.step == total-1 {
		label = "Finish"
	}
	return ui.Button(label).Style(modern.Primary).OnClick(func() {
		if state.onboarding.step < total-1 {
			state.onboarding.step++
		} else {
			state.onboarding.completed = true
		}
		app.RequestUpdate()
	})
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
