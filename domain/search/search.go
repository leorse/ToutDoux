// Package search porte la recherche globale (§2.9).
//
// Deux écarts délibérés avec le prototype :
//
// La signature perd les paramètres scope et selectedProjectId. Ils étaient les
// vestiges d'un mode « Projet » que la v2 a retiré : le seul appel réel passait
// « global » en dur. La recherche est toujours globale, il n'y a pas de variante.
//
// Highlight ne rend plus du JSX mais une structure. Dans le prototype la
// fonction renvoyait un fragment React contenant un <mark>, ce qui n'est pas
// du domaine mais du rendu. Ici le domaine découpe, le frontend décide de la
// balise — et la découpe devient testable sans DOM.
package search

import (
	"sort"
	"strings"
	"time"

	"toutdoux/domain"
)

// MinQueryLength est la longueur à partir de laquelle une recherche se
// déclenche : au-delà de 2 caractères, donc 3 (§2.9).
const MinQueryLength = 3

// DefaultContextChars est le nombre de caractères conservés de part et d'autre
// de l'occurrence dans l'extrait.
const DefaultContextChars = 90

// Type est la nature d'un résultat, qui détermine son icône (§2.9).
type Type string

const (
	TypeTask    Type = "task"    // case à cocher réelle + pastille d'importance
	TypeNote    Type = "note"    // 📝
	TypeMeeting Type = "meeting" // 📞
)

// Result est une entrée de la liste de résultats (§2.9).
//
// L'entité d'origine est jointe parce que l'affichage en a besoin : la ligne
// d'une tâche montre son état coché et une pastille colorée, pas un simple ✓.
type Result struct {
	Type      Type       `json:"type"`
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	ProjectID string     `json:"projectId"`
	Date      *time.Time `json:"date"`

	Task     *domain.Task            `json:"task,omitempty"`
	Note     *domain.Note            `json:"note,omitempty"`
	Meeting  *domain.Meeting         `json:"meeting,omitempty"`
	Instance *domain.MeetingInstance `json:"instance,omitempty"`
}

// Sort ordonne les résultats pour l'affichage (§2.9).
//
// Plus récent d'abord ; les résultats sans date passent en dernier — une tâche
// sans échéance n'est pas « ancienne », elle est simplement non datée, et la
// reléguer en fin de liste vaut mieux que de lui inventer une place.
//
// L'ordre est décidé ici et non en SQL : c'est une règle d'affichage, elle ne
// doit pas dépendre du moteur d'indexation, que le §3.9 prévoit remplaçable.
func Sort(results []Result) []Result {
	out := make([]Result, len(results))
	copy(out, results)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Date, out[j].Date
		switch {
		case a != nil && b != nil:
			return a.After(*b)
		case a != nil:
			return true
		default:
			return false
		}
	})
	return out
}

// Part est un fragment d'extrait : du texte ordinaire, ou l'occurrence trouvée.
type Part struct {
	Text  string `json:"text"`
	Match bool   `json:"match"` // le frontend rend true en <mark> jaune (§2.9)
}

// Snippet est l'extrait affiché dans le panneau de détail (§2.9).
type Snippet struct {
	LeadingEllipsis  bool   `json:"leadingEllipsis"`
	Parts            []Part `json:"parts"`
	TrailingEllipsis bool   `json:"trailingEllipsis"`
}

// Text rend l'extrait à plat, sans balisage. Utile aux tests et aux contextes
// qui ne savent pas rendre la surbrillance, comme le menu de la barre système.
func (s Snippet) Text() string {
	var b strings.Builder
	if s.LeadingEllipsis {
		b.WriteString("…")
	}
	for _, p := range s.Parts {
		b.WriteString(p.Text)
	}
	if s.TrailingEllipsis {
		b.WriteString("…")
	}
	return b.String()
}

// Highlight découpe un texte autour de la première occurrence du terme cherché.
//
// Le découpage se fait en runes et non en octets : « réunion » et « échéance »
// sont partout dans ce corpus francophone, et raisonner en octets couperait un
// caractère accentué en deux, produisant un extrait invalide.
//
// Sans occurrence, rend le début du texte tronqué — le résultat peut avoir été
// trouvé sur un autre champ, il faut quand même montrer quelque chose.
func Highlight(text, query string, contextChars int) *Snippet {
	if text == "" {
		return nil
	}
	if contextChars <= 0 {
		contextChars = DefaultContextChars
	}

	runes := []rune(text)
	lower := []rune(strings.ToLower(text))
	q := []rune(strings.ToLower(strings.TrimSpace(query)))

	idx := -1
	if len(q) > 0 && len(q) <= len(lower) {
		// Recherche sur les runes minuscules ; l'index vaut pour `runes`, car
		// la mise en minuscules du français conserve le nombre de runes.
		idx = indexRunes(lower, q)
	}

	if idx == -1 {
		const maxPlain = 160
		if len(runes) > maxPlain {
			return &Snippet{Parts: []Part{{Text: string(runes[:maxPlain])}}, TrailingEllipsis: true}
		}
		return &Snippet{Parts: []Part{{Text: text}}}
	}

	start := max(0, idx-contextChars)
	end := min(len(runes), idx+len(q)+contextChars)

	s := &Snippet{LeadingEllipsis: start > 0, TrailingEllipsis: end < len(runes)}
	if before := string(runes[start:idx]); before != "" {
		s.Parts = append(s.Parts, Part{Text: before})
	}
	s.Parts = append(s.Parts, Part{Text: string(runes[idx : idx+len(q)]), Match: true})
	if after := string(runes[idx+len(q) : end]); after != "" {
		s.Parts = append(s.Parts, Part{Text: after})
	}
	return s
}

// indexRunes rend l'index de la première occurrence de needle dans haystack,
// exprimé en runes, ou -1.
func indexRunes(haystack, needle []rune) int {
	if len(needle) == 0 {
		return -1
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		found := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}
