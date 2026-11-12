package ui

import (
	"github.com/golang-gui/goui/core/geometry"
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
//
// The wizard is shown when there is no profile, no download directory or no
// export decision. The check is conservative: a missing value means the
// wizard is needed.
func needsOnboarding() bool {
	// For now, always show onboarding until the settings are wired up.
	// Once the settings are connected, this will check the actual values.
	return true
}

// onboardingOverlay returns the wizard as a floating overlay item, or nil
// when onboarding is not needed.
func onboardingOverlay(state *onboardingState) *ui.OverlayItemView {
	if state.completed || !needsOnboarding() {
		return nil
	}
	return ui.OverlayItem(wizardCard(state)).Fill(true).OnPosition(func(available, size geometry.Size, position *geometry.Point) {
		// Centre the card.
		position.X = (available.Width - size.Width) / 2
		position.Y = (available.Height - size.Height) / 2
	})
}

// wizardCard is the wizard's floating card.
func wizardCard(state *onboardingState) ui.View {
	steps := []string{"Profile", "Download directory", "Export"}
	total := len(steps)

	title := "Welcome to Unseal"
	if state.step < total {
		title = "Step " + itoa(state.step+1) + " of " + itoa(total) + ": " + steps[state.step]
	}

	return ui.VBox(
		ui.Label(title).Style("section-title"),
		ui.Label("Set up Unseal in a few steps.").Style("muted"),
		wui.ProgressBar(float32(state.step+1)/float32(total)),
		wizardStepContent(state),
		ui.HBox(
			wizardBackButton(state),
			wizardNextButton(state, total),
		).Spacing(8),
	).Spacing(8).Padding(24).Style("card").MinWidth(400).MaxWidth(480)
}

// wizardStepContent returns the content for the current step.
func wizardStepContent(state *onboardingState) ui.View {
	switch state.step {
	case 0:
		return ui.VBox(
			ui.Label("Register an Audible device").Style("muted"),
			ui.Label("Unseal signs you in through your browser and registers itself as an Audible device. Your password never reaches the application.").Style("muted"),
			ui.Button("Register a profile…").Style(modern.Primary),
		).Spacing(6)
	case 1:
		return ui.VBox(
			ui.Label("Choose where downloaded files are written.").Style("muted"),
			ui.Button("Choose download directory…").Style(modern.Flat),
			ui.Label("(not set)").Style("muted"),
		).Spacing(6)
	case 2:
		return ui.VBox(
			ui.Label("Decide whether to export to the .aax format for AAX Audio Converter.").Style("muted"),
			ui.HBox(
				ui.Button("Yes, export").Style(modern.Primary),
				ui.Button("No, keep .m4b only").Style(modern.Flat),
			).Spacing(8),
		).Spacing(6)
	default:
		return ui.Label("Setup complete.").Style("muted")
	}
}

// wizardBackButton returns the Back button, or a spacer when on the first step.
func wizardBackButton(state *onboardingState) ui.View {
	if state.step == 0 {
		return ui.Label("").MainWeight(1)
	}
	return ui.Button("Back").Style(modern.Flat).OnClick(func() {
		if state.step > 0 {
			state.step--
		}
	})
}

// wizardNextButton returns the Next/Finish button.
func wizardNextButton(state *onboardingState, total int) ui.View {
	label := "Next"
	if state.step == total-1 {
		label = "Finish"
	}
	return ui.Button(label).Style(modern.Primary).OnClick(func() {
		if state.step < total-1 {
			state.step++
		} else {
			state.completed = true
		}
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
