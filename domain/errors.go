package domain

import "errors"

// Erreurs métier. Elles sont volontairement explicites plutôt que silencieuses :
// la spec insiste sur ce point pour l'anti-cycle et l'unicité des noms (§2.1, §2.2).
var (
	// ErrNotFound : l'entité visée n'existe pas.
	ErrNotFound = errors.New("entité introuvable")

	// ErrCycle : un déplacement ferait d'une tâche la sous-tâche d'un de ses
	// propres descendants (§2.2).
	ErrCycle = errors.New("déplacement impossible : une tâche ne peut pas devenir sous-tâche d'un de ses propres descendants")

	// ErrDuplicateName : nom de projet déjà pris, comparaison insensible à la
	// casse (§2.1).
	ErrDuplicateName = errors.New("un projet porte déjà ce nom")

	// ErrProjectLocked : tentative de renommage, de suppression ou de masquage
	// du projet "Transverse / Divers" (§2.1).
	ErrProjectLocked = errors.New("ce projet est verrouillé : il ne peut être ni renommé, ni supprimé, ni caché")

	// ErrInvalidImportance : importance hors des quatre valeurs autorisées.
	ErrInvalidImportance = errors.New("importance invalide")

	// ErrEmptyName : nom vide ou composé uniquement d'espaces.
	ErrEmptyName = errors.New("le nom ne peut pas être vide")

	// ErrInvalidLayout : disposition des notes incohérente — note manquante ou
	// en double, groupe inconnu, ou groupe dont les notes ne sont pas contiguës.
	ErrInvalidLayout = errors.New("disposition des notes invalide")

	// ErrInvalidColor : couleur hors de la palette des notes et des réunions.
	ErrInvalidColor = errors.New("couleur invalide")
)

// ErrModelUnavailable : le modèle sémantique n'est pas déposé, ou n'a pas pu
// être chargé (§2.12).
//
// L'application ne le télécharge jamais : c'est à l'utilisateur de le déposer.
// Cette erreur est donc un état normal du produit, pas une panne — l'interface
// la traduit en fenêtre explicative avec le chemin attendu (§2.13).
var ErrModelUnavailable = errors.New("le modèle sémantique n'est pas disponible")
