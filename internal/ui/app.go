package ui

import (
	"log"

	"github.com/golang-gui/goui/theme/modern"
	"github.com/golang-gui/goui/ui"
	wui "github.com/golang-gui/goui/widgets/ui"
)

// AppID is the stable application identity.
const AppID = "com.audiamus.unseal"

// Run starts the desktop application.
//
// It must be called from main, never from a background goroutine.
func Run() error {
	err := ui.Run(AppID, func(app ui.App) ui.RootView {
		return ui.Root().
			StyleSheet(modern.FromSettings(app.Settings())).
			Windows(
				ui.Window("main").
					Title("Unseal").
					Size(960, 640).
					Center(true).
					Content(mainView(app)),
			)
	})
	if err != nil {
		log.Printf("unseal: %v", err)
	}
	return err
}

// mainView is the content of the main window.
func mainView(app ui.App) ui.View {
	return ui.VBox(
		settingsPanel(app),
		downloadQueue(app),
		progressPanel(app),
	).Spacing(8).Padding(12)
}

// settingsPanel holds the three settings group boxes.
func settingsPanel(app ui.App) ui.View {
	return ui.HBox(
		profileGroup(app),
		downloadGroup(app),
		exportGroup(app),
	).Spacing(8)
}

// profileGroup is the profile settings group.
func profileGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Profile").Style("title"),
		ui.Button("Profiles…"),
	).Spacing(4).Padding(8)
}

// downloadGroup is the download settings group.
func downloadGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Download").Style("title"),
		ui.Label("(settings will go here)"),
	).Spacing(4).Padding(8)
}

// exportGroup is the export settings group.
func exportGroup(app ui.App) ui.View {
	return ui.VBox(
		ui.Label("Export").Style("title"),
		ui.Label("(settings will go here)"),
	).Spacing(4).Padding(8)
}

// downloadQueue is the centre panel with the download queue.
func downloadQueue(app ui.App) ui.View {
	return ui.VBox(
		ui.HBox(
			ui.Button("Download…"),
			ui.Button("Remove"),
		).Spacing(4),
		ui.Label("Download queue (empty)"),
	).Spacing(4).Padding(8)
}

// progressPanel holds the three progress bars and the Run/Abort buttons.
func progressPanel(app ui.App) ui.View {
	return ui.VBox(
		ui.HBox(
			ui.Label("Download"),
			wui.ProgressBar(0),
		).Spacing(4),
		ui.HBox(
			ui.Label("Decrypt"),
			wui.ProgressBar(0),
		).Spacing(4),
		ui.HBox(
			ui.Label("Export"),
			wui.ProgressBar(0),
		).Spacing(4),
		ui.HBox(
			ui.Button("Run"),
			ui.Button("Abort"),
		).Spacing(4),
	).Spacing(4).Padding(8)
}
