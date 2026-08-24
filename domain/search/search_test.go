package search

import (
	"testing"
	"time"
)

func TestHighlight_SplitsAroundMatch(t *testing.T) {
	s := Highlight("avant réunion après", "réunion", 90)
	if s == nil {
		t.Fatal("extrait nil")
	}
	if len(s.Parts) != 3 {
		t.Fatalf("%d fragments, 3 attendus", len(s.Parts))
	}
	if s.Parts[0].Text != "avant " || s.Parts[0].Match {
		t.Errorf("fragment 0 = %+v", s.Parts[0])
	}
	if s.Parts[1].Text != "réunion" || !s.Parts[1].Match {
		t.Errorf("fragment 1 = %+v, doit être l'occurrence", s.Parts[1])
	}
	if s.Parts[2].Text != " après" || s.Parts[2].Match {
		t.Errorf("fragment 2 = %+v", s.Parts[2])
	}
	if s.LeadingEllipsis || s.TrailingEllipsis {
		t.Error("texte court : pas de points de suspension")
	}
}

// L'occurrence conserve la casse du texte source, pas celle de la requête.
func TestHighlight_PreservesOriginalCase(t *testing.T) {
	s := Highlight("La Réunion annuelle", "réunion", 90)
	if s.Parts[1].Text != "Réunion" {
		t.Errorf("occurrence = %q, attendu \"Réunion\"", s.Parts[1].Text)
	}
}

// Le découpage se fait en runes : en octets, les accents qui précèdent
// l'occurrence décaleraient les bornes et couperaient un caractère en deux.
func TestHighlight_RuneOffsetsNotBytes(t *testing.T) {
	text := "ééééé cible fin"
	s := Highlight(text, "cible", 90)
	if s.Parts[0].Text != "ééééé " {
		t.Errorf("avant = %q, attendu \"ééééé \"", s.Parts[0].Text)
	}
	if s.Parts[1].Text != "cible" {
		t.Errorf("occurrence = %q", s.Parts[1].Text)
	}
	if got := s.Text(); got != text {
		t.Errorf("recomposition = %q, attendu %q", got, text)
	}
}
func TestHighlight_TrimsContextAndAddsEllipsis(t *testing.T) {
	long := ""
	for i := 0; i < 50; i++ {
		long += "abcde "
	}
	s := Highlight(long+"cible"+long, "cible", 10)
	if !s.LeadingEllipsis || !s.TrailingEllipsis {
		t.Error("un texte long des deux côtés doit porter les deux ellipses")
	}
	// 10 caractères de contexte de chaque côté, plus l'occurrence.
	if n := len([]rune(s.Parts[0].Text)); n != 10 {
		t.Errorf("contexte avant = %d runes, attendu 10", n)
	}
	if n := len([]rune(s.Parts[2].Text)); n != 10 {
		t.Errorf("contexte après = %d runes, attendu 10", n)
	}
}

// Sans occurrence, il faut quand même montrer quelque chose : le résultat peut
// avoir été trouvé sur un autre champ.
func TestHighlight_NoMatchFallsBackToPlainText(t *testing.T) {
	s := Highlight("texte court", "absent", 90)
	if s == nil {
		t.Fatal("extrait nil")
	}
	if len(s.Parts) != 1 || s.Parts[0].Match {
		t.Errorf("fragments = %+v, un seul fragment non surligné attendu", s.Parts)
	}
	if s.Parts[0].Text != "texte court" {
		t.Errorf("texte = %q", s.Parts[0].Text)
	}
}
func TestHighlight_EmptyText(t *testing.T) {
	if s := Highlight("", "réunion", 90); s != nil {
		t.Errorf("extrait = %+v, nil attendu sur un texte vide", s)
	}
}

// Plus récent d'abord ; sans date en dernier (§2.9).
func TestSort(t *testing.T) {
	ancien := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	entree := []Result{
		{ID: "sansDate"},
		{ID: "ancienne", Date: &ancien},
		{ID: "recente", Date: &recent},
	}
	got := Sort(entree)
	want := []string{"recente", "ancienne", "sansDate"}
	for i := range want {
		if got[i].ID != want[i] {
			t.Fatalf("ordre = %v, attendu %v", ids(got), want)
		}
	}
	// La tranche d'entrée ne doit pas être réordonnée sous les pieds de l'appelant.
	if entree[0].ID != "sansDate" {
		t.Error("la tranche d'entrée a été mutée")
	}
}
func ids(rs []Result) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}
