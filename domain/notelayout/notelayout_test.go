package notelayout

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"toutdoux/domain"
)

// parse lit une disposition écrite « A B:g1 C:g1 D » : une note par mot, son
// groupe éventuel après les deux-points.
func parse(s string) []Item {
	out := []Item{}
	for _, mot := range strings.Fields(s) {
		note, groupe, _ := strings.Cut(mot, ":")
		out = append(out, Item{NoteID: note, GroupID: groupe})
	}
	return out
}

func TestGroup(t *testing.T) {
	cas := []struct {
		nom, depart string
		selection   []string
		attendu     string
	}{
		{"trois notes éparses", "A B C D", []string{"A", "C", "D"}, "A:n C:n D:n B"},
		{"l'ordre de la liste prime sur celui de la sélection", "A B C D", []string{"D", "A"}, "A:n D:n B C"},
		{"une seule note", "A B C", []string{"B"}, "A B:n C"},
		{"note prise à un autre groupe", "A:g1 B:g1 C", []string{"B", "C"}, "A:g1 B:n C:n"},
		{"première note au milieu d'un groupe : posé après lui", "A:g1 B:g1 C:g1 X", []string{"B", "X"}, "A:g1 C:g1 B:n X:n"},
		{"groupe entièrement repris", "A:g1 B:g1 C", []string{"A", "B"}, "A:n B:n C"},
		{"sélection inconnue", "A B", []string{"Z"}, "A B"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			obtenu := Group(parse(c.depart), c.selection, "n")
			if !reflect.DeepEqual(obtenu, parse(c.attendu)) {
				t.Errorf("obtenu %v, attendu %v", obtenu, parse(c.attendu))
			}
		})
	}
}

func TestDissolve(t *testing.T) {
	obtenu := Dissolve(parse("A B:g C:g D:h"), "g")
	if attendu := parse("A B C D:h"); !reflect.DeepEqual(obtenu, attendu) {
		t.Errorf("obtenu %v, attendu %v", obtenu, attendu)
	}
}

func TestValidate(t *testing.T) {
	notes := []string{"A", "B", "C", "D"}
	groupes := []string{"g", "h"}
	cas := []struct {
		nom, layout string
		valide      bool
	}{
		{"sans groupe", "D C B A", true},
		{"deux groupes contigus", "A:g B:g C:h D", true},
		{"note manquante", "A B C", false},
		{"note en double", "A A B C", false},
		{"note inconnue", "A B C Z", false},
		{"groupe inconnu", "A:x B C D", false},
		{"groupe coupé par une note libre", "A:g B C:g D", false},
		{"groupe coupé par un autre groupe", "A:g B:h C:g D", false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			err := Validate(parse(c.layout), notes, groupes)
			if c.valide && err != nil {
				t.Errorf("erreur inattendue : %v", err)
			}
			if !c.valide && !errors.Is(err, domain.ErrInvalidLayout) {
				t.Errorf("erreur = %v, attendu ErrInvalidLayout", err)
			}
		})
	}
}

func TestFromNotes(t *testing.T) {
	g := "g"
	obtenu := FromNotes([]domain.Note{{ID: "A"}, {ID: "B", GroupID: &g}})
	if attendu := parse("A B:g"); !reflect.DeepEqual(obtenu, attendu) {
		t.Errorf("obtenu %v, attendu %v", obtenu, attendu)
	}
}
