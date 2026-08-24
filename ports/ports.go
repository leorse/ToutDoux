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
