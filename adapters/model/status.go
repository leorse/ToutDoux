// Package model détecte la présence du modèle sémantique déposé par
// l'utilisateur (§2.12, §2.13, §3.3).
//
// Il ne télécharge rien et ne crée rien. C'est une contrainte produit, pas une
// simplification : le §6 interdit toute requête sortante, et le §2.12 pose
// explicitement qu'aucune tentative automatique n'a lieu, même sur clic. Le
// rôle de ce package se limite à dire ce qui manque et où le poser.
package model

import (
	"os"
	"path/filepath"
)

// Nom des trois fichiers attendus dans le dossier des modèles (§3.3).
//
// Ils sont fixes et non configurables : un chemin paramétrable ajouterait un
// réglage à documenter et à valider pour un gain nul, l'utilisateur n'ayant
// aucune raison de vouloir les ranger ailleurs.
const (
	FichierModele     = "model.onnx"
	FichierTokenizer  = "tokenizer.json"
	FichierSentencePi = "sentencepiece.bpe.model"
)

// DownloadURL est la page depuis laquelle récupérer le modèle, à ouvrir depuis
// un poste ayant accès au réseau (§2.12).
const DownloadURL = "https://huggingface.co/Xenova/multilingual-e5-small/tree/main"

// File décrit un fichier attendu et son état (§2.13, « détail fichier par
// fichier »).
type File struct {
	Name string `json:"name"`

	// Source est le chemin du fichier dans le dépôt d'origine. Il diffère du
	// nom local pour le modèle : le dépôt publie plusieurs quantifications, et
	// c'est la version int8 qu'il faut prendre, puis renommer (§3.5).
	Source string `json:"source"`

	Present bool  `json:"present"`
	Size    int64 `json:"size"`
}

// Status est l'état complet du modèle. Une seule structure alimente la fenêtre
// du §2.12 et la vue Préférences du §2.13, pour que les deux ne puissent pas
// diverger (§3.6).
type Status struct {
	Available     bool     `json:"available"`
	Directory     string   `json:"directory"`
	ExpectedFiles []string `json:"expectedFiles"`
	MissingFiles  []string `json:"missingFiles"`
	DownloadURL   string   `json:"downloadUrl"`
	Files         []File   `json:"files"`

	// Runtime décrit `onnxruntime.dll`, qui ne vit **pas** dans ce dossier :
	// c'est une bibliothèque d'exécution livrée avec l'application, à côté de
	// l'exécutable (§3.5). Elle figure ici parce que l'utilisateur qui cherche
	// pourquoi la recherche sémantique ne marche pas doit voir les deux moitiés
	// du problème au même endroit — sinon il retéléchargera le modèle.
	Runtime File `json:"runtime"`
}

// attendus décrit les trois fichiers et leur provenance.
var attendus = []File{
	{Name: FichierModele, Source: "onnx/model_int8.onnx"},
	{Name: FichierTokenizer, Source: "tokenizer.json"},
	{Name: FichierSentencePi, Source: "sentencepiece.bpe.model"},
}

// Dir rend le dossier des modèles, à côté de la base (§3.3).
//
// C'est le dossier des données utilisateur et non celui de l'exécutable : ce
// dernier peut être installé à un emplacement non inscriptible, et le poste
// cible est restreint (§3.5).
func Dir(dbPath string) string { return filepath.Join(filepath.Dir(dbPath), "models") }

// Detect examine le dossier et rend l'état des trois fichiers.
//
// Un dossier absent n'est pas une erreur : c'est l'état de départ de toute
// installation, la fonctionnalité étant optionnelle. On rend simplement les
// trois fichiers comme manquants.
func Detect(dir string) Status {
	s := Status{
		Directory:     dir,
		ExpectedFiles: make([]string, 0, len(attendus)),
		MissingFiles:  []string{},
		DownloadURL:   DownloadURL,
		Files:         make([]File, 0, len(attendus)),
	}

	for _, f := range attendus {
		s.ExpectedFiles = append(s.ExpectedFiles, f.Name)

		// Un fichier de taille nulle est traité comme absent : c'est ce que
		// laisse un téléchargement interrompu, et le charger produirait une
		// erreur ONNX bien plus obscure que « fichier manquant ».
		if info, err := os.Stat(filepath.Join(dir, f.Name)); err == nil && !info.IsDir() && info.Size() > 0 {
			f.Present = true
			f.Size = info.Size()
		} else {
			s.MissingFiles = append(s.MissingFiles, f.Name)
		}
		s.Files = append(s.Files, f)
	}

	s.Available = len(s.MissingFiles) == 0
	return s
}

// Paths rend les chemins complets des trois fichiers, dans l'ordre modèle,
// tokenizer, SentencePiece.
func Paths(dir string) (modele, tokenizer, sentencepiece string) {
	return filepath.Join(dir, FichierModele),
		filepath.Join(dir, FichierTokenizer),
		filepath.Join(dir, FichierSentencePi)
}
