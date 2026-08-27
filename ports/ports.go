// Package ports définit les contrats que le domaine attend de l'extérieur (§3.9).
//
// Une interface n'est introduite ici que si l'une des deux conditions de la
// spec est remplie : une seconde implémentation existe ou est concrètement
// prévue, ou un besoin de test réel serait autrement pénible. Ce n'est pas une
// couche systématique — Settings, par exemple, reste un CRUD trivial appelé en
// direct, sans port.
//
// Les ports MeetingRepository et SearchIndex sont prévus par la spec mais
// n'apparaissent qu'en Phase 3, avec les fonctionnalités qui les consomment :
// les déclarer maintenant reviendrait à figer un contrat sans appelant.
package ports

import (
	"time"

	"toutdoux/domain"
)

// Clock rend l'heure courante.
//
// Ce port existe pour une raison de test, pas d'architecture : les règles
// d'échéance dépendent de l'instant (urgence sous 5 minutes, vendredi qui
// bascule au lundi). Sans horloge injectable, ces règles ne se testent qu'en
// attendant vraiment, ou en acceptant des tests qui échouent un vendredi.
type Clock interface {
	Now() time.Time
}

// ProjectRepository persiste les projets (§2.1).
type ProjectRepository interface {
	List() ([]domain.Project, error)
	Get(id string) (domain.Project, error)
	Create(p domain.Project) error
	Rename(id, newName string) error

	// Delete supprime le projet et, en cascade, ses tâches, notes, réunions et
	// instances (§2.1). L'implémentation doit être transactionnelle : une
	// cascade à moitié appliquée laisserait des orphelins.
	Delete(id string) error

	// ExistsByName teste l'unicité insensible à la casse, en ignorant
	// éventuellement un identifiant — celui du projet en cours de renommage,
	// qui ne doit pas entrer en conflit avec lui-même (§2.1).
	ExistsByName(name string, excludeID string) (bool, error)
}

// TaskRepository persiste les tâches (§2.2).
type TaskRepository interface {
	// ListByProject rend les tâches d'un projet, triées par order_index.
	ListByProject(projectID string) ([]domain.Task, error)

	// ListAll rend toutes les tâches, tous projets confondus. C'est ce que
	// consomment les vues transverses : Priorités (§2.8) et le menu de la barre
	// système (§2.10).
	ListAll() ([]domain.Task, error)

	Get(id string) (domain.Task, error)
	Create(t domain.Task) error
	Update(t domain.Task) error

	// UpdateMany applique en une seule transaction les tâches modifiées par une
	// cascade. C'est l'opération que réclame le domaine : une cascade de
	// réactivation partiellement écrite produirait exactement l'incohérence que
	// la règle cherche à empêcher.
	UpdateMany(tasks []domain.Task) error

	// Delete supprime la tâche et toute sa descendance (§2.2).
	Delete(id string) error
}

// NoteRepository persiste les notes (§2.6).
type NoteRepository interface {
	ListByProject(projectID string) ([]domain.Note, error)
	Get(id string) (domain.Note, error)
	Create(n domain.Note) error
	Update(n domain.Note) error
	Delete(id string) error
}

// MeetingRepository persiste les réunions et leurs instances (§2.7).
//
// Les deux vivent derrière la même interface parce qu'une instance n'a pas
// d'existence hors de sa réunion : elle n'est jamais listée ni supprimée
// indépendamment, et supprimer la réunion emporte ses instances.
type MeetingRepository interface {
	ListByProject(projectID string) ([]domain.Meeting, error)
	Get(id string) (domain.Meeting, error)
	Create(m domain.Meeting) error
	Rename(id, newTitle string) error
	Delete(id string) error

	// ListInstances rend les instances d'une réunion, la plus récente d'abord (§2.7).
	ListInstances(meetingID string) ([]domain.MeetingInstance, error)
	ListInstancesByProject(projectID string) ([]domain.MeetingInstance, error)
	GetInstance(id string) (domain.MeetingInstance, error)
	CreateInstance(i domain.MeetingInstance) error
	UpdateInstanceNotes(id, notes string) error
	DeleteInstance(id string) error
}

// SearchIndex alimente et interroge l'index plein texte (§2.9, §3.2).
//
// Ce port existe pour la première des deux raisons du §3.9 : le moteur peut
// changer. FTS5 est une fonctionnalité de SQLite, donc lié au choix de base ;
// une synchronisation cloud ou un moteur externe imposerait une autre
// implémentation sans que la recherche du domaine en soit affectée.
type SearchIndex interface {
	// Put insère ou remplace l'entrée d'une entité.
	Put(e domain.IndexEntry) error

	// Delete retire une entité de l'index. Sans erreur si elle n'y était pas :
	// l'appelant supprime des entités sans savoir si elles étaient indexées.
	Delete(entityID string) error

	// DeleteByProject retire toutes les entrées d'un projet, pour accompagner
	// la suppression en cascade du §2.1.
	DeleteByProject(projectID string) error

	// Search rend les entités correspondant à la requête, non triées : l'ordre
	// d'affichage est une règle du domaine (§2.9).
	Search(query string) ([]domain.IndexEntry, error)
}

// EmbeddingProvider transforme un texte en vecteur de sens (§2.12).
//
// C'est le second critère du §3.9 — un besoin de test réel, autrement pénible.
// Sans ce port, tester la moindre règle touchant à la recherche sémantique
// imposerait de charger un modèle de 120 Mo à *chaque* `go test` : la suite
// passerait de moins d'une seconde à plusieurs dizaines.
//
// Deux implémentations : celle qui appelle ONNX, et FakeEmbeddingProvider, qui
// rend un vecteur déterministe dérivé du texte. Le Fake couvre tout ce qui
// entoure le modèle — opt-in, revectorisation, non-recalcul, tri par score. Ce
// qu'il ne couvre pas, et ne prétend pas couvrir, c'est la qualité sémantique
// du vrai modèle (§3.10).
type EmbeddingProvider interface {
	// Embed rend le vecteur d'un texte **indexé**, ou
	// domain.ErrModelUnavailable si le modèle n'est pas déposé — ce qui est un
	// état normal, pas une panne.
	Embed(text string) ([]float32, error)

	// EmbedQuery rend le vecteur d'une **requête**.
	//
	// La distinction n'est pas cosmétique : la famille e5 est entraînée avec
	// deux préfixes, « query: » et « passage: », et les mélanger dégrade
	// nettement la pertinence. Le port porte donc les deux rôles plutôt que de
	// laisser l'appelant fabriquer un préfixe qui dépend du modèle choisi.
	EmbedQuery(text string) ([]float32, error)

	// Available indique si une vectorisation est possible dès maintenant.
	//
	// Elle existe pour éviter le coût d'un chargement de modèle quand on veut
	// seulement savoir si le bouton « Ajouter à la recherche sémantique » doit
	// être actif.
	Available() bool
}

// EmbeddingRepository persiste l'index sémantique (§3.2, table `embeddings`).
//
// Même statut que les autres repositories : le domaine énonce ce dont il a
// besoin, l'adaptateur SQLite s'y conforme. La sérialisation du vecteur en BLOB
// ne remonte jamais jusqu'ici.
type EmbeddingRepository interface {
	// Put insère ou remplace le vecteur d'une entité.
	Put(e domain.Embedding) error

	// Get rend le vecteur d'une entité, ou domain.ErrNotFound.
	Get(entityID string) (domain.Embedding, error)

	// Delete retire une entité de l'index sémantique, sans erreur si elle n'y
	// figurait pas.
	Delete(entityID string) error

	// DeleteByProject accompagne la suppression en cascade d'un projet (§2.1).
	DeleteByProject(projectID string) error

	// List rend tout l'index. La recherche balaie l'ensemble : le §3.2 assume
	// ce choix, quelques milliers de vecteurs se comparent en moins d'une
	// milliseconde et évitent une extension vectorielle native.
	List() ([]domain.Embedding, error)

	// Count rend le nombre d'entités indexées, pour distinguer « aucun
	// résultat » d'« index vide » (§2.12).
	Count() (int, error)
}
