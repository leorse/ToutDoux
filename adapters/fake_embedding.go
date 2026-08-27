package adapters

import (
	"hash/fnv"
	"math"
	"strings"
	"sync"
	"unicode"

	"toutdoux/domain"
)

// FakeDimensions est la taille des vecteurs rendus par FakeEmbeddingProvider.
//
// Volontairement petite : le vrai modèle produit 384 composantes, mais rien
// dans les règles testées ne dépend de ce nombre, et des vecteurs courts
// rendent les échecs de test lisibles.
const FakeDimensions = 64

// FakeEmbeddingProvider rend un vecteur déterministe dérivé du texte (§3.9).
//
// Il vit ici plutôt que dans un fichier _test.go pour la même raison que
// FixedClock : plusieurs packages en ont besoin, et un helper de test n'est pas
// importable d'un package à l'autre en Go.
//
// La méthode est un sac de mots haché : chaque mot tombe dans une composante,
// le vecteur est ensuite normalisé. Deux propriétés en découlent, et ce sont
// exactement celles dont les tests ont besoin :
//
//   - deux textes identiques donnent le même vecteur, deux textes différents
//     des vecteurs différents ;
//   - deux textes qui partagent des mots sont plus proches que deux textes qui
//     n'en partagent aucun, ce qui permet de tester le tri par score sur des
//     données lisibles plutôt que sur des nombres arbitraires.
//
// Ce qu'il ne simule pas, et ne prétend pas simuler : la proximité de sens
// entre deux mots différents. Que « colère » soit proche de « mécontent »
// relève du vrai modèle, et se vérifie à la main (§3.10).
type FakeEmbeddingProvider struct {
	mu sync.Mutex

	// Calls compte les appels à Embed. C'est ce qui permet de vérifier qu'un
	// changement de métadonnée ne déclenche aucun calcul (§2.12).
	Calls int

	// Unavailable simule l'absence de modèle, pour tester le chemin d'erreur
	// sans retirer de fichier.
	Unavailable bool
}

// NewFakeEmbeddingProvider construit un fournisseur factice disponible.
func NewFakeEmbeddingProvider() *FakeEmbeddingProvider { return &FakeEmbeddingProvider{} }

// Available indique si le fournisseur accepte de vectoriser.
func (p *FakeEmbeddingProvider) Available() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.Unavailable
}

// CallCount rend le nombre de vectorisations effectuées.
func (p *FakeEmbeddingProvider) CallCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.Calls
}

// Reset remet le compteur à zéro entre deux phases d'un test.
func (p *FakeEmbeddingProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Calls = 0
}

// Embed rend le vecteur déterministe du texte.
func (p *FakeEmbeddingProvider) Embed(text string) ([]float32, error) {
	p.mu.Lock()
	if p.Unavailable {
		p.mu.Unlock()
		return nil, domain.ErrModelUnavailable
	}
	p.Calls++
	p.mu.Unlock()

	vec := make([]float32, FakeDimensions)
	for _, mot := range decouper(text) {
		h := fnv.New32a()
		h.Write([]byte(mot))
		vec[h.Sum32()%FakeDimensions]++
	}

	// Normalisation : sans elle, un texte long serait « plus proche » de tout
	// qu'un texte court, alors que la similarité cosinus doit être insensible à
	// la longueur.
	var norme float64
	for _, v := range vec {
		norme += float64(v) * float64(v)
	}
	if norme == 0 {
		// Un texte vide n'a aucun mot : on rend un vecteur non nul et constant
		// plutôt qu'un vecteur de norme nulle, que la similarité refuserait.
		for i := range vec {
			vec[i] = 1
		}
		norme = float64(FakeDimensions)
	}
	facteur := float32(1 / math.Sqrt(norme))
	for i := range vec {
		vec[i] *= facteur
	}
	return vec, nil
}

// decouper rend les mots d'un texte, en minuscules.
func decouper(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// EmbedQuery rend le même vecteur qu'Embed.
//
// Le Fake ignore volontairement la distinction requête/document : elle relève
// des préfixes du vrai modèle e5, pas d'une règle de l'application. La simuler
// rendrait un texte différent de lui-même selon le rôle, et les tests de tri
// perdraient leur lisibilité sans rien vérifier de plus.
func (p *FakeEmbeddingProvider) EmbedQuery(text string) ([]float32, error) {
	return p.Embed(text)
}
