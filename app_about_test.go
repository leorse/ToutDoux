package main

import "testing"

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
	ok := Release{Version: "1.0.1", Date: "2026-09-25", Changes: []string{"Une modification"}}
	ancienne := Release{Version: "1.0.0", Date: "2026-08-27", Changes: []string{"Départ"}}

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
		{"modification vide", "1.0.1", []Release{{Version: "1.0.1", Date: "2026-09-25", Changes: []string{"  "}}}, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if err := checkReleases(c.version, c.releases); (err != nil) != c.wantErr {
				t.Fatalf("erreur = %v, wantErr %v", err, c.wantErr)
			}
		})
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
