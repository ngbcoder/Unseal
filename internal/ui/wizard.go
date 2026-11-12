package ui

import (
	"context"
	"fmt"

	"github.com/golang-gui/goui/gui"
	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	"github.com/golang-gui/goui/widgets"
	wui "github.com/golang-gui/goui/widgets/ui"

	"github.com/ngbcoder/Unseal/internal/audible"
	"github.com/ngbcoder/Unseal/internal/types"
)

// onboardingState holds the state of the first-run wizard.
type onboardingState struct {
	// step is the current step (0-based).
	step int
	// total is the number of steps.
	total int
	// completed says whether onboarding is finished.
	completed bool

	// regionIndex is the selected region.
	regionIndex int
	// regions is the dropdown model.
	regions *gui.SliceListModel[dropDownItem]

	// signInURL is the generated Amazon or Audible sign-in URL.
	signInURL ui.State[string]
	// redirectURL is the URL the user pasted back from the browser.
	redirectURL ui.State[string]
	// status carries the last message.
	status ui.State[string]
	// busy says whether a registration is in flight.
	busy ui.State[bool]

	// login is the generated login data.
	login audible.Login
	// detected says whether the region was detected from the system.
	detected bool
}

// dropDownItem is one entry of the region dropdown.
type dropDownItem = widgets.DropDownItem

// newOnboardingState creates the wizard state.
//
// The region is detected from the system and preselected, so the common case
// needs no interaction.
func newOnboardingState() *onboardingState {
	items := make([]widgets.DropDownItem, 0, len(types.AllRegions))
	for _, r := range types.AllRegions {
		items = append(items, widgets.DropDownItem{Text: string(r)})
	}

	s := &onboardingState{
		total:       3,
		regions:     gui.NewSliceListModel(items),
		signInURL:   ui.MakeState(""),
		redirectURL: ui.MakeState(""),
		status:      ui.MakeState(""),
		busy:        ui.MakeState(false),
	}
	s.detectRegion(nil)
	return s
}

// needsOnboarding reports whether the first-run wizard should be shown.
func (o *onboardingState) needsOnboarding() bool {
	// TODO: check the actual settings (profiles, download directory,
	// export decision) once they are wired up.
	return !o.completed
}

// detectRegion preselects the region that matches the system.
//
// The format locale follows the user's geography, so it is tried first; the
// interface language is the fallback.
func (o *onboardingState) detectRegion(app ui.App) {
	if o.detected {
		return
	}
	o.detected = true

	candidates := []string{audible.SystemLocale()}
	if app != nil {
		candidates = append(candidates, app.Settings().Language())
	}

	region, ok := audible.DetectRegion(candidates...)
	if !ok {
		return
	}

	for i, r := range types.AllRegions {
		if r == region {
			o.regionIndex = i
			return
		}
	}
}

// currentRegion returns the selected region.
func (o *onboardingState) currentRegion() types.Region {
	regions := types.AllRegions
	if o.regionIndex < 0 || o.regionIndex >= len(regions) {
		return regions[0]
	}
	return regions[o.regionIndex]
}

// createSignInURL generates the sign-in URL.
func (o *onboardingState) createSignInURL() {
	o.login = audible.NewLogin(string(o.currentRegion()), false)
	o.signInURL.Set(audible.BuildAuthURI(o.login, ""))
	o.status.Set("Sign-in URL created. Open it in your browser to sign in.")
}

// register performs the device registration.
//
// onSuccess is called on the interface goroutine when the registration
// succeeds.
func (o *onboardingState) register(app ui.App, onSuccess func(*audible.Profile)) {
	if o.login.Serial == "" {
		o.status.Set("Create the sign-in URL first.")
		app.RequestUpdate()
		return
	}

	raw := o.redirectURL.Get()
	if raw == "" {
		o.status.Set("Paste the final URL from the browser.")
		app.RequestUpdate()
		return
	}

	code, err := audible.ParseRedirectURL(raw)
	if err != nil {
		o.status.Set("That does not look like a redirect URL: " + err.Error())
		app.RequestUpdate()
		return
	}

	region := o.currentRegion()
	o.busy.Set(true)
	o.status.Set("Registering...")
	app.RequestUpdate()

	// The registration is a network call, so it runs off the interface
	// goroutine and posts the result back.
	go func() {
		ctx := context.Background()
		profile, err := audible.Register(ctx, string(region), o.login, code, false)

		app.Post(func() {
			o.busy.Set(false)
			if err != nil {
				o.status.Set("Registration failed: " + err.Error())
			} else {
				o.status.Set(fmt.Sprintf("Registered %s (account %s).", profile.DeviceInfo.Name, profile.CustomerInfo.AccountID))
				if onSuccess != nil {
					onSuccess(profile)
				}
			}
			app.RequestUpdate()
		})
	}()
}

// wizardView is the whole wizard.
func wizardView(o *onboardingState, app ui.App, onSuccess func(*audible.Profile)) ui.View {
	steps := []string{"Profile", "Download directory", "Export"}
	total := len(steps)
	if o.step >= total {
		o.step = total - 1
	}

	return ui.VBox(
		ui.Label("Welcome to Unseal").Style("app-title"),
		ui.Label("Step "+itoa(o.step+1)+" of "+itoa(total)+": "+steps[o.step]).Style("section-title"),
		ui.Label("Set up Unseal in a few steps. You can change all of this later.").Style("muted").WrapMode(gui.WrapWordChar),
		wui.ProgressBar(float32(o.step+1)/float32(total)),
		card(o.wizardStep(app, onSuccess)).MainWeight(1),
		ui.HBox(
			o.wizardBackButton(app),
			o.wizardNextButton(app, total),
		).Spacing(8),
	).Spacing(10).Padding(20)
}

// wizardStep returns the content for the current step.
func (o *onboardingState) wizardStep(app ui.App, onSuccess func(*audible.Profile)) ui.View {
	switch o.step {
	case 0:
		return o.registrationStep(app, onSuccess)
	case 1:
		return o.directoryStep(app)
	case 2:
		return o.exportStep(app)
	default:
		return ui.Label("Setup complete.").Style("muted")
	}
}

// registrationStep is the device registration step.
func (o *onboardingState) registrationStep(app ui.App, onSuccess func(*audible.Profile)) ui.View {
	return ui.VBox(
		ui.Label("Register an Audible device").Style("section-title"),
		ui.Label("Unseal signs you in through your browser and registers itself as an Audible device. Your password never reaches the application.").Style("muted").WrapMode(gui.WrapWordChar),

		ui.HBox(
			ui.Label("Region").Style("section-title").MinWidth(96),
			wui.DropDown(o.regions).Selected(o.regionIndex).OnSelected(func(i int) {
				o.regionIndex = i
				o.signInURL.Set("")
				app.RequestUpdate()
			}),
		).Spacing(8),

		ui.HBox(
			ui.Label("Sign-in URL").Style("section-title").MinWidth(96),
			ui.TextInput().BindText(&o.signInURL).ReadOnly(true).Placeholder("not created yet").MainWeight(1),
		).Spacing(8),

		ui.HBox(
			ui.Button("Create URL").OnClick(func() {
				o.createSignInURL()
				app.RequestUpdate()
			}),
			ui.Button("Copy URL").OnClick(func() {
				url := o.signInURL.Get()
				if url == "" {
					o.status.Set("Create the sign-in URL first.")
					app.RequestUpdate()
					return
				}
				if err := app.Clipboard().SetText(url); err != nil {
					o.status.Set("Cannot copy to the clipboard: " + err.Error())
				} else {
					o.status.Set("Sign-in URL copied to the clipboard.")
				}
				app.RequestUpdate()
			}),
			ui.Button("Open in browser").OnClick(func() {
				if o.signInURL.Get() == "" {
					o.status.Set("Create the sign-in URL first.")
					app.RequestUpdate()
					return
				}
				if err := app.OpenURL(o.signInURL.Get()); err != nil {
					o.status.Set("Cannot open the browser: " + err.Error())
				} else {
					o.status.Set("Sign-in URL opened. Paste the final URL back here when you are done.")
				}
				app.RequestUpdate()
			}),
		).Spacing(8),

		ui.HBox(
			ui.Label("Redirect URL").Style("section-title").MinWidth(96),
			ui.TextInput().BindText(&o.redirectURL).Placeholder("paste the final URL from the browser").MainWeight(1),
		).Spacing(8),

		ui.Label(o.status.Get()).Style("muted").WrapMode(gui.WrapWordChar),

		ui.HBox(
			ui.Label("").MainWeight(1),
			ui.Button("Register").Style(modern.Primary).OnClick(func() {
				o.register(app, onSuccess)
			}),
		).Spacing(8),
	).Spacing(8).Padding(16)
}

// directoryStep is the download directory step.
func (o *onboardingState) directoryStep(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Choose where downloaded files are written").Style("section-title"),
		ui.Label("The decrypted .m4b files and the encrypted .aaxc payloads are written here.").Style("muted").WrapMode(gui.WrapWordChar),
		ui.Button("Choose download directory…").OnClick(func() {
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
}

// exportStep is the export format step.
func (o *onboardingState) exportStep(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Choose an export format").Style("section-title"),
		ui.Label("Exporting writes an .aax file plus sidecar JSON, which AAX Audio Converter turns into .mp3 or .m4a.").Style("muted").WrapMode(gui.WrapWordChar),
		ui.HBox(
			ui.Button("Yes, export to .aax").Style(modern.Primary).OnClick(func() {
				// TODO: set exportToAax to true.
				app.RequestUpdate()
			}),
			ui.Button("No, keep .m4b only").OnClick(func() {
				// TODO: set exportToAax to false.
				app.RequestUpdate()
			}),
		).Spacing(8),
	).Spacing(8).Padding(16)
}

// wizardBackButton returns the Back button, or a spacer when on the first step.
func (o *onboardingState) wizardBackButton(app ui.App) ui.View {
	if o.step == 0 {
		return ui.Label("").MainWeight(1)
	}
	return ui.Button("Back").OnClick(func() {
		if o.step > 0 {
			o.step--
			app.RequestUpdate()
		}
	})
}

// wizardNextButton returns the Next/Finish button.
func (o *onboardingState) wizardNextButton(app ui.App, total int) ui.View {
	label := "Next"
	if o.step == total-1 {
		label = "Finish"
	}
	return ui.Button(label).Style(modern.Primary).OnClick(func() {
		if o.step < total-1 {
			o.step++
		} else {
			o.completed = true
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
