package cascade

import (
	"errors"
	"testing"

	"toutdoux/domain"
)

// Les cinq cas ci-dessous sont ceux que la spec désigne comme les corrections
// de fond de la v2 (§2.2). Ils sont écrits avant toute UI, précisément parce
// que le prototype n'isolait pas cette règle et ne pouvait donc pas la tester.

func ptr(s string) *string { return &s }

// tree construit une chaîne racine → enfant → petit-enfant, chacun avec le
// statut demandé, plus les tâches supplémentaires passées en argument.
func task(id string, parent *string, completed, cancelled bool) domain.Task {
	return domain.Task{
		ID:         id,
		ProjectID:  "p1",
		ParentID:   parent,
		Name:       id,
		Importance: domain.ImportanceNormale,
		Completed:  completed,
		Cancelled:  cancelled,
	}
}

func find(tasks []domain.Task, id string) (domain.Task, bool) {
	for _, t := range tasks {
		if t.ID == id {
			return t, true
		}
	}
	return domain.Task{}, false
}

func mustActive(t *testing.T, tasks []domain.Task, id string) {
	t.Helper()
	got, ok := find(tasks, id)
	if !ok {
		t.Fatalf("%s absent des tâches modifiées, il devait être réactivé", id)
	}
	if !got.Active() {
		t.Errorf("%s : completed=%v cancelled=%v, attendu actif", id, got.Completed, got.Cancelled)
	}
}

func mustAbsent(t *testing.T, tasks []domain.Task, id string) {
	t.Helper()
	if _, ok := find(tasks, id); ok {
		t.Errorf("%s ne devait pas être modifié", id)
	}
}

// Cas 1 — ajout d'une sous-tâche sous un parent terminé.
// La remontée doit s'arrêter au premier ancêtre déjà actif, et pas aller au-delà.
func TestReactivateAncestors_StopsAtFirstActiveAncestor(t *testing.T) {
	// racine (active) → a (terminée) → b (terminée) → nouvelle sous-tâche
	tasks := []domain.Task{
		task("racine", nil, false, false),
		task("a", ptr("racine"), true, false),
		task("b", ptr("a"), true, false),
	}

	changed := ReactivateAncestors(tasks, "b")

	mustActive(t, changed, "b")
	mustActive(t, changed, "a")
	// racine était déjà active : la remonter serait inutile et la marquerait
	// modifiée à tort.
	mustAbsent(t, changed, "racine")
	if len(changed) != 2 {
		t.Errorf("%d tâches modifiées, attendu 2", len(changed))
	}
}

func TestReactivateAncestors_NoopOnActiveParent(t *testing.T) {
	tasks := []domain.Task{task("racine", nil, false, false)}
	if changed := ReactivateAncestors(tasks, "racine"); len(changed) != 0 {
		t.Errorf("%d tâches modifiées, attendu 0 sur un parent déjà actif", len(changed))
	}
}

// La chaîne réactivée peut mêler terminé et annulé (§2.2).
func TestReactivateAncestors_ClearsBothFlags(t *testing.T) {
	tasks := []domain.Task{
		task("racine", nil, false, false),
		task("a", ptr("racine"), false, true), // annulée
		task("b", ptr("a"), true, false),      // terminée
	}
	changed := ReactivateAncestors(tasks, "b")
	mustActive(t, changed, "b")
	mustActive(t, changed, "a")
}

// Cas 2 — drag & drop d'une tâche ACTIVE sous un parent annulé : même cascade.
func TestReparent_ActiveTaskReactivatesCancelledAncestors(t *testing.T) {
	tasks := []domain.Task{
		task("racine", nil, false, false),
		task("cible", ptr("racine"), false, true), // annulée
		task("libre", nil, false, false),          // active, à déplacer
	}

	changed, err := Reparent(tasks, "libre", ptr("cible"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	mustActive(t, changed, "cible")

	moved, _ := find(changed, "libre")
	if moved.ParentID == nil || *moved.ParentID != "cible" {
		t.Errorf("parent = %v, attendu \"cible\"", moved.ParentID)
	}
}

// Cas 3 — drag & drop d'une tâche DÉJÀ TERMINÉE : aucune cascade.
// C'est le piège classique : le parent terminé reste cohérent, le réactiver
// serait une régression.
func TestReparent_CompletedTaskTriggersNoCascade(t *testing.T) {
	tasks := []domain.Task{
		task("racine", nil, false, false),
		task("cible", ptr("racine"), true, false), // terminée
		task("finie", nil, true, false),           // terminée, à déplacer
	}

	changed, err := Reparent(tasks, "finie", ptr("cible"))
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	mustAbsent(t, changed, "cible")
	if len(changed) != 1 {
		t.Errorf("%d tâches modifiées, attendu 1 (seulement la tâche déplacée)", len(changed))
	}
}

// Cas 4 — décocher un enfant fait perdre au parent completed ET cancelled.
// La v1 ne traitait que completed, laissant des parents annulés au-dessus
// d'enfants actifs.
func TestSetCompleted_UncheckClearsBothOnAncestors(t *testing.T) {
	tasks := []domain.Task{
		task("parent", nil, true, true), // terminée ET annulée
		task("enfant", ptr("parent"), true, false),
	}

	changed, err := SetCompleted(tasks, "enfant", false)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	parent, ok := find(changed, "parent")
	if !ok {
		t.Fatal("le parent devait être modifié")
	}
	if parent.Completed {
		t.Error("le parent doit perdre completed")
	}
	if parent.Cancelled {
		t.Error("le parent doit perdre cancelled (§2.2, correction v2)")
	}
}

// Cas 5 — anti-cycle : erreur explicite, jamais un échec silencieux.
func TestCheckReparent_RejectsCycles(t *testing.T) {
	tasks := []domain.Task{
		task("a", nil, false, false),
		task("b", ptr("a"), false, false),
		task("c", ptr("b"), false, false),
	}

	t.Run("sous son propre descendant direct", func(t *testing.T) {
		if err := CheckReparent(tasks, "a", ptr("b")); !errors.Is(err, domain.ErrCycle) {
			t.Errorf("erreur = %v, attendu ErrCycle", err)
		}
	})
	t.Run("sous un descendant indirect", func(t *testing.T) {
		if err := CheckReparent(tasks, "a", ptr("c")); !errors.Is(err, domain.ErrCycle) {
			t.Errorf("erreur = %v, attendu ErrCycle", err)
		}
	})
	t.Run("sous elle-même", func(t *testing.T) {
		if err := CheckReparent(tasks, "a", ptr("a")); !errors.Is(err, domain.ErrCycle) {
			t.Errorf("erreur = %v, attendu ErrCycle", err)
		}
	})
	t.Run("vers la racine est permis", func(t *testing.T) {
		if err := CheckReparent(tasks, "c", nil); err != nil {
			t.Errorf("erreur = %v, attendu nil", err)
		}
	})
	t.Run("sous un ancêtre est permis", func(t *testing.T) {
		if err := CheckReparent(tasks, "c", ptr("a")); err != nil {
			t.Errorf("erreur = %v, attendu nil", err)
		}
	})

	// Reparent doit refuser aussi, pas seulement CheckReparent.
	if _, err := Reparent(tasks, "a", ptr("c")); !errors.Is(err, domain.ErrCycle) {
		t.Errorf("Reparent : erreur = %v, attendu ErrCycle", err)
	}
}

// Auto-complétion du parent quand tous ses enfants directs sont terminés (§2.2),
// en cascade jusqu'à la racine.
func TestSetCompleted_ParentCompletesWhenAllChildrenDone(t *testing.T) {
	tasks := []domain.Task{
		task("racine", nil, false, false),
		task("parent", ptr("racine"), false, false),
		task("e1", ptr("parent"), true, false),
		task("e2", ptr("parent"), false, false),
	}

	changed, err := SetCompleted(tasks, "e2", true)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}

	parent, ok := find(changed, "parent")
	if !ok || !parent.Completed {
		t.Error("le parent doit passer terminé quand tous ses enfants le sont")
	}
	racine, ok := find(changed, "racine")
	if !ok || !racine.Completed {
		t.Error("la cascade doit remonter jusqu'à la racine")
	}
}

// Cocher une tâche coche toute sa descendance et lève leur annulation.
func TestSetCompleted_PropagatesDownAndClearsCancelled(t *testing.T) {
	tasks := []domain.Task{
		task("parent", nil, false, false),
		task("enfant", ptr("parent"), false, true), // annulé
		task("petit", ptr("enfant"), false, false),
	}

	changed, err := SetCompleted(tasks, "parent", true)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	for _, id := range []string{"parent", "enfant", "petit"} {
		got, ok := find(changed, id)
		if !ok {
			t.Fatalf("%s devait être modifié", id)
		}
		if !got.Completed {
			t.Errorf("%s : completed attendu", id)
		}
		if got.Cancelled {
			t.Errorf("%s : cancelled devait être levé", id)
		}
	}
}

// Une tâche sans enfant ne devient pas terminée toute seule : la règle
// "tous les enfants sont terminés" exige au moins un enfant.
func TestSetCompleted_ChildlessParentNotAutoCompleted(t *testing.T) {
	tasks := []domain.Task{
		task("parent", nil, false, false),
		task("enfant", ptr("parent"), false, false),
	}
	changed, err := SetCompleted(tasks, "enfant", false)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if p, ok := find(changed, "parent"); ok && p.Completed {
		t.Error("le parent ne doit pas être terminé")
	}
}

func TestSetCancelled_ClearsCompleted(t *testing.T) {
	tasks := []domain.Task{task("a", nil, true, false)}
	changed, err := SetCancelled(tasks, "a", true)
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	got, _ := find(changed, "a")
	if !got.Cancelled || got.Completed {
		t.Errorf("completed=%v cancelled=%v, attendu completed=false cancelled=true", got.Completed, got.Cancelled)
	}
}

func TestSetCompleted_UnknownTask(t *testing.T) {
	if _, err := SetCompleted(nil, "fantome", true); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
}

// Les fonctions sont pures : l'entrée ne doit jamais être mutée.
func TestSetCompleted_DoesNotMutateInput(t *testing.T) {
	tasks := []domain.Task{
		task("parent", nil, false, false),
		task("enfant", ptr("parent"), false, false),
	}
	if _, err := SetCompleted(tasks, "enfant", true); err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if tasks[1].Completed {
		t.Error("la tranche d'entrée a été mutée")
	}
	if tasks[0].Completed {
		t.Error("la tranche d'entrée a été mutée (parent)")
	}
}
