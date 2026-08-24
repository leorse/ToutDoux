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

	err := wails.Run(&options.App{
		Title: "Tout Doux",

		// La barre de titre est celle du système (§2.11) : le prototype web
		// portait une barre HTML custom, qui n'existait que parce qu'il tournait
		// dans un onglet de navigateur. Elle ne doit pas être reprise ici.
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,

		// Le X cache la fenêtre au lieu de quitter : l'application continue en
		// arrière-plan (§2.10).
		//
		// Ce drapeau est indissociable de l'entrée « Quitter » du menu de la
		// barre système, qui est la seule sortie de l'application. Si le tray
		// venait à être retiré, celui-ci devrait repasser à false dans le même
		// mouvement — sinon la fenêtre se ferme sur un processus invisible et
		// impossible à arrêter, et `wails dev` ne rend jamais la main.
		HideWindowOnClose: true,

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
