// Commande seed : remplit la base avec un jeu d'essai réaliste.
//
// Elle sert à essayer l'application à la main — arbres de tâches, filtres,
// priorités, barre système, recherche mot-clé et recherche sémantique — sans
// avoir à saisir soi-même de quoi rendre ces écrans intéressants.
//
// # Utilisation
//
//	go run ./cmd/seed              # remplit la base réelle
//	go run ./cmd/seed -remove      # retire uniquement le jeu d'essai
//	go run ./cmd/seed -db chemin   # vise une autre base
//
// # Deux garanties, parce qu'elle écrit dans les vraies données
//
// Elle ne touche **que** ses propres projets, dont les identifiants commencent
// tous par « seed- ». Tes projets et le projet verrouillé « Transverse /
// Divers » ne sont jamais lus ni modifiés.
//
// Elle est **rejouable** : chaque exécution efface d'abord le jeu précédent.
// Relancer ne crée donc pas de doublons, et permet de repartir d'un état connu
// après avoir tout chamboulé en essayant l'application.
//
// # Ce que le jeu d'essai contient exprès
//
// Des échéances calées sur l'instant présent : une dans trois minutes pour voir
// clignoter la barre système, une dépassée, plusieurs à venir. Des tâches
// terminées et annulées pour éprouver les filtres. Des hiérarchies à trois
// niveaux. Et surtout, des **pièges sémantiques** : des textes qui ne
// contiennent pas les mots qu'on cherchera. Voir suggestionsDEssai plus bas.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"toutdoux/adapters/sqlite"
	"toutdoux/domain"
)

func main() {
	var (
		chemin  = flag.String("db", "", "chemin de la base (par défaut : celle de l'application)")
		retirer = flag.Bool("remove", false, "retirer le jeu d'essai sans le recréer")
	)
	flag.Parse()

	if *chemin == "" {
		defaut, err := cheminParDefaut()
		if err != nil {
			log.Fatalf("emplacement de la base : %v", err)
		}
		*chemin = defaut
	}

	db, err := sqlite.Open(*chemin)
	if err != nil {
		log.Fatalf("ouverture de %s : %v", *chemin, err)
	}
	defer db.Close()

	fmt.Printf("Base : %s\n\n", *chemin)

	projets := sqlite.NewProjectRepository(db)
	taches := sqlite.NewTaskRepository(db)
	notes := sqlite.NewNoteRepository(db)
	reunions := sqlite.NewMeetingRepository(db)

	// Nettoyage systématique, y compris avant un remplissage : c'est ce qui
	// rend la commande rejouable.
	efface := 0
	for _, p := range jeuDEssai {
		if _, err := projets.Get(p.ID); err != nil {
			continue // absent, rien à retirer
		}
		if err := projets.Delete(p.ID); err != nil {
			log.Fatalf("suppression de %s : %v", p.Nom, err)
		}
		efface++
	}
	if efface > 0 {
		fmt.Printf("Ancien jeu d'essai retiré : %d projets.\n", efface)
	}

	if *retirer {
		fmt.Println("Terminé.")
		return
	}

	maintenant := time.Now()
	var nbTaches, nbNotes, nbReunions, nbInstances int

	for _, p := range jeuDEssai {
		if err := projets.Create(domain.Project{
			ID: p.ID, Name: p.Nom, CreatedAt: maintenant, UpdatedAt: maintenant,
		}); err != nil {
			log.Fatalf("création du projet %s : %v", p.Nom, err)
		}

		n := ecrireTaches(taches, p, p.Taches, nil, maintenant)
		nbTaches += n

		for i, note := range p.Notes {
			if err := notes.Create(domain.Note{
				ID:        fmt.Sprintf("%s-n%d", p.ID, i+1),
				ProjectID: p.ID,
				Title:     note.Titre,
				Content:   note.Contenu,
				CreatedAt: maintenant.Add(-time.Duration(i+1) * 24 * time.Hour),
				UpdatedAt: maintenant.Add(-time.Duration(i+1) * time.Hour),
			}); err != nil {
				log.Fatalf("création d'une note de %s : %v", p.Nom, err)
			}
			nbNotes++
		}

		for i, r := range p.Reunions {
			idReunion := fmt.Sprintf("%s-r%d", p.ID, i+1)
			if err := reunions.Create(domain.Meeting{
				ID: idReunion, ProjectID: p.ID, Title: r.Titre,
				CreatedAt: maintenant.Add(-30 * 24 * time.Hour), UpdatedAt: maintenant,
			}); err != nil {
				log.Fatalf("création de la réunion %s : %v", r.Titre, err)
			}
			nbReunions++

			for j, inst := range r.Instances {
				// Les instances sont datées à reculons : la plus récente en
				// premier dans la liste du §2.7.
				quand := maintenant.Add(-time.Duration(j) * 7 * 24 * time.Hour)
				if err := reunions.CreateInstance(domain.MeetingInstance{
					ID: fmt.Sprintf("%s-i%d", idReunion, j+1), MeetingID: idReunion,
					Notes: inst, Timestamp: quand, CreatedAt: quand, UpdatedAt: quand,
				}); err != nil {
					log.Fatalf("création d'une instance de %s : %v", r.Titre, err)
				}
				nbInstances++
			}
		}

		fmt.Printf("  %-32s %2d tâches, %d notes, %d réunions\n",
			p.Nom, n, len(p.Notes), len(p.Reunions))
	}

	fmt.Printf("\n%d projets, %d tâches, %d notes, %d réunions, %d instances.\n",
		len(jeuDEssai), nbTaches, nbNotes, nbReunions, nbInstances)
	fmt.Print(suggestionsDEssai)
}

// ecrireTaches insère une branche de l'arbre et rend le nombre de tâches
// écrites.
//
// Les parents sont insérés avant leurs enfants : la colonne `parent_id` porte
// une clé étrangère vers `tasks`, l'ordre n'est donc pas négociable.
func ecrireTaches(repo *sqlite.TaskRepository, p projet, specs []tache, parent *string, maintenant time.Time) int {
	total := 0
	for i, s := range specs {
		id := prochainID(p.ID, parent, i)

		var echeance *time.Time
		if s.Dans != nil {
			t := maintenant.Add(*s.Dans)
			echeance = &t
		}

		if err := repo.Create(domain.Task{
			ID: id, ProjectID: p.ID, ParentID: parent,
			Name: s.Nom, Description: s.Description,
			Importance: s.Importance,
			Completed:  s.Terminee, Cancelled: s.Annulee,
			DueDate:    echeance,
			OrderIndex: i,
			CreatedAt:  maintenant.Add(-7 * 24 * time.Hour),
			UpdatedAt:  maintenant.Add(-time.Hour),
		}); err != nil {
			log.Fatalf("création de la tâche %q : %v", s.Nom, err)
		}
		total++

		if len(s.Enfants) > 0 {
			total += ecrireTaches(repo, p, s.Enfants, &id, maintenant)
		}
	}
	return total
}

// prochainID fabrique un identifiant stable et lisible.
//
// Stable, parce qu'un identifiant qui changerait à chaque exécution rendrait
// impossible de comparer deux essais. Lisible, parce qu'on finit toujours par
// lire ces identifiants dans les logs de la barre système.
func prochainID(projetID string, parent *string, rang int) string {
	if parent == nil {
		return fmt.Sprintf("%s-t%d", projetID, rang+1)
	}
	return fmt.Sprintf("%s-%d", *parent, rang+1)
}

func cheminParDefaut() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "ToutDoux")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "app.db"), nil
}

/* ---------------- Description du jeu d'essai ---------------- */

type projet struct {
	ID       string
	Nom      string
	Taches   []tache
	Notes    []note
	Reunions []reunion
}

type tache struct {
	Nom         string
	Description string
	Importance  domain.Importance
	// Dans est l'échéance, relative à l'instant du remplissage. Négatif pour une
	// échéance dépassée, nil pour aucune échéance.
	Dans     *time.Duration
	Terminee bool
	Annulee  bool
	Enfants  []tache
}

type note struct {
	Titre   string
	Contenu string // HTML, comme ce que produit l'éditeur
}

type reunion struct {
	Titre     string
	Instances []string // notes de chaque instance, en HTML
}

func dans(d time.Duration) *time.Duration { return &d }

const (
	critique = domain.ImportanceCritique
	haute    = domain.ImportanceHaute
	normale  = domain.ImportanceNormale
	basse    = domain.ImportanceBasse
)

var jeuDEssai = []projet{
	{
		ID:  "seed-sirh",
		Nom: "Migration SIRH",
		Taches: []tache{
			{
				Nom:         "Préparer la réunion de cadrage",
				Description: "Ordre du jour, support de présentation et liste des participants.",
				Importance:  critique,
				// Trois minutes : c'est sous le seuil des 5 minutes du §2.3, donc
				// l'icône de la barre système doit clignoter et une notification
				// doit partir.
				Dans: dans(3 * time.Minute),
				Enfants: []tache{
					{Nom: "Rassembler les chiffres de l'exercice", Importance: haute, Dans: dans(-2 * time.Hour),
						Description: "Effectifs, masse salariale, taux de rotation."},
					{Nom: "Relire le compte rendu du comité précédent", Importance: normale, Terminee: true,
						Enfants: []tache{
							{Nom: "Corriger les annexes chiffrées", Importance: basse, Terminee: true},
						}},
					{Nom: "Réserver la salle et le pont visio", Importance: normale, Dans: dans(2 * time.Hour)},
				},
			},
			{
				Nom:         "Migrer le référentiel des agents",
				Description: "Bascule de l'ancien référentiel vers la nouvelle plateforme, par lots de 500.",
				Importance:  haute,
				Dans:        dans(30 * time.Hour),
				Enfants: []tache{
					{Nom: "Détecter les doublons d'identité", Importance: haute},
					{Nom: "Écrire le script de reprise", Importance: normale, Dans: dans(6 * time.Hour)},
					{Nom: "Reprendre l'ancien périmètre régional", Importance: basse, Annulee: true,
						Description: "Abandonné : le périmètre régional sort du projet."},
				},
			},
			{Nom: "Former les gestionnaires de paie", Importance: normale, Dans: dans(20 * 24 * time.Hour)},
			{Nom: "Rédiger le plan de bascule", Importance: haute},
		},
		Notes: []note{
			{
				Titre: "Points durs de la migration",
				// Piège sémantique : ni « surcharge », ni « retard » n'apparaissent.
				Contenu: `<p>La reprise des données historiques prend beaucoup plus de temps que prévu.</p>
<p>L'équipe technique enchaîne les soirées depuis trois semaines et plusieurs personnes commencent à donner des signes d'épuisement. Il faudra arbitrer entre le périmètre et la date.</p>
<p>Le prestataire ne répond plus depuis vendredi.</p>`,
			},
			{
				Titre: "Compte rendu du comité du 12",
				Contenu: `<p>Échéance de bascule repoussée au 30.</p>
<ul><li>Périmètre régional retiré du projet</li><li>Deux gestionnaires supplémentaires affectés</li><li>Recette prévue sur la semaine 12</li></ul>`,
			},
			{
				Titre:   "Contacts utiles",
				Contenu: `<p>Référent fonctionnel : direction des ressources humaines, bureau 214.</p><p>Astreinte technique : voir l'annuaire interne.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Comité de pilotage SIRH",
				Instances: []string{
					`<p>Validation du report de la bascule. Le sponsor demande un point hebdomadaire jusqu'à la mise en production.</p>`,
					`<p>Présentation du plan de reprise. Discussion sur le périmètre régional, tranchée en fin de séance : il sort du projet.</p>`,
					`<p>Première réunion de lancement. Tour de table, rappel des objectifs et du calendrier initial.</p>`,
				},
			},
			{
				Titre: "Point technique hebdomadaire",
				Instances: []string{
					`<p>Les doublons d'identité sont plus nombreux qu'estimé : environ 4 % du référentiel. Script de dédoublonnage à écrire.</p>`,
					`<p>Revue de l'architecture cible. Aucune objection.</p>`,
				},
			},
		},
	},

	{
		ID:  "seed-portail",
		Nom: "Refonte du portail client",
		Taches: []tache{
			{
				Nom:         "Traiter la réclamation de la société Berthier",
				Description: "Dossier remonté par le commerce, à traiter en priorité.",
				Importance:  critique,
				Dans:        dans(-26 * time.Hour),
				Enfants: []tache{
					{Nom: "Reconstituer la chronologie de la livraison", Importance: haute},
					{Nom: "Proposer un geste commercial", Importance: haute, Dans: dans(4 * time.Hour)},
				},
			},
			{
				Nom:        "Maquetter la page d'accueil",
				Importance: haute,
				Dans:       dans(72 * time.Hour),
				Enfants: []tache{
					{Nom: "Variante mobile", Importance: normale},
					{Nom: "Variante impression", Importance: basse, Annulee: true},
				},
			},
			{Nom: "Recetter le parcours de connexion", Importance: normale, Terminee: true},
			{Nom: "Mesurer les temps de chargement", Importance: normale},
		},
		Notes: []note{
			{
				Titre: "Retour de la société Berthier",
				// Piège sémantique : « furieux », « exaspéré » — jamais « mécontent ».
				Contenu: `<p>Monsieur Berthier a rappelé ce matin. Il est furieux : la commande devait arriver mardi, elle est arrivée vide et sans document de transport.</p>
<p>Ton très dur au téléphone, il menace de partir à la concurrence. Il exige d'être rappelé par un responsable avant vendredi.</p>`,
			},
			{
				Titre: "Charte graphique du portail",
				Contenu: `<p>Palette validée par la direction de la communication.</p>
<p>Les contrastes doivent rester lisibles pour les personnes malvoyantes : ratio minimum de 4,5 pour le texte courant.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Revue de conception",
				Instances: []string{
					`<p>Maquettes présentées au commerce. Deux demandes de modification sur le tunnel de commande.</p>`,
					`<p>Cadrage graphique. La charte est reprise telle quelle, sans dérogation.</p>`,
				},
			},
			{
				Titre: "Point client hebdomadaire",
				Instances: []string{
					`<p>Le client revient sur l'incident de livraison. Ambiance tendue, il faudra un geste.</p>`,
				},
			},
		},
	},

	{
		ID:  "seed-contrats",
		Nom: "Contrats et fournisseurs",
		Taches: []tache{
			{
				// Piège sémantique : chercher « arrêter le contrat » doit trouver
				// « résilier l'abonnement ».
				Nom:         "Résilier l'abonnement à la solution de visioconférence",
				Description: "Préavis de trois mois, lettre recommandée avant le 15.",
				Importance:  haute,
				Dans:        dans(9 * 24 * time.Hour),
				Enfants: []tache{
					{Nom: "Retrouver les conditions générales signées", Importance: normale},
					{Nom: "Chiffrer le coût de sortie", Importance: haute, Dans: dans(48 * time.Hour)},
					{Nom: "Prévenir les équipes utilisatrices", Importance: normale},
				},
			},
			{
				Nom:        "Renégocier le marché d'impression",
				Importance: normale,
				Enfants: []tache{
					{Nom: "Comparer trois offres", Importance: normale, Terminee: true},
					{Nom: "Rédiger la note de synthèse", Importance: normale},
				},
			},
			{Nom: "Archiver les contrats échus en 2025", Importance: basse, Terminee: true},
		},
		Notes: []note{
			{
				Titre: "Conditions de sortie des contrats en cours",
				Contenu: `<p>Visioconférence : préavis de trois mois, pénalité forfaitaire si rupture avant l'échéance annuelle.</p>
<p>Impression : reconduction tacite chaque 1er janvier, dénonciation possible jusqu'au 30 novembre.</p>
<p>Hébergement : engagement de trente-six mois, aucune sortie anticipée prévue au contrat.</p>`,
			},
			{
				Titre:   "Interlocuteurs fournisseurs",
				Contenu: `<p>Chaque fournisseur a un référent commercial nommé. Passer par lui plutôt que par le support générique, les délais sont sans commune mesure.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Revue trimestrielle des achats",
				Instances: []string{
					`<p>Décision de ne pas reconduire la solution de visioconférence : usage marginal depuis le retour au bureau, coût par utilisateur devenu indéfendable.</p>`,
					`<p>Revue du portefeuille. Trois contrats arrivent à échéance dans l'année.</p>`,
				},
			},
		},
	},

	{
		ID:  "seed-securite",
		Nom: "Sécurité et conformité",
		Taches: []tache{
			{
				Nom:         "Clôturer l'incident du 3 août",
				Description: "Rapport à remettre au délégué à la protection des données.",
				Importance:  critique,
				Dans:        dans(45 * time.Minute),
				Enfants: []tache{
					{Nom: "Reconstituer le journal des accès", Importance: haute, Terminee: true},
					{Nom: "Évaluer les personnes concernées", Importance: critique, Dans: dans(30 * time.Minute)},
					{Nom: "Rédiger la notification", Importance: haute},
				},
			},
			{
				Nom:        "Campagne de renouvellement des mots de passe",
				Importance: normale,
				Dans:       dans(15 * 24 * time.Hour),
				Enfants: []tache{
					{Nom: "Préparer le message aux agents", Importance: normale},
					{Nom: "Prévoir le renfort du support", Importance: haute},
				},
			},
			{Nom: "Mettre à jour le registre des traitements", Importance: normale},
			{Nom: "Ancien audit externe", Importance: basse, Annulee: true},
		},
		Notes: []note{
			{
				Titre: "Chronologie de l'incident du 3 août",
				// Piège sémantique : « fuite », « exfiltration » — jamais « incident
				// de sécurité » ni « violation de données ».
				Contenu: `<p>Un compte de service disposant de droits étendus a été utilisé depuis une adresse inconnue pendant la nuit du 2 au 3.</p>
<p>Environ 4 000 fiches ont été extraites de la base avant que l'accès ne soit coupé. Les fichiers ont quitté le réseau interne : nom, adresse et coordonnées téléphoniques sont concernés.</p>
<p>Aucune donnée bancaire dans le périmètre.</p>`,
			},
			{
				Titre:   "Règles de conservation",
				Contenu: `<p>Journaux d'accès : douze mois. Enregistrements de support : six mois. Candidatures non retenues : deux ans, sauf opposition.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Cellule de crise",
				Instances: []string{
					`<p>Point de situation. La notification aux personnes concernées partira une fois la liste stabilisée.</p>`,
					`<p>Découverte de l'anomalie. Coupure immédiate du compte de service, réquisition des journaux.</p>`,
				},
			},
			{
				Titre: "Comité sécurité mensuel",
				Instances: []string{
					`<p>Retour d'expérience sur l'incident. Deux mesures retenues : rotation automatique des secrets, et alerte sur les connexions hors plage horaire.</p>`,
				},
			},
		},
	},

	{
		ID:  "seed-equipe",
		Nom: "Vie de l'équipe",
		Taches: []tache{
			{
				Nom:        "Organiser les entretiens annuels",
				Importance: haute,
				Dans:       dans(5 * 24 * time.Hour),
				Enfants: []tache{
					{Nom: "Envoyer les trames de préparation", Importance: normale, Terminee: true},
					{Nom: "Caler les créneaux", Importance: normale},
					{Nom: "Réserver un bureau au calme", Importance: basse},
				},
			},
			{
				Nom:        "Préparer l'arrivée de la nouvelle recrue",
				Importance: normale,
				Dans:       dans(10 * 24 * time.Hour),
				Enfants: []tache{
					{Nom: "Commander le poste de travail", Importance: haute, Dans: dans(-4 * time.Hour)},
					{Nom: "Ouvrir les accès applicatifs", Importance: normale},
					{Nom: "Désigner un référent d'accueil", Importance: normale},
				},
			},
			{Nom: "Planifier le séminaire d'automne", Importance: basse},
			{Nom: "Relancer le point café du vendredi", Importance: basse, Terminee: true},
		},
		Notes: []note{
			{
				Titre: "Retours des points individuels",
				// Piège sémantique : chercher « surcharge de travail » ou
				// « équipe fatiguée » doit trouver ce texte.
				Contenu: `<p>Trois personnes ont évoqué spontanément des semaines à rallonge et des week-ends entamés.</p>
<p>L'une d'elles dit ne plus arriver à décrocher le soir et envisage de demander une mutation. Le sujet revient à chaque entretien depuis six mois sans qu'aucune décision ne soit prise.</p>
<p>Il faut soit réduire le périmètre, soit renforcer l'effectif.</p>`,
			},
			{
				Titre:   "Idées pour le séminaire",
				Contenu: `<p>Format sur une journée et demie, hors des locaux. Un temps de travail le matin, une activité commune l'après-midi.</p>`,
			},
			{
				Titre:   "Trombinoscope et rôles",
				Contenu: `<p>Qui fait quoi, mis à jour après chaque arrivée ou départ. À afficher près de la machine à café.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Réunion d'équipe",
				Instances: []string{
					`<p>Annonce de l'arrivée d'un nouveau collègue le mois prochain. Discussion sur la répartition des dossiers.</p>`,
					`<p>Retour sur la charge de travail. Le sujet est posé, sans arbitrage à ce stade.</p>`,
					`<p>Organisation des congés d'été. Planning validé.</p>`,
				},
			},
		},
	},
}

// suggestionsDEssai est imprimé à la fin, parce qu'un jeu de données sans mode
// d'emploi ne sert qu'à celui qui l'a écrit.
const suggestionsDEssai = `
À ESSAYER
─────────

Barre système (§2.10)
  L'échéance « Préparer la réunion de cadrage » tombe dans 3 minutes : l'icône
  doit se mettre à clignoter et une notification doit partir. « Clôturer
  l'incident du 3 août » suit à 45 minutes.

Filtres (§2.4)
  Chaque projet a des tâches terminées et annulées. Décocher « Actives » doit
  vider l'arbre, les cocher toutes doit tout montrer.

Recherche mot-clé (§2.9)
  « échéance » ou « périmètre » remontent plusieurs projets, avec surbrillance.

Recherche sémantique (§2.12) — le vrai essai
  Ajoute d'abord les notes ci-dessous à l'index avec leur bouton 🧠, puis coche
  la bascule 🧠 dans la barre. Aucune des requêtes ne contient les mots du texte
  visé : c'est exactement ce que la recherche mot-clé ne sait pas faire.

    « client mécontent »      → Retour de la société Berthier   (dit « furieux »)
    « surcharge de travail »  → Retours des points individuels  (dit « semaines à rallonge »)
    « fuite de données »      → Chronologie de l'incident       (dit « extraites », « quitté le réseau »)
    « arrêter le contrat »    → Résilier l'abonnement           (tâche, à indexer aussi)
    « réuion de cadrage »     → Préparer la réunion de cadrage  (faute de frappe)

  Compare : les mêmes requêtes en mot-clé ne rendent rien.

Remise à zéro
  go run ./cmd/seed            relance à neuf
  go run ./cmd/seed -remove    retire tout le jeu d'essai
`
