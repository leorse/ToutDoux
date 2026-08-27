package onnx

import (
	"os"
	"path/filepath"
	"testing"

	"toutdoux/domain/semantic"
)

// Ces tests exécutent le **vrai** modèle. Ils se sautent s'il n'est pas déposé.
//
// Ils ne prétendent pas mesurer la qualité sémantique — cela reste une
// vérification manuelle (§3.10, étape 5.14). Ce qu'ils verrouillent, c'est que
// la chaîne complète produit des vecteurs exploitables : bonne dimension,
// normalisés, déterministes, et ordonnés dans le sens attendu sur des cas où
// la réponse ne fait aucun doute.

func vraiProvider(t *testing.T) *Provider {
	t.Helper()

	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("dossier de configuration indisponible : %v", err)
	}
	dir := filepath.Join(base, "ToutDoux", "models")

	p := NewProvider(dir)
	if !p.Available() {
		t.Skipf("modèle ou %s absent de %s", NomRuntime, dir)
	}
	t.Cleanup(p.Close)
	return p
}

func TestProvider_VecteurNormaliseEtDeLaBonneDimension(t *testing.T) {
	p := vraiProvider(t)

	v, err := p.Embed("Le client est en colère depuis la livraison")
	if err != nil {
		t.Fatalf("vectorisation : %v", err)
	}
	// multilingual-e5-small produit 384 dimensions.
	if len(v) != 384 {
		t.Fatalf("%d dimensions, attendu 384", len(v))
	}

	// La normalisation L2 est ce qui rend le produit scalaire égal à la
	// similarité cosinus : sans elle, les scores ne seraient pas comparables.
	s, err := semantic.Cosine(v, v)
	if err != nil {
		t.Fatal(err)
	}
	if s < 0.999 {
		t.Fatalf("similarité avec soi-même = %v", s)
	}
}

func TestProvider_Deterministe(t *testing.T) {
	p := vraiProvider(t)

	a, err := p.Embed("Réunion de cadrage")
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.Embed("Réunion de cadrage")
	if err != nil {
		t.Fatal(err)
	}
	s, err := semantic.Cosine(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if s < 0.9999 {
		t.Fatalf("deux vectorisations du même texte diffèrent : similarité %v", s)
	}
}

// La reformulation, qui est la raison d'être de la fonctionnalité (§2.12).
func TestProvider_ReformulationPlusProcheQueSujetSansRapport(t *testing.T) {
	p := vraiProvider(t)

	requete, err := p.EmbedQuery("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	proche, err := p.Embed("Le client est en colère depuis la livraison")
	if err != nil {
		t.Fatal(err)
	}
	loin, err := p.Embed("Commander du matériel de bureau pour l’étage")
	if err != nil {
		t.Fatal(err)
	}

	sProche, err := semantic.Cosine(requete, proche)
	if err != nil {
		t.Fatal(err)
	}
	sLoin, err := semantic.Cosine(requete, loin)
	if err != nil {
		t.Fatal(err)
	}
	if sProche <= sLoin {
		t.Fatalf("« client en colère » (%.4f) ne sort pas devant « matériel de bureau » (%.4f)", sProche, sLoin)
	}
	t.Logf("reformulation %.4f contre sujet étranger %.4f", sProche, sLoin)
}

// La tolérance aux fautes de frappe, second cas du §2.12.
func TestProvider_FauteDeFrappeRetrouveQuandMeme(t *testing.T) {
	p := vraiProvider(t)

	faute, err := p.EmbedQuery("réuion de cadrage")
	if err != nil {
		t.Fatal(err)
	}
	bon, err := p.Embed("Préparer la réunion de cadrage")
	if err != nil {
		t.Fatal(err)
	}
	autre, err := p.Embed("Commander du matériel de bureau")
	if err != nil {
		t.Fatal(err)
	}

	sBon, _ := semantic.Cosine(faute, bon)
	sAutre, _ := semantic.Cosine(faute, autre)
	if sBon <= sAutre {
		t.Fatalf("la faute de frappe ne retrouve pas la réunion : %.4f contre %.4f", sBon, sAutre)
	}
	t.Logf("faute de frappe %.4f contre sujet étranger %.4f", sBon, sAutre)
}

// Les deux préfixes e5 doivent réellement produire des vecteurs distincts :
// c'est ce qui justifie d'avoir deux méthodes sur le port plutôt qu'une.
func TestProvider_RequeteEtPassageDifferent(t *testing.T) {
	p := vraiProvider(t)

	q, err := p.EmbedQuery("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Embed("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	s, err := semantic.Cosine(q, d)
	if err != nil {
		t.Fatal(err)
	}
	if s > 0.9999 {
		t.Fatal("les préfixes query:/passage: ne sont pas appliqués")
	}
}

func TestProvider_TexteLongNeCassePas(t *testing.T) {
	p := vraiProvider(t)

	long := ""
	for i := 0; i < 500; i++ {
		long += "réunion de cadrage avec le client mécontent "
	}
	v, err := p.Embed(long)
	if err != nil {
		t.Fatalf("un texte dépassant 512 jetons doit être tronqué, pas refusé : %v", err)
	}
	if len(v) != 384 {
		t.Fatalf("%d dimensions", len(v))
	}
}
