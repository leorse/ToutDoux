package tasktree

import (
	"testing"
	"time"

	"toutdoux/domain"
)

func ptr(s string) *string { return &s }

func mk(id string, parent *string, order int, imp domain.Importance, completed, cancelled bool) domain.Task {
	return domain.Task{
		ID: id, ProjectID: "p1", ParentID: parent, Name: id,
		Importance: imp, OrderIndex: order,
		Completed: completed, Cancelled: cancelled,
	}
}

func active(id string, parent *string, order int) domain.Task {
	return mk(id, parent, order, domain.ImportanceNormale, false, false)
}

// L'ordre des frères suit order_index, pas l'ordre d'insertion (§3.2).
func TestChildrenMap_SortsByOrderIndex(t *testing.T) {
	tasks := []domain.Task{
		active("c", nil, 2),
		active("a", nil, 0),
		active("b", nil, 1),
	}
	roots := ChildrenMap(tasks)[RootKey]
	want := []string{"a", "b", "c"}
	for i, w := range want {
		if roots[i].ID != w {
			t.Errorf("position %d = %s, attendu %s", i, roots[i].ID, w)
		}
	}
}

func TestDescendantIDs(t *testing.T) {
	tasks := []domain.Task{
		active("a", nil, 0),
		active("b", ptr("a"), 0),
		active("c", ptr("b"), 0),
		active("autre", nil, 1),
	}
	got := DescendantIDs(tasks, "a")
	if !got["b"] || !got["c"] {
		t.Errorf("descendants = %v, b et c attendus", got)
	}
	if got["a"] {
		t.Error("la racine ne fait pas partie de ses propres descendants")
	}
	if got["autre"] {
		t.Error("une tâche d'une autre branche ne doit pas apparaître")
	}
	if len(DescendantIDs(tasks, "c")) != 0 {
		t.Error("une feuille n'a pas de descendant")
	}
}

// Un cycle en base ne doit pas faire boucler le parcours indéfiniment.
func TestDescendantIDs_SurvivesCycle(t *testing.T) {
	tasks := []domain.Task{
		{ID: "a", ProjectID: "p1", ParentID: ptr("b")},
		{ID: "b", ProjectID: "p1", ParentID: ptr("a")},
	}
	done := make(chan bool, 1)
	go func() {
		DescendantIDs(tasks, "a")
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("boucle infinie sur un cycle")
	}
}

// Règle centrale du filtrage (§2.4) : un parent non filtrant reste visible si
// un descendant, lui, correspond — sinon le chemin vers le résultat disparaît.
func TestComputeVisibility_KeepsAncestorsOfMatchingTasks(t *testing.T) {
	tasks := []domain.Task{
		mk("parent", nil, 0, domain.ImportanceNormale, true, false), // terminée
		mk("enfant", ptr("parent"), 0, domain.ImportanceNormale, false, false),
	}
	f := DefaultFilters() // actives uniquement

	visible := ComputeVisibility(tasks, f)
	if !visible["enfant"] {
		t.Error("l'enfant actif doit être visible")
	}
	if !visible["parent"] {
		t.Error("le parent terminé doit rester visible : il porte un enfant qui matche")
	}
}

func TestComputeVisibility_HidesFullyNonMatchingBranch(t *testing.T) {
	tasks := []domain.Task{
		mk("parent", nil, 0, domain.ImportanceNormale, true, false),
		mk("enfant", ptr("parent"), 0, domain.ImportanceNormale, true, false),
	}
	visible := ComputeVisibility(tasks, DefaultFilters())
	if len(visible) != 0 {
		t.Errorf("branche entièrement terminée : %v visible, aucune attendue", visible)
	}
}

func TestMatches(t *testing.T) {
	f := DefaultFilters()
	if !Matches(active("a", nil, 0), f) {
		t.Error("une tâche active de priorité normale doit passer les filtres par défaut")
	}
	if Matches(mk("b", nil, 0, domain.ImportanceNormale, true, false), f) {
		t.Error("une tâche terminée ne doit pas passer le filtre actif")
	}

	f.DueOnly = true
	if Matches(active("c", nil, 0), f) {
		t.Error("sans échéance, une tâche ne doit pas passer le filtre \"échéance uniquement\"")
	}
	withDue := active("d", nil, 0)
	now := time.Now()
	withDue.DueDate = &now
	if !Matches(withDue, f) {
		t.Error("avec échéance, la tâche doit passer")
	}
}

// Le filtre Statut est multi-sélection en v2 : Actives + Annulées ensemble (§2.4).
func TestMatches_MultiSelectStatus(t *testing.T) {
	f := DefaultFilters()
	f.Status[domain.StatusCancelled] = true

	if !Matches(mk("a", nil, 0, domain.ImportanceNormale, false, true), f) {
		t.Error("une tâche annulée doit passer quand Annulée est sélectionné")
	}
	if Matches(mk("b", nil, 0, domain.ImportanceNormale, true, false), f) {
		t.Error("une tâche terminée ne doit pas passer, Complétée n'est pas sélectionné")
	}
}

// §2.2 : le critère porte sur la sous-arborescence, pas sur la tâche seule.
func TestDefaultExpanded_ParentWithActiveChild(t *testing.T) {
	tasks := []domain.Task{
		mk("fini", nil, 0, domain.ImportanceNormale, true, false),
		active("encoreActif", ptr("fini"), 0),
		mk("toutFini", nil, 1, domain.ImportanceNormale, true, false),
		mk("aussiFini", ptr("toutFini"), 0, domain.ImportanceNormale, true, false),
	}

	expanded := DefaultExpanded(tasks, "p1")
	if !expanded["fini"] {
		t.Error("un parent terminé portant un enfant actif doit être déplié")
	}
	if expanded["toutFini"] {
		t.Error("une branche entièrement terminée doit démarrer repliée")
	}
}

func TestDefaultExpanded_ScopedToProject(t *testing.T) {
	other := domain.Task{ID: "ailleurs", ProjectID: "p2", Name: "ailleurs"}
	tasks := []domain.Task{active("ici", nil, 0), other}
	expanded := DefaultExpanded(tasks, "p1")
	if expanded["ailleurs"] {
		t.Error("une tâche d'un autre projet ne doit pas apparaître")
	}
}

func TestAncestorIDs(t *testing.T) {
	tasks := []domain.Task{
		active("a", nil, 0),
		active("b", ptr("a"), 0),
		active("c", ptr("b"), 0),
	}
	got := AncestorIDs(tasks, "c")
	if len(got) != 2 || got[0] != "b" || got[1] != "a" {
		t.Errorf("ancêtres = %v, attendu [b a] (du plus proche à la racine)", got)
	}
	if len(AncestorIDs(tasks, "a")) != 0 {
		t.Error("une racine n'a pas d'ancêtre")
	}
}

func TestNextOrderIndex(t *testing.T) {
	tasks := []domain.Task{
		active("a", nil, 0),
		active("b", nil, 4),
		active("enfant", ptr("a"), 7),
	}
	if got := NextOrderIndex(tasks, "p1", nil); got != 5 {
		t.Errorf("racine : %d, attendu 5", got)
	}
	if got := NextOrderIndex(tasks, "p1", ptr("a")); got != 8 {
		t.Errorf("sous a : %d, attendu 8", got)
	}
	if got := NextOrderIndex(tasks, "p1", ptr("vide")); got != 0 {
		t.Errorf("fratrie vide : %d, attendu 0", got)
	}
	// Les fratries d'un autre projet ne doivent pas décaler le compteur.
	if got := NextOrderIndex(tasks, "p2", nil); got != 0 {
		t.Errorf("autre projet : %d, attendu 0", got)
	}
}

func TestMove(t *testing.T) {
	base := func() []domain.Task {
		return []domain.Task{
			active("a", nil, 0),
			active("b", nil, 1),
			active("c", nil, 2),
			active("d", nil, 3),
		}
	}
	order := func(tasks []domain.Task, changed []domain.Task) []string {
		byID := map[string]int{}
		for _, x := range tasks {
			byID[x.ID] = x.OrderIndex
		}
		for _, x := range changed {
			byID[x.ID] = x.OrderIndex
		}
		out := make([]string, len(byID))
		for id, idx := range byID {
			out[idx] = id
		}
		return out
	}

	cases := []struct {
		name string
		id   string
		dir  Direction
		want []string
	}{
		{"monter", "c", DirectionUp, []string{"a", "c", "b", "d"}},
		{"descendre", "b", DirectionDown, []string{"a", "c", "b", "d"}},
		{"envoyer au début", "d", DirectionTop, []string{"d", "a", "b", "c"}},
		{"envoyer à la fin", "a", DirectionBottom, []string{"b", "c", "d", "a"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tasks := base()
			got := order(tasks, Move(tasks, c.id, c.dir))
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("ordre = %v, attendu %v", got, c.want)
				}
			}
		})
	}
}

// Un mouvement sans effet ne doit rien renvoyer : écrire des lignes identiques
// en base serait du bruit pur.
func TestMove_NoOpAtBoundaries(t *testing.T) {
	tasks := []domain.Task{active("a", nil, 0), active("b", nil, 1)}
	if changed := Move(tasks, "a", DirectionUp); len(changed) != 0 {
		t.Errorf("%d tâches modifiées, 0 attendue (déjà en tête)", len(changed))
	}
	if changed := Move(tasks, "b", DirectionBottom); len(changed) != 0 {
		t.Errorf("%d tâches modifiées, 0 attendue (déjà en fin)", len(changed))
	}
	if changed := Move(tasks, "fantome", DirectionUp); changed != nil {
		t.Errorf("une tâche inconnue doit rendre nil, obtenu %v", changed)
	}
}

// Le déplacement reste dans la fratrie : ni le parent ni le projet ne bougent.
func TestMove_StaysWithinSiblings(t *testing.T) {
	tasks := []domain.Task{
		active("racine1", nil, 0),
		active("racine2", nil, 1),
		active("enfant", ptr("racine1"), 0),
	}
	changed := Move(tasks, "racine2", DirectionUp)
	for _, c := range changed {
		if c.ID == "enfant" {
			t.Error("un enfant d'une autre fratrie a été renuméroté")
		}
	}
}

// Deux projets ont chacun leurs racines : déplacer dans l'un ne doit pas
// renuméroter celles de l'autre.
func TestMove_ScopedToProject(t *testing.T) {
	autre := active("autre", nil, 0)
	autre.ProjectID = "p2"
	tasks := []domain.Task{active("a", nil, 0), active("b", nil, 1), autre}

	for _, c := range Move(tasks, "b", DirectionUp) {
		if c.ID == "autre" {
			t.Error("une racine d'un autre projet a été renumérotée")
		}
	}
}
