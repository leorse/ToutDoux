package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectDossierAbsent(t *testing.T) {
	// L'état de départ de toute installation : la fonctionnalité est
	// optionnelle, le dossier n'existe pas encore.
	s := Detect(filepath.Join(t.TempDir(), "models"))

	if s.Available {
		t.Fatal("modèle annoncé disponible alors que le dossier n'existe pas")
	}
	if len(s.MissingFiles) != 3 {
		t.Fatalf("%d fichiers manquants, attendu 3 : %v", len(s.MissingFiles), s.MissingFiles)
	}
	if s.DownloadURL == "" || s.Directory == "" {
		t.Fatal("le statut doit toujours porter le chemin et le lien, c'est tout son intérêt")
	}
}

func TestDetectModeleComplet(t *testing.T) {
	dir := t.TempDir()
	for _, nom := range []string{FichierModele, FichierTokenizer, FichierSentencePi} {
		if err := os.WriteFile(filepath.Join(dir, nom), []byte("contenu"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := Detect(dir)
	if !s.Available {
		t.Fatalf("modèle annoncé absent, manquants : %v", s.MissingFiles)
	}
	if len(s.MissingFiles) != 0 {
		t.Fatalf("fichiers manquants inattendus : %v", s.MissingFiles)
	}
	for _, f := range s.Files {
		if !f.Present || f.Size == 0 {
			t.Fatalf("fichier %s mal détecté : %+v", f.Name, f)
		}
	}
}

func TestDetectFichierVideCompteCommeAbsent(t *testing.T) {
	// Ce que laisse un téléchargement interrompu. Le charger produirait une
	// erreur ONNX bien plus obscure que « fichier manquant ».
	dir := t.TempDir()
	for _, nom := range []string{FichierModele, FichierTokenizer, FichierSentencePi} {
		if err := os.WriteFile(filepath.Join(dir, nom), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := Detect(dir)
	if s.Available {
		t.Fatal("trois fichiers vides ne constituent pas un modèle utilisable")
	}
	if len(s.MissingFiles) != 3 {
		t.Fatalf("manquants = %v, attendu les trois", s.MissingFiles)
	}
}

func TestDirEstVoisinDeLaBase(t *testing.T) {
	if got := Dir(filepath.Join("C:", "data", "app.db")); got != filepath.Join("C:", "data", "models") {
		t.Fatalf("Dir = %q", got)
	}
}
