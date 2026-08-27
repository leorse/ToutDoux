// Package onnx implémente le fournisseur de vecteurs réel (§2.12, étape 5.7).
//
// Il contient deux choses : un tokenizer XLM-RoBERTa en Go pur, et
// l'inférence ONNX. Le premier existe parce qu'aucune bibliothèque Go ne sait
// lire le `tokenizer.json` de ce modèle — voir tokenizerPipeline ci-dessous.
package onnx

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Identifiants des jetons spéciaux, lus dans le `tokenizer.json` du modèle et
// figés ici parce qu'ils font partie du contrat du modèle, pas d'un réglage.
const (
	idDebut  = 0 // <s>
	idFin    = 2 // </s>
	idInconn = 3 // <unk>
)

// metaspace est le caractère qui remplace l'espace dans SentencePiece.
const metaspace = "▁" // ▁

// maxTokens borne la séquence transmise au modèle.
//
// multilingual-e5-small est un XLM-RoBERTa à 512 positions : au-delà,
// l'inférence échoue. Les deux places réservées sont <s> et </s>.
const maxTokens = 512

// maxContenu est le nombre de jetons de contenu admis.
const maxContenu = maxTokens - 2

// tokenizerPipeline documente ce que fait ce fichier, et pourquoi il existe.
//
// Le `tokenizer.json` de multilingual-e5-small décrit ce pipeline :
//
//	normalizer   : Sequence[ Precompiled(charsmap), Replace(/ {2,}/ → " ") ]
//	pre_tokenizer: Metaspace(replacement "▁", add_prefix_space true)
//	model        : Unigram(unk_id 3, 250k pièces avec log-probabilité)
//	post_process : TemplateProcessing → <s> A </s>
//
// Aucune bibliothèque Go pure ne prend en charge le couple Unigram +
// Precompiled : `sugarme/tokenizer` annonce BERT, RoBERTa et GPT-2, et les
// autres pistes passent par des liaisons Rust, donc par CGO côté build. Les
// quatre étapes sont réimplémentées ici.
//
// **Un seul écart, assumé et localisé : le normaliseur.** `Precompiled` est une
// table de correspondance SentencePiece sérialisée (trie à double tableau)
// d'environ 200 Ko, dont le décodage représenterait à lui seul plus de code que
// tout le reste de ce fichier. Elle applique, pour l'essentiel, **NFKC**. On
// applique donc NFKC, que `golang.org/x/text` fournit déjà — la dépendance est
// présente dans le module.
//
// Ce que cet écart peut coûter : sur quelques caractères exotiques, un
// découpage légèrement différent de celui de la bibliothèque de référence. Ce
// qu'il ne coûte pas : la cohérence. Requête et documents passent par ce même
// tokenizer, donc les vecteurs restent comparables entre eux — ce qui est la
// seule propriété dont la recherche a besoin.
type tokenizerPipeline struct{}

// tokenizer découpe un texte en identifiants de jetons.
type tokenizer struct {
	// pieces associe une pièce du vocabulaire à son identifiant et à son score.
	pieces map[string]piece

	// maxRunes est la longueur de la plus longue pièce, en runes. Elle borne la
	// recherche de Viterbi : au-delà, aucune pièce ne peut correspondre.
	maxRunes int

	// penaliteInconnu est le score attribué à une rune absente du vocabulaire.
	// Volontairement très bas : un découpage qui l'évite doit toujours gagner.
	penaliteInconnu float64
}

type piece struct {
	id    int
	score float64
}

// entreeVocab lit une entrée `["▁le", -7.42]` du vocabulaire.
type entreeVocab struct {
	texte string
	score float64
}

func (e *entreeVocab) UnmarshalJSON(data []byte) error {
	var paire [2]json.RawMessage
	if err := json.Unmarshal(data, &paire); err != nil {
		return err
	}
	if err := json.Unmarshal(paire[0], &e.texte); err != nil {
		return err
	}
	return json.Unmarshal(paire[1], &e.score)
}

// fichierTokenizer ne déclare que ce qu'on lit.
//
// Les champs non déclarés — dont `precompiled_charsmap`, plusieurs centaines de
// kilo-octets de base64 — sont ignorés par `encoding/json` sans être alloués.
type fichierTokenizer struct {
	Model struct {
		Type  string        `json:"type"`
		UnkID int           `json:"unk_id"`
		Vocab []entreeVocab `json:"vocab"`
	} `json:"model"`
}

// chargerTokenizer lit le `tokenizer.json` déposé par l'utilisateur.
func chargerTokenizer(chemin string) (*tokenizer, error) {
	brut, err := os.ReadFile(chemin)
	if err != nil {
		return nil, fmt.Errorf("lecture du tokenizer : %w", err)
	}

	var f fichierTokenizer
	if err := json.Unmarshal(brut, &f); err != nil {
		return nil, fmt.Errorf("tokenizer illisible : %w", err)
	}
	if f.Model.Type != "Unigram" {
		// Un autre type de modèle voudrait un autre algorithme de découpage :
		// mieux vaut le dire que de produire des jetons silencieusement faux.
		return nil, fmt.Errorf("tokenizer de type %q, attendu Unigram", f.Model.Type)
	}
	if len(f.Model.Vocab) == 0 {
		return nil, fmt.Errorf("vocabulaire vide")
	}

	t := &tokenizer{pieces: make(map[string]piece, len(f.Model.Vocab))}
	pire := 0.0
	for id, e := range f.Model.Vocab {
		// Le premier identifiant gagne : le vocabulaire n'a pas de doublon,
		// mais écraser silencieusement fausserait les identifiants s'il en
		// apparaissait un.
		if _, deja := t.pieces[e.texte]; !deja {
			t.pieces[e.texte] = piece{id: id, score: e.score}
		}
		if n := utf8.RuneCountInString(e.texte); n > t.maxRunes {
			t.maxRunes = n
		}
		if e.score < pire {
			pire = e.score
		}
	}
	t.penaliteInconnu = pire - 10
	return t, nil
}

// Encode rend les identifiants de jetons d'un texte, jetons spéciaux compris.
func (t *tokenizer) Encode(texte string) []int64 {
	runes := []rune(preTokeniser(normaliser(texte)))

	ids := t.viterbi(runes)
	if len(ids) > maxContenu {
		// Troncature plutôt qu'erreur : une note longue doit rester indexable,
		// et le début du texte porte l'essentiel du sujet.
		ids = ids[:maxContenu]
	}

	out := make([]int64, 0, len(ids)+2)
	out = append(out, idDebut)
	out = append(out, ids...)
	return append(out, idFin)
}

// normaliser applique le normaliseur du modèle, à l'approximation près
// documentée sur tokenizerPipeline.
func normaliser(texte string) string {
	// NFKC tient lieu de table Precompiled.
	texte = norm.NFKC.String(texte)
	// Puis la seconde étape, celle-là exacte : les suites d'espaces sont
	// réduites à un seul. Strings.Fields traite au passage tabulations et
	// sauts de ligne, que le texte issu de TipTap contient en quantité.
	return strings.Join(strings.Fields(texte), " ")
}

// preTokeniser applique Metaspace : l'espace devient ▁, et le texte en reçoit
// un en tête.
func preTokeniser(texte string) string {
	if texte == "" {
		return ""
	}
	return metaspace + strings.ReplaceAll(texte, " ", metaspace)
}

// viterbi rend le découpage de score total maximal.
//
// C'est l'algorithme d'Unigram : chaque pièce du vocabulaire porte une
// log-probabilité, et le meilleur découpage est celui qui maximise leur somme.
// Un simple « plus longue pièce d'abord » donnerait un découpage différent, et
// donc des vecteurs qui ne correspondraient pas à ce que le modèle a appris.
func (t *tokenizer) viterbi(runes []rune) []int64 {
	n := len(runes)
	if n == 0 {
		return nil
	}

	// meilleur[i] : score du meilleur découpage des i premières runes.
	meilleur := make([]float64, n+1)
	// depuis[i] : début de la dernière pièce de ce découpage.
	depuis := make([]int, n+1)
	// jeton[i] : identifiant de cette dernière pièce.
	jeton := make([]int64, n+1)

	for i := 1; i <= n; i++ {
		meilleur[i] = math.Inf(-1)
	}

	for i := 0; i < n; i++ {
		if math.IsInf(meilleur[i], -1) {
			continue
		}
		trouve := false

		max := t.maxRunes
		if i+max > n {
			max = n - i
		}
		for l := 1; l <= max; l++ {
			p, ok := t.pieces[string(runes[i:i+l])]
			if !ok {
				continue
			}
			trouve = true
			if s := meilleur[i] + p.score; s > meilleur[i+l] {
				meilleur[i+l], depuis[i+l], jeton[i+l] = s, i, int64(p.id)
			}
		}

		// Aucune pièce ne commence ici : la rune est inconnue du modèle. On
		// avance d'une rune en payant la pénalité, plutôt que d'abandonner —
		// un seul caractère exotique ne doit pas rendre une note inindexable.
		if !trouve {
			if s := meilleur[i] + t.penaliteInconnu; s > meilleur[i+1] {
				meilleur[i+1], depuis[i+1], jeton[i+1] = s, i, idInconn
			}
		}
	}

	if math.IsInf(meilleur[n], -1) {
		return nil
	}

	// Remontée du chemin, puis inversion.
	var envers []int64
	for i := n; i > 0; i = depuis[i] {
		envers = append(envers, jeton[i])
	}
	ids := make([]int64, len(envers))
	for i, id := range envers {
		ids[len(envers)-1-i] = id
	}
	return ids
}

// pieceParID reconstruit la table inverse, pour les tests et le diagnostic.
//
// Elle n'est pas construite au chargement : l'inférence n'en a jamais besoin,
// et 250 000 entrées de plus coûteraient de la mémoire pour rien.
func (t *tokenizer) pieceParID() map[int64]string {
	out := make(map[int64]string, len(t.pieces))
	for texte, p := range t.pieces {
		out[int64(p.id)] = texte
	}
	return out
}
