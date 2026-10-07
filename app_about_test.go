package main

import (
	"strings"
	"testing"
)

func TestWindowTitle(t *testing.T) {
	if got, want := windowTitle("1.0.1"), "Tout Doux (1.0.1)"; got != want {
		t.Errorf("windowTitle = %q, want %q", got, want)
	}
}

func TestVersionFromConfig(t *testing.T) {
	cas := []struct {
		nom     string
		raw     string
		want    string
		wantErr bool
	}{
		{"valide", `{"info":{"productVersion":"1.0.1"}}`, "1.0.1", false},
		{"absente", `{"name":"toutdoux"}`, "", true},
		{"vide", `{"info":{"productVersion":""}}`, "", true},
		{"deux nombres", `{"info":{"productVersion":"1.0"}}`, "", true},
		{"suffixe", `{"info":{"productVersion":"1.0.1-beta"}}`, "", true},
		{"json invalide", `{`, "", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got, err := versionFromConfig([]byte(c.raw))
			if (err != nil) != c.wantErr {
				t.Fatalf("erreur = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("version = %q, want %q", got, c.want)
			}
		})
	}
}

func TestEmbeddedConfigHasValidVersion(t *testing.T) {
	if _, err := versionFromConfig(wailsConfig); err != nil {
		t.Fatalf("wails.json embarqué : %v", err)
	}
}

// Garde-fou : la version la plus récente de releases.json doit être la version
// configurée dans wails.json. Ce test échoue quand on change l'une sans l'autre.
func TestEmbeddedReleasesMatchVersion(t *testing.T) {
	version, err := versionFromConfig(wailsConfig)
	if err != nil {
		t.Fatalf("wails.json : %v", err)
	}
	releases, err := parseReleases(releasesFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := checkReleases(version, releases); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReleases(t *testing.T) {
	ajout := Change{Type: "ajout", Level: "majeur", Text: "Une modification"}
	ok := Release{Version: "1.0.1", Date: "2026-09-25", Changes: []Change{ajout}}
	ancienne := Release{Version: "1.0.0", Date: "2026-08-27", Changes: []Change{{Type: "correction", Level: "mineur", Text: "Départ"}}}
	avec := func(c Change) []Release {
		return []Release{{Version: "1.0.1", Date: "2026-09-25", Changes: []Change{ajout, c}}}
	}

	cas := []struct {
		nom      string
		version  string
		releases []Release
		wantErr  bool
	}{
		{"cohérent", "1.0.1", []Release{ok, ancienne}, false},
		{"version configurée plus récente", "1.0.2", []Release{ok, ancienne}, true},
		{"historique vide", "1.0.1", nil, true},
		{"date invalide", "1.0.1", []Release{{Version: "1.0.1", Date: "25/09/2026", Changes: ok.Changes}}, true},
		{"version mal formée", "1.0.1", []Release{ok, {Version: "1.0", Date: "2026-08-27", Changes: ancienne.Changes}}, true},
		{"aucune modification", "1.0.1", []Release{{Version: "1.0.1", Date: "2026-09-25"}}, true},
		{"modification vide", "1.0.1", avec(Change{Type: "ajout", Level: "mineur", Text: "  "}), true},
		{"nature inconnue", "1.0.1", avec(Change{Type: "évolution", Level: "mineur", Text: "x"}), true},
		{"nature absente", "1.0.1", avec(Change{Level: "mineur", Text: "x"}), true},
		{"portée inconnue", "1.0.1", avec(Change{Type: "correction", Level: "moyen", Text: "x"}), true},
		{"portée absente", "1.0.1", avec(Change{Type: "correction", Text: "x"}), true},
		{"correction mineure", "1.0.1", avec(Change{Type: "correction", Level: "mineur", Text: "x"}), false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if err := checkReleases(c.version, c.releases); (err != nil) != c.wantErr {
				t.Fatalf("erreur = %v, wantErr %v", err, c.wantErr)
			}
		})
	}
}

// L'erreur doit dire quelle version corriger : l'historique grossit, et un
// message sans repère obligerait à relire tout le fichier.
func TestCheckReleases_NamesTheRelease(t *testing.T) {
	releases := []Release{{Version: "1.0.1", Date: "2026-09-25", Changes: []Change{{Type: "bug", Level: "mineur", Text: "x"}}}}
	err := checkReleases("1.0.1", releases)
	if err == nil || !strings.Contains(err.Error(), "1.0.1") {
		t.Errorf("erreur = %v, attendu une erreur citant la version 1.0.1", err)
	}
}

// Une entrée restée à l'ancien format — une simple chaîne — ne doit pas être
// affichée sans catégorie : elle est refusée à la lecture.
func TestParseReleases_RejectsUncategorisedChange(t *testing.T) {
	raw := []byte(`{"releases":[{"version":"1.0.0","date":"2026-01-01","changes":["texte nu"]}]}`)
	if _, err := parseReleases(raw); err == nil {
		t.Error("une modification sans nature ni portée aurait dû être refusée")
	}
}

func TestGetAppInfo(t *testing.T) {
	info, err := NewApp().GetAppInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Version == "" || len(info.Releases) == 0 {
		t.Fatalf("AppInfo incomplet : %+v", info)
	}
	if info.Releases[0].Version != info.Version {
		t.Errorf("la première release est %q, la version courante %q", info.Releases[0].Version, info.Version)
	}
}
