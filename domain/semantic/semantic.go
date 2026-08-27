// Package semantic porte les règles de la recherche sémantique (§2.12).
//
// Rien ici ne connaît ONNX, ni SQLite, ni le format de sérialisation des
// vecteurs. Le package reçoit des tranches de float32 et rend des scores :
// c'est ce qui permet de tester l'intégralité des règles avec un fournisseur
// factice, sans charger un modèle de 120 Mo à chaque `go test` (§3.9).
package semantic

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"sort"
	"strings"
)

// ErrDimensionMismatch : deux vecteurs de tailles différentes.
//
// Le cas n'est pas théorique : il se produit dès qu'un vecteur a été calculé
// par un modèle et comparé à un vecteur produit par un autre. Plutôt que de
// comparer les premières composantes et de rendre un score dépourvu de sens,
// on refuse (§3.2, colonne `dimensions`).
var ErrDimensionMismatch = errors.New("vecteurs de dimensions différentes")

// ErrEmptyVector : vecteur vide, ou de norme nulle.
//
// Un vecteur nul n'a pas de direction : sa similarité cosinus avec quoi que ce
// soit n'est pas définie, et la formule donnerait une division par zéro.
var ErrEmptyVector = errors.New("vecteur vide ou de norme nulle")

// Cosine rend la similarité cosinus de deux vecteurs, entre -1 et 1.
//
// C'est la mesure retenue par le §3.2 : un balayage complet avec ce calcul en
// Go pur reste sous la milliseconde sur quelques milliers de vecteurs, ce qui
// évite d'ajouter une extension vectorielle native.
func Cosine(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, ErrDimensionMismatch
	}
	if len(a) == 0 {
		return 0, ErrEmptyVector
	}

	// Les accumulateurs sont en float64 alors que les vecteurs sont en float32 :
	// sur 384 composantes, sommer en float32 accumule une erreur visible sur les
	// dernières décimales, et la similarité d'un vecteur avec lui-même ne
	// tomberait plus exactement sur 1.
	var produit, normeA, normeB float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		produit += x * y
		normeA += x * x
		normeB += y * y
	}
	if normeA == 0 || normeB == 0 {
		return 0, ErrEmptyVector
	}

	score := produit / (math.Sqrt(normeA) * math.Sqrt(normeB))

	// L'arithmétique flottante peut dépasser très légèrement les bornes — un
	// vecteur comparé à lui-même sort parfois à 1.0000000000000002. On ramène
	// dans l'intervalle plutôt que de laisser fuiter une valeur impossible.
	return math.Max(-1, math.Min(1, score)), nil
}

// Candidate est un vecteur candidat au classement, identifié par son entité.
type Candidate struct {
	EntityID string
	Vector   []float32
}

// Match est un résultat classé.
type Match struct {
	EntityID string  `json:"entityId"`
	Score    float64 `json:"score"`
}

// Rank classe les candidats par similarité décroissante avec la requête (§2.12).
//
// Les candidats de dimension incompatible sont **ignorés, pas remontés en
// erreur** : ils viennent d'un modèle antérieur, et faire échouer toute la
// recherche parce qu'un vieux vecteur traîne en base priverait l'utilisateur
// des résultats valides. Ils disparaîtront à la prochaine revectorisation.
//
// Le tri est stable : à score égal, l'ordre d'entrée est conservé, ce qui rend
// deux recherches identiques comparables.
func Rank(query []float32, candidates []Candidate, minScore float64) []Match {
	matches := make([]Match, 0, len(candidates))
	for _, c := range candidates {
		score, err := Cosine(query, c.Vector)
		if err != nil {
			continue
		}
		if score < minScore {
			continue
		}
		matches = append(matches, Match{EntityID: c.EntityID, Score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	return matches
}

// DefaultMinScore est un plancher de sécurité, pas un filtre de pertinence.
//
// **L'échelle de e5 est tassée** : mesuré sur ce corpus, tout se situe entre
// 0,79 et 0,88, y compris une requête sans aucun rapport avec le document. Un
// seuil absolu ne discrimine donc rien, et le régler haut reviendrait à le
// caler sur un corpus particulier. Il ne sert qu'à écarter l'aberration — un
// vecteur corrompu, un texte dégénéré.
//
// C'est CutRelative qui fait le vrai travail de coupe.
const DefaultMinScore = 0.70

// DefaultMargin est l'écart au meilleur score au-delà duquel un résultat est
// écarté.
//
// Mesures qui l'ont fixé : sur une requête où le modèle a un avis net
// (« réuion de cadrage »), le premier devance le deuxième de 0,039 ; sur une
// requête où il n'en a aucun (« gâteau »), les six candidats tiennent dans
// 0,019. Une marge de 0,03 coupe donc la traîne quand une réponse se détache,
// et laisse la liste entière quand rien ne se détache — ce qui est honnête :
// dans ce cas, le modèle ne sait effectivement pas trancher.
const DefaultMargin = 0.03

// MaxResults borne la liste rendue au frontend.
const MaxResults = 50

// CutRelative ne garde que les résultats proches du meilleur.
//
// Elle s'applique **après** Rank, sur une liste déjà triée. La coupe est
// relative et non absolue parce que la similarité cosinus de e5 n'a pas de
// signification absolue : seul l'écart entre deux résultats d'une même requête
// en a une.
func CutRelative(matches []Match, margin float64) []Match {
	if len(matches) == 0 || margin <= 0 {
		return matches
	}
	plancher := matches[0].Score - margin
	for i, m := range matches {
		if m.Score < plancher {
			return matches[:i]
		}
	}
	return matches
}

// Hash rend l'empreinte du texte vectorisé (§3.2, colonne `source_hash`).
//
// Elle sert à ne pas recalculer un vecteur pour un texte inchangé. La
// normalisation des espaces avant hachage est délibérée : la sauvegarde
// automatique réenregistre du HTML produit par TipTap, dont l'indentation peut
// varier sans que le texte ait bougé.
func Hash(text string) string {
	somme := sha256.Sum256([]byte(strings.Join(strings.Fields(text), " ")))
	return hex.EncodeToString(somme[:])
}

// SourceText assemble le texte à vectoriser depuis les champs d'une entité.
//
// Les champs sont joints par un saut de ligne et les vides sont écartés : un
// séparateur isolé ferait changer l'empreinte d'une note dont on vide le titre,
// pour un contenu identique.
func SourceText(champs ...string) string {
	retenus := make([]string, 0, len(champs))
	for _, c := range champs {
		if c = strings.TrimSpace(c); c != "" {
			retenus = append(retenus, c)
		}
	}
	return strings.Join(retenus, "\n")
}
