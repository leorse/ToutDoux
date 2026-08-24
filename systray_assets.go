package main

import "embed"

// Icônes de la barre système (§2.10). Elles sont embarquées dans le binaire
// plutôt que lues sur disque : la librairie systray réclame les octets de
// l'icône à l'exécution, et l'exe doit rester distribuable seul.
//
//go:embed assets/systray/*.ico
var systrayIcons embed.FS

// États de l'icône de la barre système, par urgence croissante (§2.10).
//
// Il n'y a pas d'icône vide : le clignotement alterne entre l'icône du niveau
// courant et TrayNormal — critique ↔ normale, ou haute ↔ normale. Alterner vers
// une icône vide donnerait l'impression que l'application a disparu de la barre.
// Le fichier assets/systray/blank.ico n'est donc pas utilisé.
const (
	TrayNormal   = "assets/systray/normal.ico"
	TrayHigh     = "assets/systray/high.ico"
	TrayCritical = "assets/systray/critical.ico"

	// TrayClock alterne avec l'icône du niveau quand une tâche est imminente
	// ou en retard (§2.10).
	TrayClock = "assets/systray/clock.ico"
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
