package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"toutdoux/adapters/model"
	"toutdoux/adapters/onnx"
	"toutdoux/domain"
	"toutdoux/domain/search"
	"toutdoux/domain/semantic"
)

// ErrNothingToEmbed : l'entité ne porte aucun texte à vectoriser.
//
// Ajouter une note vide à l'index sémantique produirait un vecteur dépourvu de
// sens qui remonterait au hasard dans les résultats. Mieux vaut le refuser
// explicitement que d'accepter et de laisser l'utilisateur en déduire que la
// recherche fonctionne mal.
var ErrNothingToEmbed = errors.New("cette entité ne contient aucun texte à indexer")

/* ---------------- Statut du modèle (§2.13) ---------------- */

// GetModelStatus rend l'état des fichiers du modèle sémantique (§3.6).
//
// Une seule commande alimente la fenêtre du §2.12 et la vue Préférences du
// §2.13 : deux commandes distinctes pourraient diverger, et c'est exactement ce
// qu'un utilisateur cherchant pourquoi la recherche ne marche pas ne pardonne
// pas.
//
// Elle relit le disque à chaque appel, sans cache : c'est ce qui donne son sens
// au bouton « Réessayer », qui doit voir un fichier déposé une seconde plus tôt.
func (a *App) GetModelStatus() model.Status {
	s := model.Detect(a.modelDir)
	s.Runtime = model.File{
		Name:   onnx.NomRuntime,
		Source: "onnxruntime-win-x64-1.29.0.zip, dans lib/",
	}
	// Le chemin du runtime est connu du seul fournisseur : il le cherche à côté
	// de l'exécutable, puis dans le dossier courant, puis ici (§3.5).
	if chemin := onnx.TrouverRuntime(a.modelDir); chemin != "" {
		s.Runtime.Present = true
		if info, err := os.Stat(chemin); err == nil {
			s.Runtime.Size = info.Size()
		}
	}
	return s
}

// SemanticStatus est ce qu'il faut à l'interface pour décider quoi afficher
// avant même de chercher (§2.12).
type SemanticStatus struct {
	// ModelAvailable pilote l'activation du bouton « Ajouter à la recherche
	// sémantique » et l'ouverture de la fenêtre « modèle absent ». C'est le
	// fournisseur qui répond, pas le disque : les fichiers peuvent être là sans
	// que la vectorisation soit possible.
	ModelAvailable bool `json:"modelAvailable"`

	// FilesPresent dit si les trois fichiers sont déposés (§3.3).
	//
	// Il est distinct de ModelAvailable pour que l'interface puisse dire *quoi*
	// manque. Les deux faux : « déposez le modèle », avec le chemin. Fichiers
	// présents mais modèle indisponible : le dépôt est bon, c'est le moteur
	// d'inférence qui n'est pas branché — inutile d'envoyer l'utilisateur
	// retélécharger 120 Mo qu'il a déjà.
	FilesPresent bool `json:"filesPresent"`

	// IndexedCount distingue « aucun résultat » d'« index vide ». Sans lui, un
	// index vide afficherait « aucun résultat », ce qui ferait croire à une
	// absence de contenu correspondant plutôt qu'à une absence d'indexation.
	IndexedCount int `json:"indexedCount"`
}

// GetSemanticStatus rend l'état de la recherche sémantique.
func (a *App) GetSemanticStatus() (SemanticStatus, error) {
	n, err := a.embeddings.Count()
	if err != nil {
		return SemanticStatus{}, err
	}
	return SemanticStatus{
		ModelAvailable: a.embedder.Available(),
		FilesPresent:   model.Detect(a.modelDir).Available,
		IndexedCount:   n,
	}, nil
}

/* ---------------- Index sémantique (§2.12) ---------------- */

// AddToSemanticIndex vectorise une entité et l'enregistre (§3.6).
//
// C'est l'action du bouton « Ajouter à la recherche sémantique ». Rien n'entre
// dans l'index autrement : l'opt-in est strict, parce qu'une vectorisation
// coûte quelques centaines de millisecondes et que l'imposer sur tout le
// contenu serait un coût subi pour un bénéfice non demandé (§2.12).
func (a *App) AddToSemanticIndex(entityType string, entityID string) error {
	typ := domain.SearchType(entityType)
	texte, projectID, err := a.texteSource(typ, entityID)
	if err != nil {
		return err
	}
	if !a.embedder.Available() {
		return domain.ErrModelUnavailable
	}

	vecteur, err := a.embedder.Embed(texte)
	if err != nil {
		return err
	}
	return a.embeddings.Put(domain.Embedding{
		EntityID:   entityID,
		Type:       typ,
		ProjectID:  projectID,
		Vector:     vecteur,
		Dimensions: len(vecteur),
		SourceHash: semantic.Hash(texte),
	})
}

// RemoveFromSemanticIndex retire une entité de l'index sémantique.
func (a *App) RemoveFromSemanticIndex(entityID string) error {
	return a.embeddings.Delete(entityID)
}

// IsInSemanticIndex indique si une entité est vectorisée. Alimente l'état du
// bouton (§3.6).
func (a *App) IsInSemanticIndex(entityID string) (bool, error) {
	_, err := a.embeddings.Get(entityID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RefreshEmbedding revectorise silencieusement une entité déjà indexée (§2.12).
//
// Trois portes, dans cet ordre, et chacune évite un calcul :
//
//  1. l'entité n'est pas dans l'index — l'opt-in n'a pas été donné, on ne fait
//     rien ;
//  2. l'empreinte du texte est inchangée — c'est le cas dominant, la sauvegarde
//     automatique réenregistrant sans arrêt le même contenu ;
//  3. le modèle est absent — voir la limite connue ci-dessous.
//
// **Limite connue et assumée (§2.12).** Si le modèle a été retiré après
// l'indexation, la revectorisation échoue silencieusement et le vecteur reste
// celui de la version précédente du texte. Aucun indicateur ne le signale. Le
// cas suppose que l'utilisateur retire le modèle après s'en être servi, et la
// conséquence est un résultat légèrement périmé, pas une perte de données.
func (a *App) RefreshEmbedding(entityID string) error {
	existant, err := a.embeddings.Get(entityID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	texte, projectID, err := a.texteSource(existant.Type, entityID)
	if errors.Is(err, ErrNothingToEmbed) {
		// Le texte a été vidé : le vecteur ne correspond plus à rien. On retire
		// l'entité plutôt que de conserver un vecteur qui remonterait sur des
		// requêtes sans rapport.
		return a.embeddings.Delete(entityID)
	}
	if err != nil {
		return err
	}

	empreinte := semantic.Hash(texte)
	if empreinte == existant.SourceHash {
		return nil
	}
	if !a.embedder.Available() {
		log.Printf("[semantique] modele absent : le vecteur de %s reste celui du texte precedent", entityID)
		return nil
	}

	vecteur, err := a.embedder.Embed(texte)
	if err != nil {
		return err
	}
	return a.embeddings.Put(domain.Embedding{
		EntityID:   entityID,
		Type:       existant.Type,
		ProjectID:  projectID,
		Vector:     vecteur,
		Dimensions: len(vecteur),
		SourceHash: empreinte,
	})
}

// rafraichirEnArrierePlan lance une revectorisation hors du chemin de réponse.
//
// Sans cela, chaque sauvegarde automatique attendrait la vectorisation —
// quelques centaines de millisecondes — avant de rendre la main à l'éditeur.
// L'appel est délibérément sans valeur de retour : une revectorisation ratée ne
// doit jamais faire échouer l'enregistrement du texte, qui est la seule
// opération dont l'utilisateur a réellement besoin.
func (a *App) rafraichirEnArrierePlan(entityID string) {
	a.enArrierePlan(func() {
		if err := a.RefreshEmbedding(entityID); err != nil {
			log.Printf("[semantique] revectorisation de %s impossible : %v", entityID, err)
		}
	})
}

/* ---------------- Recherche sémantique (§2.12) ---------------- */

// SearchSemantic cherche par proximité de sens (§3.6).
//
// Les résultats ne sont **jamais** fusionnés avec ceux de SearchGlobal : ce
// sont deux modes que l'utilisateur choisit, pas deux sources d'une même liste.
// Le tri est par score décroissant et non par date, et l'extrait ne porte
// aucune surbrillance — aucun mot exact n'a été mis en correspondance, il n'y a
// donc rien à surligner (§2.12).
func (a *App) SearchSemantic(query string) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < search.MinQueryLength {
		return []SearchResult{}, nil
	}
	if !a.embedder.Available() {
		return nil, domain.ErrModelUnavailable
	}

	indexes, err := a.embeddings.List()
	if err != nil {
		return nil, err
	}
	if len(indexes) == 0 {
		return []SearchResult{}, nil
	}

	// EmbedQuery et non Embed : la famille e5 distingue une requête d'un passage
	// par un préfixe d'entraînement, et les confondre dégrade la pertinence.
	vecteur, err := a.embedder.EmbedQuery(query)
	if err != nil {
		return nil, err
	}

	candidats := make([]semantic.Candidate, 0, len(indexes))
	typeParID := make(map[string]domain.SearchType, len(indexes))
	for _, e := range indexes {
		candidats = append(candidats, semantic.Candidate{EntityID: e.EntityID, Vector: e.Vector})
		typeParID[e.EntityID] = e.Type
	}
	// Deux coupes : un plancher absolu contre l'aberration, puis une coupe
	// relative au meilleur score — c'est elle qui filtre réellement, l'échelle
	// de e5 n'ayant pas de signification absolue (voir domain/semantic).
	classement := semantic.CutRelative(
		semantic.Rank(vecteur, candidats, semantic.DefaultMinScore),
		semantic.DefaultMargin,
	)

	projets, err := a.projects.List()
	if err != nil {
		return nil, err
	}
	nomParProjet := make(map[string]string, len(projets))
	for _, p := range projets {
		nomParProjet[p.ID] = p.Name
	}

	out := make([]SearchResult, 0, len(classement))
	for _, m := range classement {
		if len(out) >= semantic.MaxResults {
			break
		}
		r, contenu, ok := a.chargerResultat(typeParID[m.EntityID], m.EntityID)
		if !ok {
			// Vecteur orphelin : l'entité a disparu sans passer par les
			// commandes de l'application. On l'ignore ici plutôt que d'écrire
			// pendant une lecture.
			continue
		}
		out = append(out, SearchResult{
			Result: r,
			// Extrait sans surbrillance : le terme cherché n'apparaît pas
			// forcément dans le texte, c'est tout l'intérêt du mode (§2.12).
			Snippet:     search.Highlight(contenu, "", search.DefaultContextChars),
			ProjectName: nomParProjet[r.ProjectID],
			Score:       m.Score,
		})
	}
	return out, nil
}

/* ---------------- Lecture du texte source ---------------- */

// texteSource assemble le texte à vectoriser pour une entité, et rend son
// projet de rattachement.
//
// Seuls les champs **textuels** entrent dans le calcul : ni l'importance, ni
// l'échéance, ni le statut, ni l'ordre. C'est ce qui garantit que cocher une
// case ne déclenche aucune vectorisation (§2.12).
func (a *App) texteSource(typ domain.SearchType, entityID string) (string, string, error) {
	var texte, projectID string

	switch typ {
	case domain.SearchTypeTask:
		t, err := a.tasks.Get(entityID)
		if err != nil {
			return "", "", err
		}
		texte, projectID = semantic.SourceText(t.Name, texteBrut(t.Description)), t.ProjectID

	case domain.SearchTypeNote:
		n, err := a.notes.Get(entityID)
		if err != nil {
			return "", "", err
		}
		texte, projectID = semantic.SourceText(n.Title, texteBrut(n.Content)), n.ProjectID

	case domain.SearchTypeMeeting:
		// Une réunion et ses instances partagent le même type dans l'index,
		// comme dans l'index plein texte : c'est l'identifiant qui tranche.
		if m, err := a.meetings.Get(entityID); err == nil {
			texte, projectID = semantic.SourceText(m.Title), m.ProjectID
			break
		}
		i, err := a.meetings.GetInstance(entityID)
		if err != nil {
			return "", "", err
		}
		m, err := a.meetings.Get(i.MeetingID)
		if err != nil {
			return "", "", err
		}
		// Le titre de la réunion accompagne les notes de l'instance : sans lui,
		// un compte rendu qui ne rappelle jamais son sujet perdrait le seul mot
		// qui le rattache à son contexte.
		texte, projectID = semantic.SourceText(m.Title, texteBrut(i.Notes)), m.ProjectID

	default:
		return "", "", fmt.Errorf("type d'entité inconnu pour l'index sémantique : %q", typ)
	}

	if strings.TrimSpace(texte) == "" {
		return "", "", ErrNothingToEmbed
	}
	return texte, projectID, nil
}

// chargerResultat construit la ligne de résultat d'une entité et rend son
// contenu en texte brut, prêt pour l'extrait.
//
// Partagé par les deux modes de recherche : c'est ce qui garantit qu'une ligne
// de résultat sémantique s'affiche exactement comme une ligne de résultat
// mot-clé, avec la même case à cocher et la même pastille d'importance (§2.12).
//
// Rend false si l'entité a disparu — entrée d'index orpheline, qu'on ignore.
func (a *App) chargerResultat(typ domain.SearchType, entityID string) (search.Result, string, bool) {
	r := search.Result{Type: search.Type(typ), ID: entityID}

	switch typ {
	case domain.SearchTypeTask:
		t, err := a.tasks.Get(entityID)
		if err != nil {
			return r, "", false
		}
		r.Task, r.Title, r.ProjectID, r.Date = &t, t.Name, t.ProjectID, t.DueDate
		return r, texteBrut(t.Description), true

	case domain.SearchTypeNote:
		n, err := a.notes.Get(entityID)
		if err != nil {
			return r, "", false
		}
		r.Note, r.Title, r.ProjectID, r.Date = &n, n.Title, n.ProjectID, &n.UpdatedAt
		return r, texteBrut(n.Content), true

	case domain.SearchTypeMeeting:
		if m, err := a.meetings.Get(entityID); err == nil {
			r.Meeting, r.Title, r.ProjectID, r.Date = &m, m.Title, m.ProjectID, &m.UpdatedAt
			return r, "", true
		}
		i, err := a.meetings.GetInstance(entityID)
		if err != nil {
			return r, "", false
		}
		m, err := a.meetings.Get(i.MeetingID)
		if err != nil {
			return r, "", false
		}
		// L'instance s'affiche sous le titre de sa réunion : elle n'a pas de
		// nom propre (§2.9).
		r.Instance, r.Date = &i, &i.Timestamp
		r.Meeting, r.Title, r.ProjectID = &m, m.Title, m.ProjectID
		return r, texteBrut(i.Notes), true
	}
	return r, "", false
}
