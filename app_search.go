package main

import (
	"regexp"
	"strings"

	"toutdoux/domain"
	"toutdoux/domain/search"
	"toutdoux/domain/stats"
)

/* ---------------- Priorités (§2.8) ---------------- */

// GetPriorityTasks rend les deux sections de la vue Priorités (§2.8).
//
// La même lecture alimentera le menu de la barre système en Phase 4, comme le
// prévoit le §2.10 : il n'y aura pas de second calcul.
func (a *App) GetPriorityTasks() (stats.PriorityTasks, error) {
	tasks, err := a.tasks.ListAll()
	if err != nil {
		return stats.PriorityTasks{}, err
	}
	return stats.Priorities(tasks), nil
}

/* ---------------- Recherche (§2.9) ---------------- */

// SearchResult est une ligne de résultat enrichie de son extrait.
type SearchResult struct {
	search.Result
	// Snippet est l'extrait surligné, calculé côté domaine pour que le frontend
	// n'ait qu'à mapper les fragments marqués sur des <mark> (§2.9).
	Snippet *search.Snippet `json:"snippet"`
	// ProjectName évite au frontend de recroiser la liste des projets pour
	// afficher le tag de chaque ligne.
	ProjectName string `json:"projectName"`

	// Score n'est renseigné qu'en recherche sémantique (§2.12) : c'est la
	// similarité cosinus, entre 0 et 1. En recherche mot-clé il vaut 0, la
	// pertinence FTS5 n'étant pas exposée à l'utilisateur.
	Score float64 `json:"score"`
}

// SearchGlobal cherche dans les tâches, notes et réunions (§2.9).
//
// Toujours globale : le mode « Projet » de la v1 a été retiré, il n'y a pas de
// variante `search_project`.
func (a *App) SearchGlobal(query string) ([]SearchResult, error) {
	if len([]rune(strings.TrimSpace(query))) < search.MinQueryLength {
		return []SearchResult{}, nil
	}

	entries, err := a.index.Search(query)
	if err != nil {
		return nil, err
	}

	projets, err := a.projects.List()
	if err != nil {
		return nil, err
	}
	nomParProjet := make(map[string]string, len(projets))
	for _, p := range projets {
		nomParProjet[p.ID] = p.Name
	}

	results := make([]search.Result, 0, len(entries))
	extraits := make(map[string]*search.Snippet, len(entries))

	for _, e := range entries {
		// L'entité est rattachée au résultat : la ligne d'une tâche affiche son
		// état coché et sa pastille d'importance, pas un simple ✓ (§2.9).
		//
		// Le chargement est partagé avec la recherche sémantique (app_semantic.go) :
		// une seule définition de « à quoi ressemble une ligne de résultat »,
		// donc aucun risque que les deux modes divergent à l'affichage.
		r, contenu, ok := a.chargerResultat(e.Type, e.EntityID)
		if !ok {
			continue // entrée orpheline : l'entité a disparu, on l'ignore
		}

		results = append(results, r)
		// L'extrait porte sur le contenu, qui est du HTML pour les notes et les
		// comptes rendus : le balisage est retiré avant découpe, sinon
		// l'utilisateur lirait des morceaux de <p> autour de son mot-clé.
		extraits[r.ID] = search.Highlight(contenu, query, search.DefaultContextChars)
	}

	ordered := search.Sort(results)
	out := make([]SearchResult, 0, len(ordered))
	for _, r := range ordered {
		out = append(out, SearchResult{
			Result:      r,
			Snippet:     extraits[r.ID],
			ProjectName: nomParProjet[r.ProjectID],
		})
	}
	return out, nil
}

// balises retire le balisage HTML produit par TipTap.
var balises = regexp.MustCompile(`<[^>]*>`)

// texteBrut rend le contenu lisible d'un champ riche.
func texteBrut(html string) string {
	sansBalises := balises.ReplaceAllString(html, " ")
	return strings.Join(strings.Fields(sansBalises), " ")
}

/* ---------------- Tenue de l'index (§3.2) ---------------- */

// indexTask écrit l'entrée d'index d'une tâche.
func (a *App) indexTask(t domain.Task) error {
	return a.index.Put(domain.IndexEntry{
		Type: domain.SearchTypeTask, EntityID: t.ID, ProjectID: t.ProjectID,
		Title: t.Name, Content: t.Description, CreatedAt: &t.CreatedAt,
	})
}

// indexNote écrit l'entrée d'index d'une note.
func (a *App) indexNote(n domain.Note) error {
	return a.index.Put(domain.IndexEntry{
		Type: domain.SearchTypeNote, EntityID: n.ID, ProjectID: n.ProjectID,
		Title: n.Title, Content: texteBrut(n.Content), CreatedAt: &n.UpdatedAt,
	})
}

// indexMeeting écrit l'entrée d'index d'une réunion.
func (a *App) indexMeeting(m domain.Meeting) error {
	return a.index.Put(domain.IndexEntry{
		Type: domain.SearchTypeMeeting, EntityID: m.ID, ProjectID: m.ProjectID,
		Title: m.Title, Content: "", CreatedAt: &m.UpdatedAt,
	})
}

// indexInstance écrit l'entrée d'index d'une instance de réunion.
//
// L'instance est indexée sous le titre de sa réunion : c'est ce que le
// résultat affiche, une instance n'ayant pas de nom propre (§2.9).
func (a *App) indexInstance(i domain.MeetingInstance) error {
	m, err := a.meetings.Get(i.MeetingID)
	if err != nil {
		return err
	}
	return a.index.Put(domain.IndexEntry{
		Type: domain.SearchTypeMeeting, EntityID: i.ID, ProjectID: m.ProjectID,
		Title: m.Title, Content: texteBrut(i.Notes), CreatedAt: &i.Timestamp,
	})
}

// reindexAll reconstruit l'index depuis les données.
//
// Appelé au démarrage. L'indexation est faite au fil des écritures, mais elle
// n'est pas dans la même transaction qu'elles : un arrêt entre les deux
// laisserait l'index en retard. Reconstruire coûte une lecture complète au
// lancement et supprime toute une classe d'incohérences silencieuses.
func (a *App) reindexAll() error {
	projets, err := a.projects.List()
	if err != nil {
		return err
	}
	for _, p := range projets {
		if err := a.index.DeleteByProject(p.ID); err != nil {
			return err
		}
		notes, err := a.notes.ListByProject(p.ID)
		if err != nil {
			return err
		}
		for _, n := range notes {
			if err := a.indexNote(n); err != nil {
				return err
			}
		}
		meetings, err := a.meetings.ListByProject(p.ID)
		if err != nil {
			return err
		}
		for _, m := range meetings {
			if err := a.indexMeeting(m); err != nil {
				return err
			}
		}
		instances, err := a.meetings.ListInstancesByProject(p.ID)
		if err != nil {
			return err
		}
		for _, i := range instances {
			if err := a.indexInstance(i); err != nil {
				return err
			}
		}
	}

	tasks, err := a.tasks.ListAll()
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := a.indexTask(t); err != nil {
			return err
		}
	}
	return nil
}

// keysOf rend les clés d'un ensemble sous forme de tranche.
func keysOf(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}
