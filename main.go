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

		// §2.10 demande que le X cache la fenêtre au lieu de quitter, l'app
		// continuant en arrière-plan. Le drapeau existe et fonctionne :
		//
		//     HideWindowOnClose: true,
		//
		// mais il reste désactivé jusqu'à la Phase 4, parce qu'il n'a de sens
		// qu'avec la seconde moitié du §2.10 : l'icône de barre système et son
		// entrée « Quitter ». Sans elle, cacher la fenêtre laisse un processus
		// vivant, invisible et impossible à arrêter autrement qu'en le tuant —
		// et `wails dev` ne rend jamais la main.
		//
		// À réactiver dans le même temps que le systray, jamais avant.
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
