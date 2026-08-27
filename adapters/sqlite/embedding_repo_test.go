package sqlite

import (
	"errors"
	"math"
	"testing"

	"toutdoux/domain"
)

// Round-trip de l'index sémantique (§3.2). L'enjeu réel est la sérialisation du
// vecteur en BLOB : un vecteur relu à l'envers ne produirait pas une erreur,
// mais des scores absurdes — le pire des deux mondes.

func TestEmbeddingRepository_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewEmbeddingRepository(db)

	vecteur := []float32{0.125, -0.5, 3.75, 0, 1e-7, -2.25}
	e := domain.Embedding{
		EntityID:   "note-1",
		Type:       domain.SearchTypeNote,
		ProjectID:  domain.DiversProjectID,
		Vector:     vecteur,
		Dimensions: len(vecteur),
		SourceHash: "empreinte",
	}
	if err := repo.Put(e); err != nil {
		t.Fatalf("écriture : %v", err)
	}

	relu, err := repo.Get("note-1")
	if err != nil {
		t.Fatalf("lecture : %v", err)
	}
	if len(relu.Vector) != len(vecteur) {
		t.Fatalf("%d composantes relues, attendu %d", len(relu.Vector), len(vecteur))
	}
	for i := range vecteur {
		// Égalité stricte attendue : float32 → 4 octets → float32 est exact,
		// il n'y a aucune conversion de précision en jeu.
		if relu.Vector[i] != vecteur[i] {
			t.Fatalf("composante %d = %v, attendu %v", i, relu.Vector[i], vecteur[i])
		}
	}
	if relu.Type != domain.SearchTypeNote || relu.SourceHash != "empreinte" || relu.Dimensions != len(vecteur) {
		t.Fatalf("métadonnées relues : %+v", relu)
	}
	if relu.UpdatedAt.IsZero() {
		t.Fatal("updated_at non renseigné")
	}
}

func TestEmbeddingRepository_PutRemplace(t *testing.T) {
	db := newTestDB(t)
	repo := NewEmbeddingRepository(db)

	base := domain.Embedding{
		EntityID: "note-1", Type: domain.SearchTypeNote, ProjectID: domain.DiversProjectID,
		Vector: []float32{1, 0}, Dimensions: 2, SourceHash: "v1",
	}
	if err := repo.Put(base); err != nil {
		t.Fatal(err)
	}
	base.Vector, base.SourceHash = []float32{0, 1}, "v2"
	if err := repo.Put(base); err != nil {
		t.Fatal(err)
	}

	// Une entité, un vecteur : la revectorisation remplace, elle n'empile pas.
	n, err := repo.Count()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("%d vecteurs, attendu 1", n)
	}
	relu, err := repo.Get("note-1")
	if err != nil {
		t.Fatal(err)
	}
	if relu.SourceHash != "v2" || relu.Vector[1] != 1 {
		t.Fatalf("le remplacement n'a pas eu lieu : %+v", relu)
	}
}

func TestEmbeddingRepository_GetIntrouvable(t *testing.T) {
	repo := NewEmbeddingRepository(newTestDB(t))
	if _, err := repo.Get("inconnu"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("erreur = %v, attendu ErrNotFound", err)
	}
}

func TestEmbeddingRepository_DeleteEstIdempotent(t *testing.T) {
	repo := NewEmbeddingRepository(newTestDB(t))
	// L'appelant supprime des entités sans savoir si elles étaient indexées.
	if err := repo.Delete("jamais-indexe"); err != nil {
		t.Fatalf("suppression d'une entrée absente : %v", err)
	}
}

func TestEmbeddingRepository_DeleteByProject(t *testing.T) {
	db := newTestDB(t)
	repo := NewEmbeddingRepository(db)

	for _, id := range []string{"a", "b"} {
		if err := repo.Put(domain.Embedding{
			EntityID: id, Type: domain.SearchTypeTask, ProjectID: domain.DiversProjectID,
			Vector: []float32{1, 1}, Dimensions: 2, SourceHash: id,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DeleteByProject(domain.DiversProjectID); err != nil {
		t.Fatal(err)
	}
	n, err := repo.Count()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("%d vecteurs subsistent", n)
	}
}

func TestEmbeddingRepository_RefuseUnVecteurVide(t *testing.T) {
	repo := NewEmbeddingRepository(newTestDB(t))
	err := repo.Put(domain.Embedding{
		EntityID: "vide", Type: domain.SearchTypeNote, ProjectID: domain.DiversProjectID,
		SourceHash: "x",
	})
	if err == nil {
		t.Fatal("un vecteur vide devrait être refusé à l'écriture")
	}
}

func TestEncodeDecodeVector_ValeursExtremes(t *testing.T) {
	// Les bornes du float32 doivent survivre à l'aller-retour : un modèle
	// quantifié peut produire des valeurs très petites.
	v := []float32{math.MaxFloat32, -math.MaxFloat32, math.SmallestNonzeroFloat32, 0}
	blob, err := encodeVector(v)
	if err != nil {
		t.Fatal(err)
	}
	if len(blob) != len(v)*4 {
		t.Fatalf("%d octets pour %d composantes", len(blob), len(v))
	}
	relu, err := decodeVector(blob)
	if err != nil {
		t.Fatal(err)
	}
	for i := range v {
		if relu[i] != v[i] {
			t.Fatalf("composante %d = %v, attendu %v", i, relu[i], v[i])
		}
	}
}

func TestDecodeVector_TailleInvalide(t *testing.T) {
	if _, err := decodeVector([]byte{1, 2, 3}); err == nil {
		t.Fatal("un BLOB non multiple de 4 devrait être refusé")
	}
}
