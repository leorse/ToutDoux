package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	// Le nom et la version ne s'affichent que dans la barre de titre native
	// (§2.11). Une version illisible ne doit pas empêcher de démarrer : le titre
	// retombe sur le seul nom, et le test du fichier de configuration la signale.
	title := "Tout Doux"
	if version, err := versionFromConfig(wailsConfig); err != nil {
		println("Version :", err.Error())
	} else {
		title = windowTitle(version)
	}

	err := wails.Run(&options.App{
		Title: title,

		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,

		// Le bouton X ferme réellement l'application. Réduire la fenêtre la
		// laisse dans la barre des tâches, comme n'importe quelle application.
		//
		// C'est un écart assumé au §2.10, qui prévoyait que le X masque la
		// fenêtre et laisse l'application vivre en arrière-plan. Ce comportement
		// a été essayé et retiré : il impose de suivre l'état d'affichage d'une
		// fenêtre que Wails masque sans prévenir, et toutes les variantes
		// tentées ont fini par figer l'icône de la barre système ou par empêcher
		// l'application de s'arrêter.
		//
		// **Ne pas le réintroduire par un hook OnBeforeClose.** Wails consulte
		// OnBeforeClose depuis Quit() :
		//
		//	if OnBeforeClose != nil && OnBeforeClose(ctx) { return }
		//
		// Un hook qui renvoie « empêcher la fermeture » annule donc aussi les
		// arrêts volontaires : ni « Quitter », ni le Ctrl+C de `wails dev` ne
		// terminent plus le processus.
		HideWindowOnClose: false,

		AssetServer:      &assetserver.Options{Assets: assets},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Erreur :", err.Error())
	}
}
