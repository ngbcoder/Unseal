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

// profileState holds the state of the registration dialog.
type profileState struct {
	// regionIndex is the selected region.
	regionIndex int
	// preAmazon selects the legacy Audible username path.
	preAmazon bool

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
	// regions is the dropdown model.
	regions *gui.SliceListModel[widgets.DropDownItem]
}

// newProfileState creates the state of the registration dialog.
func newProfileState() *profileState {
	items := make([]widgets.DropDownItem, 0, len(types.AllRegions))
	for _, r := range types.AllRegions {
		items = append(items, widgets.DropDownItem{Text: string(r)})
	}
	return &profileState{
		regions:     gui.NewSliceListModel(items),
		signInURL:   ui.MakeState(""),
		redirectURL: ui.MakeState(""),
		status:      ui.MakeState(""),
		busy:        ui.MakeState(false),
	}
}

// currentRegion returns the selected region.
func (p *profileState) currentRegion() types.Region {
	regions := types.AllRegions
	if p.regionIndex < 0 || p.regionIndex >= len(regions) {
		return regions[0]
	}
	return regions[p.regionIndex]
}

// profileWindow is the new-profile dialog.
func profileWindow(state *profileState, app ui.App) ui.WindowView {
	return ui.Window("new-profile").
		Title("Unseal — Register a device").
		Chrome(ui.WindowChromeIntegrated).
		Size(760, 600).
		MinSize(640, 520).
		Center(true).
		SkipTaskbar(true).
		Content(profileContent(state, app))
}

// profileContent is the body of the registration dialog.
func profileContent(state *profileState, app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Register an Audible device").Style("app-title"),
		ui.Label("Unseal signs you in through your own browser. Your password never reaches the application.").Style("muted").WrapMode(gui.WrapWordChar),

		ui.HBox(
			ui.Label("Region").Style("section-title").MinWidth(96),
			wui.DropDown(state.regions).Selected(state.regionIndex).OnSelected(func(i int) {
				state.regionIndex = i
				state.signInURL.Set("")
				app.RequestUpdate()
			}),
		).Spacing(8),

		ui.HBox(
			ui.Label("Sign-in URL").Style("section-title").MinWidth(96),
			ui.TextInput().BindText(&state.signInURL).ReadOnly(true).Placeholder("not created yet").MainWeight(1),
		).Spacing(8),

		ui.HBox(
			ui.Button("Create URL").Style(modern.Flat).OnClick(func() {
				state.createSignInURL()
				app.RequestUpdate()
			}),
			ui.Button("Copy URL").Style(modern.Flat).OnClick(func() {
				url := state.signInURL.Get()
				if url == "" {
					state.status.Set("Create the sign-in URL first.")
					app.RequestUpdate()
					return
				}
				if err := app.Clipboard().SetText(url); err != nil {
					state.status.Set("Cannot copy to the clipboard: " + err.Error())
				} else {
					state.status.Set("Sign-in URL copied to the clipboard.")
				}
				app.RequestUpdate()
			}),
			ui.Button("Open in browser").Style(modern.Flat).OnClick(func() {
				if state.signInURL.Get() == "" {
					state.status.Set("Create the sign-in URL first.")
					app.RequestUpdate()
					return
				}
				if err := app.OpenURL(state.signInURL.Get()); err != nil {
					state.status.Set("Cannot open the browser: " + err.Error())
				} else {
					state.status.Set("Sign-in URL opened. Paste the final URL back here when you are done.")
				}
				app.RequestUpdate()
			}),
		).Spacing(8),

		ui.HBox(
			ui.Label("Redirect URL").Style("section-title").MinWidth(96),
			ui.TextInput().BindText(&state.redirectURL).Placeholder("paste the final URL from the browser").MainWeight(1),
		).Spacing(8),

		ui.Label(state.status.Get()).Style("muted").WrapMode(gui.WrapWordChar),

		ui.HBox(
			ui.Label("").MainWeight(1),
			ui.Button("Register").Style(modern.Primary).OnClick(func() {
				state.register(app)
			}),
		).Spacing(8),
	).Spacing(10).Padding(20)
}

// createSignInURL generates the sign-in URL.
func (p *profileState) createSignInURL() {
	region := p.currentRegion()
	p.login = audible.NewLogin(string(region), p.preAmazon)
	url := audible.BuildAuthURI(p.login, "")
	p.signInURL.Set(url)
	p.status.Set("Sign-in URL created. Open it in your browser to sign in.")
}

// register performs the device registration.
func (p *profileState) register(app ui.App) {
	if p.login.Serial == "" {
		p.status.Set("Create the sign-in URL first.")
		app.RequestUpdate()
		return
	}

	raw := p.redirectURL.Get()
	if raw == "" {
		p.status.Set("Paste the final URL from the browser.")
		app.RequestUpdate()
		return
	}

	code, err := audible.ParseRedirectURL(raw)
	if err != nil {
		p.status.Set("That does not look like a redirect URL: " + err.Error())
		app.RequestUpdate()
		return
	}

	region := p.currentRegion()
	p.busy.Set(true)
	p.status.Set("Registering...")
	app.RequestUpdate()

	// The registration is a network call, so it runs off the interface
	// goroutine and posts the result back.
	go func() {
		ctx := context.Background()
		profile, err := audible.Register(ctx, string(region), p.login, code, p.preAmazon)

		app.Post(func() {
			p.busy.Set(false)
			if err != nil {
				p.status.Set("Registration failed: " + err.Error())
			} else {
				p.status.Set(fmt.Sprintf("Registered %s (account %s).", profile.DeviceInfo.Name, profile.CustomerInfo.AccountID))
				p.onRegistered(profile)
			}
			app.RequestUpdate()
		})
	}()
}

// onRegistered is called when a registration succeeds.
//
// TODO: store the profile in the encrypted credential store and add it to
// the profile list.
func (p *profileState) onRegistered(profile *audible.Profile) {
}
