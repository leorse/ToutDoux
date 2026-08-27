package onnx

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"toutdoux/adapters/model"
)

// Ces tests tournent contre le **vrai** `tokenizer.json` déposé par
// l'utilisateur, et se sautent s'il est absent (§3.10).
//
// C'est délibéré : un tokenizer se valide sur son fichier réel, pas sur un
// fichier de démonstration. Un vocabulaire fabriqué pour le test prouverait que
// l'algorithme tourne, pas qu'il découpe le français comme le modèle l'attend.

func chargerVrai(t *testing.T) *tokenizer {
	t.Helper()

	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("dossier de configuration indisponible : %v", err)
	}
	dir := model.Dir(filepath.Join(base, "ToutDoux", "app.db"))
	chemin := filepath.Join(dir, model.FichierTokenizer)
	if _, err := os.Stat(chemin); err != nil {
		t.Skipf("modèle non déposé dans %s : la vérification du tokenizer demande le vrai fichier", dir)
	}

	tok, err := chargerTokenizer(chemin)
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	return tok
}

func TestTokenizer_VocabulaireAttendu(t *testing.T) {
	tok := chargerVrai(t)

	if len(tok.pieces) < 200_000 {
		t.Fatalf("%d pièces, attendu ~250 000 pour multilingual-e5-small", len(tok.pieces))
	}
	// Les quatre jetons spéciaux sont à leur place : c'est le contrat du modèle,
	// et une erreur ici décalerait tous les identifiants.
	for texte, attendu := range map[string]int{"<s>": 0, "<pad>": 1, "</s>": 2, "<unk>": 3} {
		p, ok := tok.pieces[texte]
		if !ok {
			t.Fatalf("jeton spécial %q absent du vocabulaire", texte)
		}
		if p.id != attendu {
			t.Fatalf("%q a l'identifiant %d, attendu %d", texte, p.id, attendu)
		}
	}
}

func TestTokenizer_EncadrementParLesJetonsSpeciaux(t *testing.T) {
	tok := chargerVrai(t)

	ids := tok.Encode("Le client est mécontent")
	if len(ids) < 3 {
		t.Fatalf("%d jetons, attendu au moins <s> contenu </s>", len(ids))
	}
	if ids[0] != idDebut || ids[len(ids)-1] != idFin {
		t.Fatalf("encadrement = %d … %d, attendu %d … %d", ids[0], ids[len(ids)-1], idDebut, idFin)
	}
}

// Le découpage doit **couvrir exactement** le texte pré-tokenisé.
//
// C'est la propriété qui attrape l'essentiel des erreurs possibles : un chemin
// de Viterbi mal remonté, une pièce sautée, un décalage d'indice. Elle se
// vérifie sans rien connaître du modèle.
func TestTokenizer_LeDecoupageCouvreLeTexte(t *testing.T) {
	tok := chargerVrai(t)
	parID := tok.pieceParID()

	for _, texte := range []string{
		"Le client est mécontent depuis la livraison",
		"Réunion du 12/08 : échéance repoussée au 30",
		"Préparer la réunion de cadrage",
		"aujourd'hui, mi-parcours — çà et là",
		"Émoji 🎉 et caractère rare ෴ mêlés",
	} {
		ids := tok.Encode(texte)
		var reconstruit strings.Builder
		for _, id := range ids[1 : len(ids)-1] {
			p, ok := parID[id]
			if !ok {
				t.Fatalf("identifiant %d hors vocabulaire", id)
			}
			if id == idInconn {
				// Une rune inconnue ne se reconstruit pas : on ne peut pas
				// exiger la couverture exacte sur ces textes-là.
				reconstruit.Reset()
				break
			}
			reconstruit.WriteString(p)
		}
		if reconstruit.Len() == 0 {
			continue
		}
		attendu := preTokeniser(normaliser(texte))
		if reconstruit.String() != attendu {
			t.Fatalf("texte %q :\n  reconstruit %q\n  attendu     %q", texte, reconstruit.String(), attendu)
		}
	}
}

// Le découpage doit être **optimal**, pas seulement valide.
//
// Un « plus longue pièce d'abord » couvrirait aussi le texte, mais donnerait
// des jetons que le modèle n'a pas appris à voir ensemble. On compare donc au
// score d'un découpage glouton : Viterbi ne doit jamais faire moins bien.
func TestTokenizer_ViterbiBatLeGlouton(t *testing.T) {
	tok := chargerVrai(t)

	for _, texte := range []string{
		"Le client est mécontent depuis la livraison",
		"Préparer la réunion de cadrage du comité de pilotage",
	} {
		runes := []rune(preTokeniser(normaliser(texte)))
		optimal := tok.score(tok.viterbi(runes), runes)
		glouton := tok.score(tok.plusLonguePieceDAbord(runes), runes)

		if optimal < glouton-1e-9 {
			t.Fatalf("texte %q : Viterbi %.4f < glouton %.4f", texte, optimal, glouton)
		}
	}
}

func TestTokenizer_Deterministe(t *testing.T) {
	tok := chargerVrai(t)
	a := tok.Encode("Le client est mécontent")
	b := tok.Encode("Le client est mécontent")
	if len(a) != len(b) {
		t.Fatalf("longueurs différentes : %d et %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("jeton %d : %d puis %d", i, a[i], b[i])
		}
	}
}

func TestTokenizer_NormalisationDesEspaces(t *testing.T) {
	tok := chargerVrai(t)
	// Le texte issu de TipTap arrive avec sauts de ligne et indentation : ils
	// ne doivent pas produire un découpage différent.
	a := tok.Encode("Le client\n   est   mécontent\t")
	b := tok.Encode("Le client est mécontent")
	if len(a) != len(b) {
		t.Fatalf("%d jetons contre %d : les espaces ne sont pas normalisés", len(a), len(b))
	}
}

func TestTokenizer_TexteVide(t *testing.T) {
	tok := chargerVrai(t)
	ids := tok.Encode("   ")
	if len(ids) != 2 || ids[0] != idDebut || ids[1] != idFin {
		t.Fatalf("texte vide = %v, attendu seulement les deux jetons spéciaux", ids)
	}
}

func TestTokenizer_TexteTrongueALaLimiteDuModele(t *testing.T) {
	tok := chargerVrai(t)
	long := strings.Repeat("réunion de cadrage avec le client mécontent ", 400)
	ids := tok.Encode(long)
	// Au-delà de 512 positions, XLM-RoBERTa échoue : la troncature n'est pas un
	// confort, c'est une condition de fonctionnement.
	if len(ids) != maxTokens {
		t.Fatalf("%d jetons, attendu exactement %d", len(ids), maxTokens)
	}
	if ids[len(ids)-1] != idFin {
		t.Fatal("le jeton de fin doit survivre à la troncature")
	}
}

/* ---------------- Aides de test ---------------- */

// score rend le score total d'un découpage, pour comparer deux stratégies.
func (t *tokenizer) score(ids []int64, runes []rune) float64 {
	parID := t.pieceParID()
	total := 0.0
	for _, id := range ids {
		if p, ok := t.pieces[parID[id]]; ok {
			total += p.score
		} else {
			total += t.penaliteInconnu
		}
	}
	return total
}

// plusLonguePieceDAbord est la stratégie naïve, présente uniquement pour
// servir de point de comparaison au test ci-dessus.
func (t *tokenizer) plusLonguePieceDAbord(runes []rune) []int64 {
	var ids []int64
	for i := 0; i < len(runes); {
		max := t.maxRunes
		if i+max > len(runes) {
			max = len(runes) - i
		}
		avance := 0
		for l := max; l >= 1; l-- {
			if p, ok := t.pieces[string(runes[i:i+l])]; ok {
				ids = append(ids, int64(p.id))
				avance = l
				break
			}
		}
		if avance == 0 {
			ids = append(ids, idInconn)
			avance = 1
		}
		i += avance
	}
	return ids
}

var _ = math.Inf
