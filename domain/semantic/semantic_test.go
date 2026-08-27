package semantic

import (
	"errors"
	"math"
	"testing"
)

func TestCosineVecteurAvecLuiMeme(t *testing.T) {
	v := []float32{0.2, -0.5, 0.9, 0.1}
	score, err := Cosine(v, v)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	// La règle de la spec : un vecteur comparé à lui-même vaut exactement 1.
	if math.Abs(score-1) > 1e-9 {
		t.Fatalf("similarité avec soi-même = %v, attendu 1", score)
	}
}

func TestCosineDimensionsDifferentes(t *testing.T) {
	_, err := Cosine([]float32{1, 0}, []float32{1, 0, 0})
	if !errors.Is(err, ErrDimensionMismatch) {
		t.Fatalf("erreur = %v, attendu ErrDimensionMismatch", err)
	}
}

func TestCosineVecteurNul(t *testing.T) {
	if _, err := Cosine([]float32{0, 0}, []float32{1, 1}); !errors.Is(err, ErrEmptyVector) {
		t.Fatalf("erreur = %v, attendu ErrEmptyVector", err)
	}
	if _, err := Cosine(nil, nil); !errors.Is(err, ErrEmptyVector) {
		t.Fatalf("erreur = %v, attendu ErrEmptyVector sur des vecteurs vides", err)
	}
}

func TestCosineOrthogonauxEtOpposes(t *testing.T) {
	if s, _ := Cosine([]float32{1, 0}, []float32{0, 1}); math.Abs(s) > 1e-9 {
		t.Fatalf("vecteurs orthogonaux : %v, attendu 0", s)
	}
	if s, _ := Cosine([]float32{1, 0}, []float32{-1, 0}); math.Abs(s+1) > 1e-9 {
		t.Fatalf("vecteurs opposés : %v, attendu -1", s)
	}
}

func TestRankTriDecroissantEtStable(t *testing.T) {
	query := []float32{1, 0}
	candidats := []Candidate{
		{EntityID: "loin", Vector: []float32{0, 1}},
		{EntityID: "proche", Vector: []float32{1, 0.1}},
		{EntityID: "egal-1", Vector: []float32{1, 1}},
		{EntityID: "egal-2", Vector: []float32{2, 2}}, // même direction que egal-1
	}

	matches := Rank(query, candidats, -1)
	if len(matches) != 4 {
		t.Fatalf("%d résultats, attendu 4", len(matches))
	}
	if matches[0].EntityID != "proche" {
		t.Fatalf("premier résultat = %s, attendu proche", matches[0].EntityID)
	}
	// Tri stable : à score identique, l'ordre d'entrée est conservé.
	if matches[1].EntityID != "egal-1" || matches[2].EntityID != "egal-2" {
		t.Fatalf("ordre à score égal non conservé : %v", matches)
	}
	if matches[3].EntityID != "loin" {
		t.Fatalf("dernier résultat = %s, attendu loin", matches[3].EntityID)
	}
}

func TestRankIgnoreLesDimensionsIncompatibles(t *testing.T) {
	// Un vecteur produit par un modèle antérieur ne doit pas faire échouer la
	// recherche entière : il est écarté, les autres résultats restent.
	matches := Rank([]float32{1, 0}, []Candidate{
		{EntityID: "ancien-modele", Vector: []float32{1, 0, 0}},
		{EntityID: "valide", Vector: []float32{1, 0}},
	}, -1)

	if len(matches) != 1 || matches[0].EntityID != "valide" {
		t.Fatalf("résultats = %v, attendu le seul candidat valide", matches)
	}
}

func TestRankAppliqueLeSeuil(t *testing.T) {
	matches := Rank([]float32{1, 0}, []Candidate{
		{EntityID: "sous-le-seuil", Vector: []float32{0, 1}}, // score 0
		{EntityID: "au-dessus", Vector: []float32{1, 0}},     // score 1
	}, DefaultMinScore)

	if len(matches) != 1 || matches[0].EntityID != "au-dessus" {
		t.Fatalf("résultats = %v, attendu le seul candidat au-dessus du seuil", matches)
	}
}

func TestHashIdentiquePourTexteIdentique(t *testing.T) {
	if Hash("client mécontent") != Hash("client mécontent") {
		t.Fatal("deux textes identiques donnent des empreintes différentes")
	}
	if Hash("client mécontent") == Hash("client content") {
		t.Fatal("deux textes différents donnent la même empreinte")
	}
}

func TestHashIgnoreLesEspacesSuperflus(t *testing.T) {
	// TipTap réenregistre du HTML dont l'indentation varie sans que le texte ait
	// changé : ces réécritures ne doivent pas déclencher de revectorisation.
	if Hash("client  mécontent\n") != Hash(" client mécontent") {
		t.Fatal("la normalisation des espaces ne joue pas")
	}
}

func TestSourceTextIgnoreLesChampsVides(t *testing.T) {
	if got := SourceText("Titre", "", "  ", "Contenu"); got != "Titre\nContenu" {
		t.Fatalf("SourceText = %q", got)
	}
	// Vider le titre ne doit pas laisser un séparateur qui changerait l'empreinte.
	if Hash(SourceText("", "Contenu")) != Hash(SourceText("Contenu")) {
		t.Fatal("un champ vide modifie l'empreinte")
	}
}

func TestCutRelative(t *testing.T) {
	matches := []Match{
		{EntityID: "premier", Score: 0.88},
		{EntityID: "proche", Score: 0.86},
		{EntityID: "traine", Score: 0.80},
	}

	// Marge 0,03 : « proche » reste, « traine » tombe.
	garde := CutRelative(matches, 0.03)
	if len(garde) != 2 || garde[1].EntityID != "proche" {
		t.Fatalf("coupe = %v, attendu les deux premiers", garde)
	}

	// Rien ne se détache : la liste passe entière, ce qui est honnête — le
	// modèle n'a pas d'avis, la cacher serait inventer une certitude.
	serres := []Match{{EntityID: "a", Score: 0.83}, {EntityID: "b", Score: 0.82}}
	if len(CutRelative(serres, 0.03)) != 2 {
		t.Fatal("des scores serrés ne doivent pas être coupés")
	}

	if len(CutRelative(nil, 0.03)) != 0 {
		t.Fatal("liste vide")
	}
	// Marge nulle ou négative : pas de coupe, pour pouvoir la désactiver.
	if len(CutRelative(matches, 0)) != 3 {
		t.Fatal("une marge nulle doit désactiver la coupe")
	}
}
