package main

import "embed"

// Icônes de la barre système (§2.10). Elles sont embarquées dans le binaire
// plutôt que lues sur disque : la librairie systray réclame les octets de
// l'icône à l'exécution, et l'exe doit rester distribuable seul.
//
//go:embed assets/systray/*.ico
var systrayIcons embed.FS

// États de l'icône de la barre système, par urgence croissante (§2.10).
const (
	TrayNormal   = "assets/systray/normal.ico"
	TrayHigh     = "assets/systray/high.ico"
	TrayCritical = "assets/systray/critical.ico"

	// TrayBlank sert uniquement à l'alternance du clignotement : il n'existe
	// pas d'API "faire clignoter", on bascule entre deux icônes sur un timer.
	TrayBlank = "assets/systray/blank.ico"
)

// TrayIcon renvoie les octets de l'icône demandée.
//
// Le chemin étant résolu à la compilation par go:embed, un échec ici signale
// un fichier manquant dans assets/systray/, soit une erreur de build, pas une
// condition d'exécution rattrapable.
func TrayIcon(name string) []byte {
	data, err := systrayIcons.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return data
}
