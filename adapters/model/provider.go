package model

import "toutdoux/domain"

// UnavailableProvider est le fournisseur branché tant que le moteur
// d'inférence ONNX n'est pas câblé (étape 5.7 du §7).
//
// Il existe pour que l'application reste **cohérente** dans cet intervalle
// plutôt que d'être à moitié branchée : la recherche sémantique se déclare
// indisponible, le bouton « Ajouter à la recherche sémantique » est inactif, et
// la vue Préférences affiche la raison. Tout le reste — détection des fichiers,
// index, commandes, tri par score — est en place et testé.
//
// Le point important : il ne se déclare **jamais** disponible, même quand les
// trois fichiers du modèle sont bien déposés. Répondre `true` sans savoir
// vectoriser produirait le pire des comportements : un bouton actif, un clic,
// et une erreur incompréhensible.
type UnavailableProvider struct {
	// Raison est le message affiché à l'utilisateur.
	Raison string
}

// NewUnavailableProvider construit le fournisseur avec sa raison par défaut.
func NewUnavailableProvider() *UnavailableProvider {
	return &UnavailableProvider{
		Raison: "le moteur d'inférence ONNX n'est pas encore intégré à cette version",
	}
}

// Available rend toujours false.
func (p *UnavailableProvider) Available() bool { return false }

// Embed refuse toute vectorisation.
func (p *UnavailableProvider) Embed(string) ([]float32, error) {
	return nil, domain.ErrModelUnavailable
}

// EmbedQuery refuse elle aussi.
func (p *UnavailableProvider) EmbedQuery(string) ([]float32, error) {
	return nil, domain.ErrModelUnavailable
}
