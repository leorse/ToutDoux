package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"toutdoux/adapters"
	"toutdoux/adapters/model"
	"toutdoux/domain"
)

// newSemanticApp rend une application dont le fournisseur de vecteurs est
// observable : les tests comptent les appels pour vérifier ce qui *ne* doit pas
// être recalculé.
func newSemanticApp(t *testing.T) (*App, *adapters.FakeEmbeddingProvider) {
	t.Helper()
	app := newTestApp(t)
	fake := adapters.NewFakeEmbeddingProvider()
	app.embedder = fake
	app.modelDir = t.TempDir()
	return app, fake
}

/* ---------------- Opt-in strict (§2.12) ---------------- */

func TestSemantic_RienNestIndexeSansDemande(t *testing.T) {
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Compte rendu")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "Compte rendu", "<p>Le client est en colère</p>"); err != nil {
		t.Fatal(err)
	}

	// Créer et écrire une note ne doit déclencher aucune vectorisation : c'est
	// tout le sens de l'opt-in, une vectorisation coûte des centaines de ms.
	if fake.CallCount() != 0 {
		t.Fatalf("%d vectorisations sans demande explicite", fake.CallCount())
	}
	dans, err := app.IsInSemanticIndex(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dans {
		t.Fatal("la note est dans l'index sémantique sans avoir été ajoutée")
	}
}

func TestSemantic_AjoutRetraitEtEtatDuBouton(t *testing.T) {
	app, _ := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Réunion client")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}
	if dans, _ := app.IsInSemanticIndex(n.ID); !dans {
		t.Fatal("la note devrait être indexée après ajout")
	}

	if err := app.RemoveFromSemanticIndex(n.ID); err != nil {
		t.Fatal(err)
	}
	if dans, _ := app.IsInSemanticIndex(n.ID); dans {
		t.Fatal("la note devrait avoir quitté l'index après retrait")
	}
}

func TestSemantic_RefusDIndexerUnTexteVide(t *testing.T) {
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Brouillon")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "", ""); err != nil {
		t.Fatal(err)
	}

	// Un vecteur calculé sur du vide n'a aucun sens et remonterait au hasard
	// dans les résultats : mieux vaut refuser franchement.
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); !errors.Is(err, ErrNothingToEmbed) {
		t.Fatalf("erreur = %v, attendu ErrNothingToEmbed", err)
	}
	if fake.CallCount() != 0 {
		t.Fatalf("%d vectorisations sur un texte vide", fake.CallCount())
	}
}

func TestSemantic_TypeDEntiteInconnuRefuse(t *testing.T) {
	app, _ := newSemanticApp(t)
	res, err := app.CreateTask(domain.DiversProjectID, "", "Tâche")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex("chimère", res.Task.ID); err == nil {
		t.Fatal("un type d'entité inconnu devrait être refusé")
	}
}

/* ---------------- Modèle absent (§2.12) ---------------- */

func TestSemantic_SansModeleLAjoutEstRefuseExplicitement(t *testing.T) {
	app, fake := newSemanticApp(t)
	fake.Unavailable = true

	n, err := app.CreateNote(domain.DiversProjectID, "Note")
	if err != nil {
		t.Fatal(err)
	}
	err = app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID)
	if !errors.Is(err, domain.ErrModelUnavailable) {
		t.Fatalf("erreur = %v, attendu ErrModelUnavailable", err)
	}

	// Et la recherche aussi : l'interface doit pouvoir ouvrir la fenêtre du
	// §2.12 plutôt que d'afficher « aucun résultat ».
	if _, err := app.SearchSemantic("client mécontent"); !errors.Is(err, domain.ErrModelUnavailable) {
		t.Fatalf("erreur de recherche = %v, attendu ErrModelUnavailable", err)
	}
}

func TestSemantic_StatutDuModele(t *testing.T) {
	app, _ := newSemanticApp(t)

	s := app.GetModelStatus()
	if s.Available {
		t.Fatal("modèle annoncé disponible dans un dossier vide")
	}
	if len(s.MissingFiles) != 3 || s.DownloadURL == "" || s.Directory == "" {
		t.Fatalf("statut incomplet : %+v", s)
	}

	// Le bouton « Réessayer » du §2.13 n'a de sens que si la détection relit le
	// disque à chaque appel, sans cache.
	for _, nom := range s.ExpectedFiles {
		if err := os.WriteFile(filepath.Join(app.modelDir, nom), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if !app.GetModelStatus().Available {
		t.Fatal("les fichiers déposés ne sont pas vus : la détection est mise en cache")
	}
}

func TestSemantic_StatutDistingueFichiersEtMoteur(t *testing.T) {
	app, fake := newSemanticApp(t)
	fake.Unavailable = true
	for _, nom := range []string{model.FichierModele, model.FichierTokenizer, model.FichierSentencePi} {
		if err := os.WriteFile(filepath.Join(app.modelDir, nom), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s, err := app.GetSemanticStatus()
	if err != nil {
		t.Fatal(err)
	}
	// Les deux informations doivent être distinctes : sinon l'interface
	// enverrait l'utilisateur retélécharger 120 Mo qu'il a déjà déposés.
	if !s.FilesPresent {
		t.Fatal("les fichiers déposés devraient être signalés présents")
	}
	if s.ModelAvailable {
		t.Fatal("le moteur est indisponible, le statut ne doit pas dire le contraire")
	}
}

/* ---------------- Maintien de l'index (§2.12) ---------------- */

func TestSemantic_ChangementDeTexteRevectorise(t *testing.T) {
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Suivi")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}
	fake.Reset()

	if _, err := app.UpdateNote(n.ID, "Suivi", "<p>Le client réclame un geste commercial</p>"); err != nil {
		t.Fatal(err)
	}
	if fake.CallCount() != 1 {
		t.Fatalf("%d vectorisations après changement de texte, attendu 1", fake.CallCount())
	}
}

func TestSemantic_ReenregistrerLeMemeTexteNeRecalculeRien(t *testing.T) {
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Suivi")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "Suivi", "<p>Contenu stable</p>"); err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}
	fake.Reset()

	// La sauvegarde automatique réenregistre sans arrêt le même contenu : c'est
	// le cas dominant, et il ne doit rien coûter.
	for i := 0; i < 5; i++ {
		if _, err := app.UpdateNote(n.ID, "Suivi", "<p>Contenu stable</p>"); err != nil {
			t.Fatal(err)
		}
	}
	if fake.CallCount() != 0 {
		t.Fatalf("%d vectorisations pour un texte inchangé", fake.CallCount())
	}
}

func TestSemantic_ChangementDeMetadonneeNeRecalculeRien(t *testing.T) {
	app, fake := newSemanticApp(t)

	res, err := app.CreateTask(domain.DiversProjectID, "", "Relancer le client")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeTask), res.Task.ID); err != nil {
		t.Fatal(err)
	}
	fake.Reset()

	// C'est le piège spécifique de la fonctionnalité (§2.12) : cocher une case
	// ou changer une importance ne modifie pas le sens du texte.
	critique := string(domain.ImportanceCritique)
	if _, err := app.UpdateTask(res.Task.ID, TaskPatch{Importance: &critique}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SetTaskDueDateQuick(res.Task.ID, "journee"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ToggleTaskCompleted(res.Task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ToggleTaskCancelled(res.Task.ID); err != nil {
		t.Fatal(err)
	}

	if fake.CallCount() != 0 {
		t.Fatalf("%d vectorisations pour de simples changements de métadonnée", fake.CallCount())
	}
}

func TestSemantic_UneEntiteNonIndexeeNestJamaisVectorisee(t *testing.T) {
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Hors index")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "Hors index", "<p>Un texte tout neuf</p>"); err != nil {
		t.Fatal(err)
	}
	if err := app.RefreshEmbedding(n.ID); err != nil {
		t.Fatal(err)
	}
	if fake.CallCount() != 0 {
		t.Fatalf("%d vectorisations sur une entité hors index", fake.CallCount())
	}
}

func TestSemantic_ModeleRetireApresIndexation(t *testing.T) {
	// Limite connue et assumée du §2.12 : la revectorisation échoue en silence
	// et le vecteur reste celui du texte précédent. Ce que le test verrouille,
	// c'est que l'enregistrement du texte, lui, ne casse pas.
	app, fake := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Note")
	if err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}

	fake.Unavailable = true
	maj, err := app.UpdateNote(n.ID, "Note", "<p>Texte modifié sans modèle</p>")
	if err != nil {
		t.Fatalf("la sauvegarde du texte ne doit pas dépendre du modèle : %v", err)
	}
	if maj.Content != "<p>Texte modifié sans modèle</p>" {
		t.Fatalf("contenu = %q", maj.Content)
	}
	if dans, _ := app.IsInSemanticIndex(n.ID); !dans {
		t.Fatal("l'entité doit rester indexée avec son vecteur périmé")
	}
}

func TestSemantic_VidageDuTexteRetireLeVecteur(t *testing.T) {
	app, _ := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Titre")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "Titre", "<p>Contenu</p>"); err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}

	// Titre et contenu vidés : le vecteur ne correspond plus à rien et ferait
	// remonter la note au hasard.
	if _, err := app.UpdateNote(n.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	if dans, _ := app.IsInSemanticIndex(n.ID); dans {
		t.Fatal("un texte vidé doit sortir de l'index sémantique")
	}
}

/* ---------------- Suppression (§2.1, §2.2) ---------------- */

func TestSemantic_SuppressionsNettoientLIndex(t *testing.T) {
	app, _ := newSemanticApp(t)

	p, err := app.CreateProject("Projet à supprimer")
	if err != nil {
		t.Fatal(err)
	}
	n, err := app.CreateNote(p.ID, "Note")
	if err != nil {
		t.Fatal(err)
	}
	res, err := app.CreateTask(p.ID, "", "Tâche")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct{ typ, id string }{
		{string(domain.SearchTypeNote), n.ID},
		{string(domain.SearchTypeTask), res.Task.ID},
	} {
		if err := app.AddToSemanticIndex(e.typ, e.id); err != nil {
			t.Fatal(err)
		}
	}

	if err := app.DeleteTask(res.Task.ID); err != nil {
		t.Fatal(err)
	}
	if dans, _ := app.IsInSemanticIndex(res.Task.ID); dans {
		t.Fatal("le vecteur d'une tâche supprimée subsiste")
	}

	// La table embeddings porte une clé étrangère sur le projet : sans purge, la
	// suppression échouerait sur une violation de contrainte.
	if err := app.DeleteProject(p.ID); err != nil {
		t.Fatalf("suppression du projet : %v", err)
	}
	statut, err := app.GetSemanticStatus()
	if err != nil {
		t.Fatal(err)
	}
	if statut.IndexedCount != 0 {
		t.Fatalf("%d vecteurs subsistent après suppression du projet", statut.IndexedCount)
	}
}

/* ---------------- Recherche (§2.12) ---------------- */

func TestSemantic_TriParScoreEtNonParDate(t *testing.T) {
	app, _ := newSemanticApp(t)

	// Le Fake est un sac de mots : les textes qui partagent des mots avec la
	// requête sortent devant. C'est suffisant pour vérifier le tri — la vraie
	// proximité de sens relève du modèle et se vérifie à la main (§3.10).
	proche, err := app.CreateNote(domain.DiversProjectID, "client mécontent réclamation")
	if err != nil {
		t.Fatal(err)
	}
	loin, err := app.CreateNote(domain.DiversProjectID, "commande matériel bureau")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{loin.ID, proche.ID} {
		if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), id); err != nil {
			t.Fatal(err)
		}
	}

	res, err := app.SearchSemantic("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("aucun résultat sémantique")
	}
	if res[0].ID != proche.ID {
		t.Fatalf("premier résultat = %s, attendu la note la plus proche", res[0].Title)
	}
	if res[0].Score <= 0 {
		t.Fatalf("score = %v, attendu strictement positif", res[0].Score)
	}
	for i := 1; i < len(res); i++ {
		if res[i].Score > res[i-1].Score {
			t.Fatalf("scores non décroissants : %v", res)
		}
	}
}

func TestSemantic_ExtraitSansSurbrillance(t *testing.T) {
	app, _ := newSemanticApp(t)

	n, err := app.CreateNote(domain.DiversProjectID, "Client")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateNote(n.ID, "Client", "<p>Le client est en colère depuis la livraison</p>"); err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), n.ID); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchSemantic("client colère livraison")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("%d résultats, attendu 1", len(res))
	}
	// Aucun mot exact n'a été mis en correspondance : il n'y a rien à surligner.
	for _, p := range res[0].Snippet.Parts {
		if p.Match {
			t.Fatalf("fragment surligné en mode sémantique : %+v", res[0].Snippet)
		}
	}
	// Le balisage TipTap ne doit pas remonter dans l'extrait.
	if got := res[0].Snippet.Text(); got == "" || contient(got, "<p>") {
		t.Fatalf("extrait = %q", got)
	}
}

func TestSemantic_SeulesLesEntitesAjouteesRemontent(t *testing.T) {
	app, _ := newSemanticApp(t)

	indexee, err := app.CreateNote(domain.DiversProjectID, "client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.CreateNote(domain.DiversProjectID, "client mécontent bis"); err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeNote), indexee.ID); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchSemantic("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].ID != indexee.ID {
		t.Fatalf("résultats = %+v, attendu la seule note ajoutée à l'index", res)
	}
}

func TestSemantic_RequeteTropCourte(t *testing.T) {
	app, _ := newSemanticApp(t)
	res, err := app.SearchSemantic("cl")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("%d résultats sous le seuil de 3 caractères", len(res))
	}
}

func TestSemantic_IndexVideSeDistingueDAucunResultat(t *testing.T) {
	app, _ := newSemanticApp(t)

	statut, err := app.GetSemanticStatus()
	if err != nil {
		t.Fatal(err)
	}
	if statut.IndexedCount != 0 {
		t.Fatalf("index annoncé non vide : %+v", statut)
	}
	// C'est ce compteur, et non la longueur des résultats, qui permet à la vue
	// de dire « index vide » plutôt que « aucun résultat » (§2.12).
	res, err := app.SearchSemantic("client mécontent")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("%d résultats sur un index vide", len(res))
	}
}

func TestSemantic_InstanceDeReunionIndexable(t *testing.T) {
	app, _ := newSemanticApp(t)

	m, err := app.CreateMeeting(domain.DiversProjectID, "Comité client")
	if err != nil {
		t.Fatal(err)
	}
	i, err := app.AddMeetingInstance(m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.UpdateInstanceNotes(i.ID, "<p>Le client réclame un avoir</p>"); err != nil {
		t.Fatal(err)
	}
	if err := app.AddToSemanticIndex(string(domain.SearchTypeMeeting), i.ID); err != nil {
		t.Fatal(err)
	}

	res, err := app.SearchSemantic("client avoir réclame")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].ID != i.ID {
		t.Fatalf("résultats = %+v, attendu l'instance", res)
	}
	// L'instance s'affiche sous le titre de sa réunion, comme en §2.9.
	if res[0].Title != "Comité client" {
		t.Fatalf("titre = %q, attendu celui de la réunion", res[0].Title)
	}
}

func contient(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
