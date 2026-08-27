package main

import (
	"errors"
	"testing"
	"time"

	"toutdoux/domain"
)

/* ---------------- Réunions (§2.7) ---------------- */

func TestMeetings_CRUDAndInstances(t *testing.T) {
	app := newTestApp(t)

	m, err := app.CreateMeeting(domain.DiversProjectID, "Comité de pilotage")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.RenameMeeting(m.ID, "Comité hebdomadaire"); err != nil {
		t.Fatal(err)
	}
	liste, err := app.GetMeetings(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(liste) != 1 || liste[0].Title != "Comité hebdomadaire" {
		t.Fatalf("réunions = %+v", liste)
	}

	i1, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateInstanceNotes(i1.ID, "<p>Échéance repoussée</p>"); err != nil {
		t.Fatal(err)
	}
	relue, err := app.GetInstances(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(relue) != 1 || relue[0].Notes != "<p>Échéance repoussée</p>" {
		t.Fatalf("instances = %+v", relue)
	}
}

// Les instances sont triées par date décroissante, la plus récente d'abord (§2.7).
func TestMeetings_InstancesSortedNewestFirst(t *testing.T) {
	app := newTestApp(t)
	horloge := app.clock.(interface{ Advance(time.Duration) })

	m, err := app.CreateMeeting(domain.DiversProjectID, "Point")
	if err != nil {
		t.Fatal(err)
	}
	ancienne, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	horloge.Advance(48 * time.Hour)
	recente, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}

	liste, err := app.GetInstances(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liste[0].ID != recente.ID || liste[1].ID != ancienne.ID {
		t.Errorf("ordre = %s puis %s, la plus récente devait être en tête", liste[0].ID, liste[1].ID)
	}
}

// Supprimer une réunion emporte ses instances (§2.7).
func TestMeetings_DeleteCascadesInstances(t *testing.T) {
	app := newTestApp(t)
	m, err := app.CreateMeeting(domain.DiversProjectID, "Point")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.AddMeetingInstance(m.ID); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteMeeting(m.ID); err != nil {
		t.Fatal(err)
	}
	liste, err := app.GetInstances(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(liste) != 0 {
		t.Errorf("%d instances restantes, 0 attendue", len(liste))
	}
	meetings, err := app.GetMeetings(domain.DiversProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 0 {
		t.Errorf("%d réunions restantes, 0 attendue", len(meetings))
	}
}

// Le timestamp date la tenue de la réunion, pas la dernière frappe (§2.7).
func TestMeetings_NotesUpdateDoesNotMoveTimestamp(t *testing.T) {
	app := newTestApp(t)
	horloge := app.clock.(interface{ Advance(time.Duration) })

	m, err := app.CreateMeeting(domain.DiversProjectID, "Point")
	if err != nil {
		t.Fatal(err)
	}
	i, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	horloge.Advance(72 * time.Hour)
	maj, err := app.UpdateInstanceNotes(i.ID, "compte rendu écrit plus tard")
	if err != nil {
		t.Fatal(err)
	}
	if !maj.Timestamp.Equal(i.Timestamp) {
		t.Errorf("timestamp = %v, attendu inchangé %v", maj.Timestamp, i.Timestamp)
	}
}

/* ---------------- Priorités (§2.8) ---------------- */

func TestGetPriorityTasks(t *testing.T) {
	app := newTestApp(t)

	critique, err := app.CreateTask(domain.DiversProjectID, "", "Critique active")
	if err != nil {
		t.Fatal(err)
	}
	imp := "Critique"
	if _, err := app.UpdateTask(critique.Task.ID, TaskPatch{Importance: &imp}); err != nil {
		t.Fatal(err)
	}
	avecEcheance, err := app.CreateTask(domain.DiversProjectID, "", "Avec échéance")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetTaskDueDateQuick(avecEcheance.Task.ID, "1h"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTask(domain.DiversProjectID, "", "Normale sans rien"); err != nil {
		t.Fatal(err)
	}

	p, err := app.GetPriorityTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.CriticalOrHigh) != 1 || p.CriticalOrHigh[0].Name != "Critique active" {
		t.Errorf("CriticalOrHigh = %+v", p.CriticalOrHigh)
	}
	if len(p.DueSoon) != 1 || p.DueSoon[0].Name != "Avec échéance" {
		t.Errorf("DueSoon = %+v", p.DueSoon)
	}
}

/* ---------------- Recherche (§2.9) ---------------- */

func TestSearchGlobal_FindsAcrossEntities(t *testing.T) {
	app := newTestApp(t)

	if _, err := app.CreateTask(domain.DiversProjectID, "", "Préparer la réunion"); err != nil {
		t.Fatal(err)
	}
	note, err := app.CreateNote(domain.DiversProjectID, "Compte rendu")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(note.ID, "Compte rendu", "<p>La réunion est repoussée</p>"); err != nil {
		t.Fatal(err)
	}
	m, err := app.CreateMeeting(domain.DiversProjectID, "Réunion hebdomadaire")
	if err != nil {
		t.Fatal(err)
	}
	inst, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateInstanceNotes(inst.ID, "<p>ordre du jour de la réunion</p>"); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 4 {
		t.Fatalf("%d résultats, 4 attendus (tâche, note, réunion, instance)", len(res))
	}
}

// La recherche ne se déclenche qu'au-delà de 2 caractères (§2.9).
func TestSearchGlobal_MinQueryLength(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "réunion"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"", "r", "ré"} {
		res, err := app.SearchGlobal(q)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 0 {
			t.Errorf("%q : %d résultats, aucun attendu", q, len(res))
		}
	}
}

// La ponctuation ne doit pas produire d'erreur de syntaxe FTS5 : c'est le piège
// principal de MATCH, qui est un langage à part entière.
func TestSearchGlobal_SurvivesPunctuation(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "Revue mi-parcours"); err != nil {
		t.Fatal(err)
	}
	requetes := []string{
		"mi-parcours",
		"aujourd" + "'" + "hui",
		`"guillemets"`,
		"point.",
		"AND OR NOT",
		"(parenthese",
		"étoile*",
	}
	for _, q := range requetes {
		if _, err := app.SearchGlobal(q); err != nil {
			t.Errorf("requête %q : %v", q, err)
		}
	}
}

func TestSearchGlobal_AccentAndCaseInsensitive(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "Échéance de la Réunion"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"réunion", "RÉUNION", "échéance"} {
		res, err := app.SearchGlobal(q)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) == 0 {
			t.Errorf("%q : aucun résultat", q)
		}
	}
}

// L'extrait retire le balisage : l'utilisateur ne doit pas lire des <p>.
func TestSearchGlobal_SnippetStripsHtml(t *testing.T) {
	app := newTestApp(t)
	note, err := app.CreateNote(domain.DiversProjectID, "Note")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(note.ID, "Note", "<p>avant <strong>réunion</strong> après</p>"); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 || res[0].Snippet == nil {
		t.Fatal("extrait absent")
	}
	if texte := res[0].Snippet.Text(); texte != "avant réunion après" {
		t.Errorf("extrait = %q, balisage mal retiré", texte)
	}

	// Le fragment surligné doit être l'occurrence, pour que le frontend le
	// rende en <mark> jaune (§2.9).
	var surligne string
	for _, p := range res[0].Snippet.Parts {
		if p.Match {
			surligne = p.Text
		}
	}
	if surligne != "réunion" {
		t.Errorf("fragment surligné = %q", surligne)
	}
}

// Une entité supprimée ne doit plus remonter : un résultat qu'on ne peut pas
// ouvrir est pire que pas de résultat.
func TestSearchGlobal_RemovesDeletedEntities(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "réunion à supprimer")
	if err != nil {
		t.Fatal(err)
	}
	avant, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(avant) != 1 {
		t.Fatalf("%d résultats avant suppression, 1 attendu", len(avant))
	}

	if err := app.DeleteTask(res.Task.ID); err != nil {
		t.Fatal(err)
	}
	apres, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(apres) != 0 {
		t.Errorf("%d résultats après suppression, 0 attendu", len(apres))
	}
}

// Renommer une tâche doit réindexer : l'ancien terme ne doit plus la trouver.
func TestSearchGlobal_ReindexesOnRename(t *testing.T) {
	app := newTestApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "ancienne appellation")
	if err != nil {
		t.Fatal(err)
	}
	nouveau := "nouvelle désignation"
	if _, err := app.UpdateTask(res.Task.ID, TaskPatch{Name: &nouveau}); err != nil {
		t.Fatal(err)
	}

	ancien, err := app.SearchGlobal("ancienne")
	if err != nil {
		t.Fatal(err)
	}
	if len(ancien) != 0 {
		t.Errorf("%d résultats sur l'ancien nom, 0 attendu", len(ancien))
	}
	nouv, err := app.SearchGlobal("désignation")
	if err != nil {
		t.Fatal(err)
	}
	if len(nouv) != 1 {
		t.Errorf("%d résultats sur le nouveau nom, 1 attendu", len(nouv))
	}
}

// Supprimer un projet purge l'index de tout son contenu (§2.1).
func TestSearchGlobal_ProjectDeletionPurgesIndex(t *testing.T) {
	app := newTestApp(t)
	p, err := app.CreateProject("Projet éphémère")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateTask(p.ID, "", "réunion du projet éphémère"); err != nil {
		t.Fatal(err)
	}
	if err := app.DeleteProject(p.ID); err != nil {
		t.Fatal(err)
	}
	got, err := app.SearchGlobal("éphémère")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("%d résultats après suppression du projet, 0 attendu", len(got))
	}
}

// La reconstruction au démarrage rattrape un index vide ou en retard.
func TestReindexAll_RebuildsFromData(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "réunion indexée"); err != nil {
		t.Fatal(err)
	}
	if err := app.index.DeleteByProject(domain.DiversProjectID); err != nil {
		t.Fatal(err)
	}
	vide, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(vide) != 0 {
		t.Fatal("préalable non tenu : l'index devait être vidé")
	}

	if err := app.reindexAll(); err != nil {
		t.Fatal(err)
	}
	got, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("%d résultats après reconstruction, 1 attendu", len(got))
	}
}

// Le résultat d'une tâche porte l'entité : la ligne affiche l'état coché et une
// pastille d'importance, pas un simple ✓ (§2.9).
func TestSearchGlobal_ResultCarriesEntityAndProjectName(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.CreateTask(domain.DiversProjectID, "", "réunion importante"); err != nil {
		t.Fatal(err)
	}
	res, err := app.SearchGlobal("réunion")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("%d résultats, 1 attendu", len(res))
	}
	if res[0].Task == nil {
		t.Error("la tâche doit être jointe au résultat")
	}
	if res[0].ProjectName != domain.DiversProjectName {
		t.Errorf("nom de projet = %q, attendu %q", res[0].ProjectName, domain.DiversProjectName)
	}
}

func TestMeetings_UnknownIDs(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.AddMeetingInstance("fantome"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("erreur = %v, attendu ErrNotFound", err)
	}
	if _, err := app.CreateMeeting(domain.DiversProjectID, "   "); !errors.Is(err, domain.ErrEmptyName) {
		t.Errorf("erreur = %v, attendu ErrEmptyName", err)
	}
}

// Les noms de projets sont cherchables au même titre que le reste (§2.9).
func TestSearchGlobal_FindsProjects(t *testing.T) {
	app := newTestApp(t)

	p, err := app.CreateProject("Refonte du portail client")
	if err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchGlobal("portail")
	if err != nil {
		t.Fatal(err)
	}
	var trouve *SearchResult
	for i := range res {
		if res[i].ID == p.ID {
			trouve = &res[i]
		}
	}
	if trouve == nil {
		t.Fatalf("le projet n'est pas trouvé : %+v", res)
	}
	if trouve.Type != "project" {
		t.Fatalf("type = %q, attendu project", trouve.Type)
	}
	// L'entité voyage avec le résultat, comme pour les autres types : la ligne
	// doit pouvoir ouvrir le projet.
	if trouve.Project == nil || trouve.Project.Name != "Refonte du portail client" {
		t.Fatalf("projet absent du résultat : %+v", trouve)
	}
}

func TestSearchGlobal_ProjectRenameReindexes(t *testing.T) {
	app := newTestApp(t)

	p, err := app.CreateProject("Ancien intitulé")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.RenameProject(p.ID, "Migration SIRH"); err != nil {
		t.Fatal(err)
	}

	apres, err := app.SearchGlobal("SIRH")
	if err != nil {
		t.Fatal(err)
	}
	if len(apres) == 0 {
		t.Fatal("le nouveau nom ne remonte pas")
	}
	avant, err := app.SearchGlobal("Ancien")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range avant {
		if r.ID == p.ID {
			t.Fatal("l'ancien nom remonte encore : l'index n'a pas été mis à jour")
		}
	}
}

// Le projet verrouillé est indexé par la reconstruction du démarrage, pas par
// CreateProject : il est créé par la couche de persistance (§2.1).
func TestSearchGlobal_FindsDiversProject(t *testing.T) {
	app := newTestApp(t)
	if err := app.reindexAll(); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchGlobal("Transverse")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.ID == domain.DiversProjectID {
			return
		}
	}
	t.Fatalf("le projet verrouillé n'est pas indexé : %+v", res)
}
