package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// wailsConfig est wails.json, embarqué : `info.productVersion` y est l'unique
// source du numéro de version. Wails la relit pour les métadonnées de l'exécutable
// et l'installateur ; l'embarquer évite de la dupliquer ailleurs.
//
//go:embed wails.json
var wailsConfig []byte

var versionFormat = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// versionFromConfig extrait `info.productVersion` du contenu de wails.json.
//
// Le format MAJEUR.MINEUR.CORRECTIF est imposé : l'installateur y ajoute « .0 »
// pour fabriquer la version à quatre nombres de Windows.
func versionFromConfig(raw []byte) (string, error) {
	var cfg struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return "", fmt.Errorf("wails.json illisible : %w", err)
	}
	v := cfg.Info.ProductVersion
	if !versionFormat.MatchString(v) {
		return "", fmt.Errorf("info.productVersion %q n'est pas de la forme MAJEUR.MINEUR.CORRECTIF", v)
	}
	return v, nil
}

// windowTitle est le titre de la fenêtre native : le nom suivi de la version.
func windowTitle(version string) string {
	return fmt.Sprintf("Tout Doux (%s)", version)
}

// releasesFile est l'historique des versions, de la plus récente à la plus
// ancienne. Le texte des changements accepte **gras** et *italique*, interprétés
// par le frontend.
//
//go:embed releases.json
var releasesFile []byte

// Nature et portée d'un changement, telles qu'écrites dans releases.json.
const (
	ChangeAjout      = "ajout"
	ChangeCorrection = "correction"

	LevelMajeur = "majeur"
	LevelMineur = "mineur"
)

// Change est une ligne de l'historique : un ajout ou une correction, majeur ou
// mineur. L'onglet « À propos » range les changements d'une version selon ces
// deux axes.
type Change struct {
	Type  string `json:"type"`
	Level string `json:"level"`
	Text  string `json:"text"`
}

// Release est une entrée de l'historique. Important est facultatif : un encadré
// mis en tête de la version, pour ce qu'il ne faut pas rater.
type Release struct {
	Version   string   `json:"version"`
	Date      string   `json:"date"`
	Important string   `json:"important,omitempty"`
	Changes   []Change `json:"changes"`
}

// AppInfo est ce que l'onglet « À propos » affiche.
type AppInfo struct {
	Version  string    `json:"version"`
	Releases []Release `json:"releases"`
}

func parseReleases(raw []byte) ([]Release, error) {
	var file struct {
		Releases []Release `json:"releases"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("releases.json illisible : %w", err)
	}
	return file.Releases, nil
}

// checkReleases vérifie que l'historique est cohérent avec la version
// configurée : c'est ce qui empêche d'oublier de documenter une version.
func checkReleases(version string, releases []Release) error {
	if len(releases) == 0 {
		return fmt.Errorf("releases.json ne contient aucune version")
	}
	if releases[0].Version != version {
		return fmt.Errorf("la version la plus récente de releases.json est %q, la version configurée est %q",
			releases[0].Version, version)
	}
	for _, r := range releases {
		if !versionFormat.MatchString(r.Version) {
			return fmt.Errorf("version %q de releases.json : format MAJEUR.MINEUR.CORRECTIF attendu", r.Version)
		}
		if _, err := time.Parse("2006-01-02", r.Date); err != nil {
			return fmt.Errorf("version %s : date %q invalide, format AAAA-MM-JJ attendu", r.Version, r.Date)
		}
		if len(r.Changes) == 0 {
			return fmt.Errorf("version %s : aucune modification listée", r.Version)
		}
		for _, c := range r.Changes {
			if strings.TrimSpace(c.Text) == "" {
				return fmt.Errorf("version %s : une modification est vide", r.Version)
			}
			if c.Type != ChangeAjout && c.Type != ChangeCorrection {
				return fmt.Errorf("version %s : nature %q inconnue pour « %s », %q ou %q attendu",
					r.Version, c.Type, c.Text, ChangeAjout, ChangeCorrection)
			}
			if c.Level != LevelMajeur && c.Level != LevelMineur {
				return fmt.Errorf("version %s : portée %q inconnue pour « %s », %q ou %q attendu",
					r.Version, c.Level, c.Text, LevelMajeur, LevelMineur)
			}
		}
	}
	return nil
}

// GetAppInfo rend la version courante et l'historique des versions (« À propos »).
func (a *App) GetAppInfo() (AppInfo, error) {
	version, err := versionFromConfig(wailsConfig)
	if err != nil {
		return AppInfo{}, err
	}
	releases, err := parseReleases(releasesFile)
	if err != nil {
		return AppInfo{}, err
	}
	return AppInfo{Version: version, Releases: releases}, nil
}
