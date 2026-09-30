// Commande devdb : recrée la base de l'application de zéro, remplie d'un jeu de
// données de développement orienté IT.
//
// Réservée au développement. Contrairement à cmd/seed, qui ajoute ses projets à
// côté des vrais sans y toucher, devdb **remplace toute la base** : on s'en sert
// pour tout chambouler dans l'application, puis repartir d'une base saine.
//
// # Utilisation
//
//	go run ./cmd/devdb              # recrée la base de l'application
//	go run ./cmd/devdb -db chemin   # vise une autre base
//
// L'application doit être fermée : sous Windows, un fichier SQLite ouvert ne
// peut pas être supprimé, et la commande s'arrête avec un message explicite.
//
// # Ce qui est effacé, et ce qui ne l'est pas
//
// Seuls app.db et ses fichiers de journal (-wal, -shm) sont concernés. Ils ne
// sont pas supprimés mais renommés en app.db.bak* — écrasés à chaque appel —
// pour qu'un lancement malheureux sur de vraies données reste rattrapable.
// Le dossier models/ voisin, qui peut contenir un modèle de 120 Mo, n'est
// jamais touché.
//
// L'index de recherche n'est pas rempli ici : l'application le reconstruit
// à chaque démarrage (voir App.startup).
//
// # Ce que le jeu de données contient
//
//   - six projets en cours, plus « Transverse / Divers », avec des tâches
//     actives partout, sur trois niveaux, dans tous les états ;
//   - trois projets terminés et cachés, dont un garde une tâche oubliée en
//     retard : le masquage étant purement visuel, elle doit rester visible dans
//     Priorités et la barre système ;
//   - des notes et des réunions cachées, dans les projets visibles comme dans
//     les projets cachés ;
//   - des échéances calées sur l'instant présent : une à trois minutes (la
//     barre système clignote), plusieurs dépassées, d'autres à venir.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"toutdoux/adapters/sqlite"
	"toutdoux/domain"
)

func main() {
	chemin := flag.String("db", "", "chemin de la base (par défaut : celle de l'application)")
	flag.Parse()

	if *chemin == "" {
		defaut, err := cheminParDefaut()
		if err != nil {
			log.Fatalf("emplacement de la base : %v", err)
		}
		*chemin = defaut
	}

	fmt.Printf("Base : %s\n\n", *chemin)

	sauvegarde, err := mettreDeCote(*chemin)
	if err != nil {
		log.Fatalf("%v\nL'application est-elle encore ouverte ? Ferme-la, y compris depuis la barre système.", err)
	}

	// Open rejoue toutes les migrations sur le fichier neuf et recrée le projet
	// verrouillé « Transverse / Divers ».
	db, err := sqlite.Open(*chemin)
	if err != nil {
		log.Fatalf("création de %s : %v", *chemin, err)
	}
	defer db.Close()

	projets := sqlite.NewProjectRepository(db)
	taches := sqlite.NewTaskRepository(db)
	notes := sqlite.NewNoteRepository(db)
	reunions := sqlite.NewMeetingRepository(db)

	maintenant := time.Now()
	var nbTaches, nbNotes, nbReunions, nbInstances int

	for _, p := range jeuDeDonnees {
		debut := maintenant.Add(-p.Age)
		derniere := maintenant.Add(-p.Inactif)

		// « Transverse / Divers » existe déjà : Open vient de le créer.
		if p.ID != domain.DiversProjectID {
			if err := projets.Create(domain.Project{
				ID: p.ID, Name: p.Nom, Hidden: p.Cache,
				CreatedAt: debut, UpdatedAt: derniere,
			}); err != nil {
				log.Fatalf("création du projet %s : %v", p.Nom, err)
			}
		}

		n := ecrireTaches(taches, p, p.Taches, nil, maintenant)
		nbTaches += n

		for i, note := range p.Notes {
			if err := notes.Create(domain.Note{
				ID:        fmt.Sprintf("%s-n%d", p.ID, i+1),
				ProjectID: p.ID,
				Title:     note.Titre,
				Content:   note.Contenu,
				Hidden:    note.Cachee,
				CreatedAt: debut.Add(time.Duration(i+1) * 24 * time.Hour),
				UpdatedAt: derniere.Add(-time.Duration(i+1) * time.Hour),
			}); err != nil {
				log.Fatalf("création de la note %q : %v", note.Titre, err)
			}
			nbNotes++
		}

		for i, r := range p.Reunions {
			idReunion := fmt.Sprintf("%s-r%d", p.ID, i+1)
			if err := reunions.Create(domain.Meeting{
				ID: idReunion, ProjectID: p.ID, Title: r.Titre, Hidden: r.Cachee,
				CreatedAt: debut, UpdatedAt: derniere,
			}); err != nil {
				log.Fatalf("création de la réunion %q : %v", r.Titre, err)
			}
			nbReunions++

			periode := r.Periode
			if periode == 0 {
				periode = 7 * 24 * time.Hour
			}
			for j, inst := range r.Instances {
				// Datées à reculons depuis la plus récente, à 10 h : la liste
				// des instances s'affiche de la plus récente à la plus ancienne.
				quand := a10h(maintenant.Add(-r.Depuis - time.Duration(j)*periode))
				if err := reunions.CreateInstance(domain.MeetingInstance{
					ID: fmt.Sprintf("%s-i%d", idReunion, j+1), MeetingID: idReunion,
					Notes: inst, Timestamp: quand, CreatedAt: quand, UpdatedAt: quand,
				}); err != nil {
					log.Fatalf("création d'une instance de %q : %v", r.Titre, err)
				}
				nbInstances++
			}
		}

		etat := ""
		if p.Cache {
			etat = "(caché)"
		}
		fmt.Printf("  %-34s %-8s %3d tâches, %d notes, %d réunions\n",
			p.Nom, etat, n, len(p.Notes), len(p.Reunions))
	}

	fmt.Printf("\n%d projets, %d tâches, %d notes, %d réunions, %d instances.\n",
		len(jeuDeDonnees), nbTaches, nbNotes, nbReunions, nbInstances)
	if sauvegarde {
		fmt.Printf("Ancienne base conservée en %s.\n", filepath.Base(*chemin)+".bak")
	}
	fmt.Println("Tu peux relancer l'application.")
}

// mettreDeCote renomme la base et ses journaux en .bak, en écrasant la
// sauvegarde précédente.
//
// Les journaux suivent la base sous le même préfixe : SQLite retrouve un WAL
// par le nom de sa base, la sauvegarde reste donc ouvrable telle quelle.
// Rend vrai si une base existait.
func mettreDeCote(chemin string) (bool, error) {
	_, err := os.Stat(chemin)
	existait := err == nil
	for _, suffixe := range []string{"", "-wal", "-shm"} {
		source := chemin + suffixe
		cible := chemin + ".bak" + suffixe

		if err := os.Remove(cible); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("suppression de l'ancienne sauvegarde %s : %w", cible, err)
		}
		if err := os.Rename(source, cible); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("mise de côté de %s : %w", source, err)
		}
	}
	return existait, nil
}

// ecrireTaches insère une branche de l'arbre et rend le nombre de tâches
// écrites. Les parents passent avant leurs enfants : parent_id est une clé
// étrangère vers tasks.
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
			CreatedAt:  maintenant.Add(-p.Age).Add(time.Duration(i+1) * time.Hour),
			UpdatedAt:  maintenant.Add(-p.Inactif).Add(-time.Duration(i+1) * time.Hour),
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

// prochainID fabrique un identifiant stable et lisible, identique d'une
// exécution à l'autre.
func prochainID(projetID string, parent *string, rang int) string {
	if parent == nil {
		return fmt.Sprintf("%s-t%d", projetID, rang+1)
	}
	return fmt.Sprintf("%s-%d", *parent, rang+1)
}

func a10h(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 10, 0, 0, 0, t.Location())
}

// cheminParDefaut reprend databasePath de l'application.
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

/* ---------------- Description du jeu de données ---------------- */

type projet struct {
	ID    string
	Nom   string
	Cache bool
	// Age est l'ancienneté du projet, Inactif le temps écoulé depuis sa
	// dernière activité (zéro pour un projet en cours).
	Age, Inactif time.Duration
	Taches       []tache
	Notes        []note
	Reunions     []reunion
}

type tache struct {
	Nom         string
	Description string
	Importance  domain.Importance
	// Dans est l'échéance relative à l'instant du remplissage : négative si
	// dépassée, nil si aucune.
	Dans     *time.Duration
	Terminee bool
	Annulee  bool
	Enfants  []tache
}

type note struct {
	Titre   string
	Contenu string // HTML, comme ce que produit l'éditeur
	Cachee  bool
}

type reunion struct {
	Titre  string
	Cachee bool
	// Depuis est l'ancienneté de l'instance la plus récente, Periode l'écart
	// entre deux instances (une semaine si zéro).
	Depuis, Periode time.Duration
	Instances       []string // notes de chaque instance, en HTML, la plus récente d'abord
}

func dans(d time.Duration) *time.Duration { return &d }

const (
	minute = time.Minute
	heure  = time.Hour
	jour   = 24 * time.Hour

	critique = domain.ImportanceCritique
	haute    = domain.ImportanceHaute
	normale  = domain.ImportanceNormale
	basse    = domain.ImportanceBasse
)

var jeuDeDonnees = []projet{
	/* ======== Projets en cours ======== */

	{
		ID: "dev-k8s", Nom: "Plateforme Kubernetes", Age: 90 * jour,
		Taches: []tache{
			{
				Nom:         "Mettre en place le cluster de production",
				Description: "Cluster AKS à trois pools de nœuds : système, applicatif, batch.",
				Importance:  haute, Dans: dans(5 * jour),
				Enfants: []tache{
					{Nom: "Provisionner les nœuds via Terraform", Importance: haute, Terminee: true},
					{
						Nom: "Configurer l'ingress NGINX et cert-manager", Importance: haute, Dans: dans(1 * jour),
						Enfants: []tache{
							{Nom: "Générer le certificat wildcard", Importance: normale, Terminee: true},
							{Nom: "Tester le renouvellement automatique", Importance: normale,
								Description: "Forcer une expiration sur l'environnement de recette."},
						},
					},
					{Nom: "Activer les Network Policies", Importance: normale},
					{Nom: "Brancher le stockage persistant (CSI)", Importance: haute, Dans: dans(-1 * jour),
						Description: "Bloqué par la classe de stockage premium, en attente du ticket Azure."},
				},
			},
			{
				Nom:        "Migrer les applications métier",
				Importance: haute, Dans: dans(20 * jour),
				Enfants: []tache{
					{Nom: "Conteneuriser l'application RH", Importance: normale, Terminee: true},
					{Nom: "Écrire les charts Helm du portail", Importance: normale, Dans: dans(3 * jour)},
					{
						Nom: "Migrer le batch de facturation", Importance: haute,
						Enfants: []tache{
							{Nom: "Remplacer les crontabs par des CronJobs", Importance: normale},
							{Nom: "Externaliser les secrets vers Vault", Importance: haute, Dans: dans(2 * jour)},
							{Nom: "Valider les volumes de la nuit de clôture", Importance: normale, Terminee: true},
						},
					},
					{Nom: "Migrer l'intranet IIS", Importance: basse, Annulee: true,
						Description: "Abandonné : reste sur VM Windows, pas de conteneurs Windows sur ce cluster."},
				},
			},
			{
				Nom: "Mettre en place GitOps avec Argo CD", Importance: normale, Dans: dans(10 * jour),
				Enfants: []tache{
					{Nom: "Structurer le dépôt de manifests", Importance: normale, Terminee: true},
					{Nom: "Définir la promotion dev → recette → prod", Importance: normale},
					{Nom: "Brancher les notifications Teams", Importance: basse},
				},
			},
			{Nom: "Rédiger le runbook d'exploitation", Importance: normale},
			{Nom: "Évaluer Rancher", Importance: basse, Annulee: true,
				Description: "Hors budget, et AKS couvre le besoin."},
		},
		Notes: []note{
			{
				Titre: "Architecture cible du cluster",
				Contenu: `<h2>Pools de nœuds</h2>
<ul><li><b>system</b> : 3 × D4s v5, composants du cluster uniquement</li><li><b>apps</b> : 3 à 8 × D8s v5, autoscaling</li><li><b>batch</b> : 0 à 4 × F16s v2, spot, taint <code>workload=batch</code></li></ul>
<h2>Réseau</h2>
<p>Azure CNI overlay, ingress NGINX derrière l'Application Gateway. Les Network Policies sont en <i>deny all</i> par défaut, chaque namespace déclare ses flux.</p>`,
			},
			{
				Titre: "Conventions de nommage",
				Contenu: `<p>Namespaces : <code>&lt;équipe&gt;-&lt;application&gt;-&lt;environnement&gt;</code>, par exemple <code>fin-facturation-prod</code>.</p>
<p>Labels obligatoires : <code>app.kubernetes.io/name</code>, <code>app.kubernetes.io/part-of</code>, <code>owner</code>.</p>`,
			},
			{
				Titre:   "Commandes kubectl utiles",
				Contenu: `<ul><li><code>kubectl get pods -A --field-selector=status.phase!=Running</code></li><li><code>kubectl top nodes</code></li><li><code>kubectl rollout restart deploy/&lt;nom&gt; -n &lt;ns&gt;</code></li></ul>`,
			},
			{
				Titre: "Comparatif AKS / EKS / on-premise", Cachee: true,
				Contenu: `<p>AKS retenu : contrat Azure existant, intégration Entra ID, coût du plan de contrôle nul. EKS écarté faute de compétences AWS en interne ; on-premise écarté pour la charge d'exploitation.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Comité technique plateforme", Depuis: 2 * jour,
				Instances: []string{
					`<p>Le stockage CSI bloque toujours. Escalade du ticket Azure chez le TAM.</p><ul><li>Décision : on démarre la migration du portail sans volume persistant</li></ul>`,
					`<p>Revue de la stratégie GitOps. Un dépôt par équipe, un dépôt commun pour les composants du cluster.</p>`,
					`<p>Point sur le provisionnement Terraform : terminé, état stocké dans un compte de stockage dédié avec verrouillage.</p>`,
					`<p>Lancement des travaux. Planning validé, cible de mise en production dans trois mois.</p>`,
				},
			},
			{
				Titre: "Atelier Helm avec les équipes de développement", Depuis: 9 * jour, Periode: 14 * jour,
				Instances: []string{
					`<p>Deuxième session : gestion des valeurs par environnement, et pourquoi on ne met pas de secret dans <code>values.yaml</code>.</p>`,
					`<p>Premiers pas : structure d'un chart, templates, <code>helm template</code> pour relire le rendu.</p>`,
				},
			},
			{
				Titre: "Lancement Kubernetes", Cachee: true, Depuis: 88 * jour,
				Instances: []string{
					`<p>Présentation du projet aux équipes. Questions surtout sur la formation et l'astreinte.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-factu", Nom: "API de facturation v2", Age: 120 * jour,
		Taches: []tache{
			{
				Nom:         "Corriger l'arrondi de TVA sur les avoirs",
				Description: "Ticket FACT-1432 : écart de 0,01 € sur les avoirs multilignes, signalé par la comptabilité.",
				Importance:  critique, Dans: dans(-2 * heure),
				Enfants: []tache{
					{Nom: "Reproduire sur un jeu de données anonymisé", Importance: haute, Terminee: true},
					{Nom: "Passer les montants en décimal", Importance: haute,
						Description: "Remplacer les double par BigDecimal dans le calcul des lignes."},
					{Nom: "Ajouter un test de non-régression", Importance: haute},
				},
			},
			{
				Nom: "Découper le monolithe en services", Importance: haute, Dans: dans(15 * jour),
				Enfants: []tache{
					{Nom: "Identifier les bounded contexts", Importance: normale, Terminee: true},
					{
						Nom: "Extraire le service de tarification", Importance: haute, Dans: dans(4 * heure),
						Enfants: []tache{
							{Nom: "Écrire les tests de contrat", Importance: normale},
							{Nom: "Brancher le cache Redis", Importance: normale, Terminee: true},
							{Nom: "Mesurer la latence au 99e centile", Importance: basse},
						},
					},
					{Nom: "Extraire le service d'édition PDF", Importance: normale},
				},
			},
			{Nom: "Documenter l'API en OpenAPI 3.1", Importance: normale, Dans: dans(7 * jour)},
			{Nom: "Limiter le débit sur la passerelle", Importance: normale,
				Description: "100 requêtes par seconde et par client, réponse 429 au-delà."},
			{
				Nom: "Monter en version Java 21", Importance: normale, Dans: dans(30 * jour),
				Enfants: []tache{
					{Nom: "Vérifier la compatibilité des dépendances", Importance: normale, Terminee: true},
					{Nom: "Adapter la pipeline CI", Importance: normale},
					{Nom: "Activer les threads virtuels", Importance: basse},
				},
			},
			{Nom: "Réécrire le moteur de calcul en Rust", Importance: basse, Annulee: true,
				Description: "Idée du vendredi soir. Aucun gain mesurable justifiant le coût."},
		},
		Notes: []note{
			{
				Titre: "Contrat d'interface avec la comptabilité",
				Contenu: `<p>Export quotidien à 2 h, format CSV UTF-8 séparé par des points-virgules.</p>
<ul><li>Montants en centimes, entiers signés</li><li>Un avoir porte la référence de sa facture d'origine</li><li>Toute modification du format est annoncée un mois à l'avance</li></ul>`,
			},
			{
				Titre: "Décisions d'architecture",
				Contenu: `<h2>ADR-004 : communication entre services</h2><p>Appels synchrones REST pour les lectures, événements Kafka pour les changements d'état. Pas de base partagée entre services.</p>
<h2>ADR-005 : identifiants</h2><p>UUID v7 partout, pour garder un tri chronologique sans colonne supplémentaire.</p>`,
			},
			{
				Titre:   "Analyse de l'écart d'arrondi",
				Contenu: `<p>Chaque ligne est arrondie avant la somme, alors que la comptabilité arrondit le total. Sur un avoir de 7 lignes à 33,33 €, l'écart atteint un centime.</p><p>Correctif : calculer en décimal exact, n'arrondir qu'une fois, au total.</p>`,
			},
			{
				Titre: "Ancienne spécification SOAP", Cachee: true,
				Contenu: `<p>WSDL v1.3 de l'ancien service de facturation. Conservée pour mémoire jusqu'à l'extinction du dernier client SOAP.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Daily équipe facturation", Depuis: 0, Periode: 1 * jour,
				Instances: []string{
					`<p>Arrondi TVA : correctif en cours de revue. Tarification : tests de contrat en retard, Karim en renfort.</p>`,
					`<p>Reproduction de l'écart d'arrondi réussie. Le correctif passe par le décimal.</p>`,
					`<p>Nouveau ticket critique de la comptabilité sur les avoirs. On bascule dessus.</p>`,
					`<p>Cache Redis branché sur la tarification, temps de réponse divisé par quatre.</p>`,
					`<p>Rien de bloquant. Revue de code en attente sur la PR du service PDF.</p>`,
				},
			},
			{
				Titre: "Revue d'architecture", Depuis: 6 * jour, Periode: 21 * jour,
				Instances: []string{
					`<p>Validation de l'ADR-004. Réserve de l'équipe sécurité sur le chiffrement des topics Kafka : à traiter avant la production.</p>`,
					`<p>Présentation du découpage en bounded contexts. Tarification en premier, édition ensuite.</p>`,
				},
			},
			{
				Titre: "Démo de fin de sprint", Depuis: 3 * jour, Periode: 14 * jour,
				Instances: []string{
					`<p>Démo du service de tarification isolé. Le métier demande l'historique des grilles tarifaires.</p>`,
					`<p>Démo de la documentation OpenAPI générée. Bon accueil.</p>`,
					`<p>Démo du nouveau pipeline CI, build en 6 minutes au lieu de 22.</p>`,
				},
			},
			{
				Titre: "Réversibilité avec l'ancien prestataire", Cachee: true, Depuis: 70 * jour,
				Instances: []string{
					`<p>Dernière séance. Code source et documentation remis, accès révoqués.</p>`,
					`<p>Transfert de connaissances sur le moteur de calcul.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-secu", Nom: "Sécurité du SI", Age: 200 * jour,
		Taches: []tache{
			{
				Nom:         "Appliquer le correctif OpenSSL sur les frontaux",
				Description: "CVE critique publiée hier soir, exploit public disponible.",
				Importance:  critique,
				// Sous le seuil des 5 minutes de la barre système : l'icône doit
				// clignoter et une notification doit partir.
				Dans: dans(3 * minute),
				Enfants: []tache{
					{Nom: "Inventorier les serveurs exposés", Importance: haute, Terminee: true},
					{Nom: "Patcher la préproduction", Importance: haute, Terminee: true},
					{Nom: "Patcher la production en fenêtre de maintenance", Importance: critique, Dans: dans(45 * minute)},
					{Nom: "Vérifier les versions avec le scanner", Importance: normale, Dans: dans(2 * heure)},
				},
			},
			{
				Nom: "Déployer le MFA pour tous les comptes", Importance: haute, Dans: dans(12 * jour),
				Enfants: []tache{
					{Nom: "Activer le MFA des administrateurs", Importance: critique, Terminee: true},
					{Nom: "Communiquer auprès des utilisateurs", Importance: normale, Dans: dans(2 * jour)},
					{
						Nom: "Traiter les comptes de service", Importance: haute,
						Enfants: []tache{
							{Nom: "Lister les comptes de service", Importance: normale, Terminee: true},
							{Nom: "Passer les comptes éligibles en identités managées", Importance: haute},
							{Nom: "Documenter les exceptions restantes", Importance: normale},
						},
					},
				},
			},
			{
				Nom: "Préparer le test d'intrusion externe", Importance: haute, Dans: dans(6 * jour),
				Enfants: []tache{
					{Nom: "Définir le périmètre avec le prestataire", Importance: normale, Terminee: true},
					{Nom: "Créer les comptes de test", Importance: normale},
					{Nom: "Prévenir le SOC de la fenêtre de test", Importance: haute, Dans: dans(5 * jour)},
				},
			},
			{Nom: "Revue trimestrielle des droits Active Directory", Importance: normale, Dans: dans(-3 * jour)},
			{
				Nom: "Remplacer l'antivirus par un EDR", Importance: haute, Dans: dans(25 * jour),
				Enfants: []tache{
					{Nom: "POC sur 20 postes", Importance: normale, Terminee: true},
					{Nom: "Déploiement par vagues", Importance: normale},
					{Nom: "Désinstaller l'ancien antivirus", Importance: basse},
				},
			},
			{Nom: "Chiffrer les sauvegardes hors site", Importance: normale},
			{Nom: "Audit ISO 27001 blanc", Importance: basse, Annulee: true,
				Description: "Reporté à l'an prochain, décision du comité."},
		},
		Notes: []note{
			{
				Titre:   "Politique de mots de passe",
				Contenu: `<ul><li>14 caractères minimum, sans règle de complexité imposée</li><li>Pas d'expiration périodique, changement forcé en cas de compromission</li><li>Contrôle contre les listes de mots de passe divulgués</li></ul>`,
			},
			{
				Titre:   "Contacts CERT et SOC",
				Contenu: `<p>SOC externalisé : astreinte 24/7, numéro dans l'annuaire interne, fiche « Sécurité ».</p><p>CERT-FR : signalement via le formulaire en ligne. Déclaration CNIL sous 72 h en cas de violation de données personnelles.</p>`,
			},
			{
				Titre:   "Résultats du POC EDR",
				Contenu: `<p>20 postes pendant 4 semaines. 3 vraies détections (macros Office, outil de prise en main à distance non autorisé), 11 faux positifs sur les outils de développement, tous traités par exclusion.</p><p>Impact CPU moyen inférieur à 2 %.</p>`,
			},
			{
				Titre: "Rapport du test d'intrusion 2025", Cachee: true,
				Contenu: `<p>5 vulnérabilités dont une critique (injection sur l'ancien extranet, depuis décommissionné). Toutes corrigées au 15 mars.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Cellule de crise OpenSSL", Depuis: 0, Periode: 1 * jour,
				Instances: []string{
					`<p>Préprod patchée sans régression. Fenêtre de maintenance production confirmée ce matin.</p>`,
					`<p>Publication de la CVE. Inventaire lancé : 14 serveurs exposés sur Internet.</p>`,
				},
			},
			{
				Titre: "Comité sécurité mensuel", Depuis: 12 * jour, Periode: 30 * jour,
				Instances: []string{
					`<p>MFA : administrateurs couverts à 100 %. Utilisateurs prévus le mois prochain.</p><p>Décision : lancement du test d'intrusion externe.</p>`,
					`<p>Présentation des résultats du POC EDR. Validation du déploiement général.</p>`,
					`<p>Revue des indicateurs : 97 % des postes à jour, 12 comptes dormants désactivés.</p>`,
				},
			},
			{
				Titre: "Suivi de l'audit 2025", Cachee: true, Depuis: 180 * jour,
				Instances: []string{
					`<p>Clôture de l'audit. Plan d'action intégralement réalisé.</p>`,
					`<p>Restitution du rapport d'audit.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-obs", Nom: "Observabilité", Age: 45 * jour,
		Taches: []tache{
			{
				Nom: "Déployer Prometheus et Grafana", Importance: haute, Dans: dans(2 * jour),
				Enfants: []tache{
					{Nom: "Installer kube-prometheus-stack", Importance: haute, Terminee: true},
					{
						Nom: "Créer les tableaux de bord par application", Importance: normale,
						Enfants: []tache{
							{Nom: "Tableau de bord API de facturation", Importance: normale, Terminee: true},
							{Nom: "Tableau de bord portail client", Importance: normale},
							{Nom: "Tableau de bord batchs de nuit", Importance: basse},
						},
					},
					{Nom: "Définir les alertes et les SLO", Importance: haute, Dans: dans(1 * jour)},
				},
			},
			{Nom: "Configurer les rotations d'astreinte", Importance: haute, Dans: dans(-1 * heure),
				Description: "Rotation hebdomadaire, relève le lundi à 9 h, escalade au bout de 15 minutes."},
			{Nom: "Centraliser les logs dans Loki", Importance: normale, Dans: dans(14 * jour)},
			{Nom: "Instrumenter les services avec OpenTelemetry", Importance: normale},
			{Nom: "Décommissionner Nagios", Importance: basse, Dans: dans(40 * jour)},
		},
		Notes: []note{
			{
				Titre:   "SLO par service",
				Contenu: `<ul><li>API de facturation : 99,9 % de disponibilité, p95 &lt; 300 ms</li><li>Portail client : 99,5 %, p95 &lt; 800 ms</li><li>Batch de nuit : terminé avant 6 h, 29 nuits sur 30</li></ul>`,
			},
			{
				Titre:   "Bonnes pratiques d'alerting",
				Contenu: `<p>Une alerte qui réveille quelqu'un doit être actionnable. Alerter sur les symptômes vus par l'utilisateur, pas sur les causes : un CPU à 90 % n'est pas un incident.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Point observabilité", Depuis: 4 * jour, Periode: 14 * jour,
				Instances: []string{
					`<p>Premiers tableaux de bord en ligne. Les équipes demandent l'accès en lecture à Grafana.</p>`,
					`<p>Choix de la stack : Prometheus, Grafana, Loki. Pas de solution SaaS pour des raisons de coût.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-run", Nom: "Support et exploitation", Age: 365 * jour,
		Taches: []tache{
			{
				Nom:         "INC-2291 : lenteurs sur le VPN",
				Description: "Remonté par une trentaine de télétravailleurs depuis lundi.",
				Importance:  critique, Dans: dans(-1 * jour),
				Enfants: []tache{
					{Nom: "Analyser les journaux du concentrateur", Importance: haute, Terminee: true},
					{Nom: "Augmenter la licence de sessions simultanées", Importance: haute, Dans: dans(4 * heure)},
					{Nom: "Activer le split tunneling pour Microsoft 365", Importance: normale},
				},
			},
			{Nom: "INC-2305 : imprimante du 3e étage hors ligne", Importance: basse, Terminee: true},
			{Nom: "Renouveler les certificats de l'intranet", Importance: haute, Dans: dans(3 * jour)},
			{
				Nom: "Tester la restauration des sauvegardes", Importance: haute, Dans: dans(8 * jour),
				Enfants: []tache{
					{Nom: "Restaurer la base RH sur l'environnement de test", Importance: normale},
					{Nom: "Mesurer le RTO réel", Importance: normale},
					{Nom: "Mettre à jour le PRA", Importance: normale},
				},
			},
			{Nom: "Nettoyer les comptes Active Directory inactifs", Importance: normale},
			{Nom: "Mettre à jour la CMDB", Importance: basse},
			{
				Nom: "Mettre à jour le firmware des switches", Importance: normale, Dans: dans(18 * jour),
				Enfants: []tache{
					{Nom: "Lire les notes de version", Importance: basse, Terminee: true},
					{Nom: "Planifier la coupure avec le CAB", Importance: normale},
				},
			},
			{Nom: "Migrer la messagerie vers Exchange 2019", Importance: normale, Annulee: true,
				Description: "Remplacé par le passage à Exchange Online."},
		},
		Notes: []note{
			{
				Titre:   "Procédure d'astreinte",
				Contenu: `<ol><li>Accuser réception de l'alerte dans les 15 minutes</li><li>Qualifier : incident ou fausse alerte</li><li>Si production impactée, prévenir le responsable d'astreinte</li><li>Tracer toute intervention dans le ticket</li></ol>`,
			},
			{
				Titre:   "Redémarrage de l'ERP",
				Contenu: `<p>Ordre impératif : base de données, puis serveur d'application, puis serveur web. L'inverse laisse des sessions orphelines qu'il faut purger à la main.</p>`,
			},
			{
				Titre:   "Accès des fournisseurs",
				Contenu: `<p>Tout accès fournisseur passe par le bastion, sur demande tracée et pour une durée limitée. Aucun compte fournisseur permanent.</p>`,
			},
			{
				Titre: "Ancienne procédure VPN PPTP", Cachee: true,
				Contenu: `<p>Obsolète depuis le passage à WireGuard. Conservée pour mémoire.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Revue hebdomadaire des incidents", Depuis: 1 * jour,
				Instances: []string{
					`<p>INC-2291 (VPN) toujours ouvert, cause identifiée : saturation des licences. 42 tickets fermés cette semaine.</p>`,
					`<p>Semaine calme. Deux incidents mineurs sur l'impression.</p>`,
					`<p>Panne de climatisation en salle serveurs samedi, aucun impact grâce à la redondance.</p>`,
					`<p>Revue du backlog : 17 tickets de plus de 30 jours, à trier.</p>`,
				},
			},
			{
				Titre: "Comité des changements (CAB)", Depuis: 5 * jour,
				Instances: []string{
					`<p>Approuvé : correctif OpenSSL en urgence. Reporté : firmware des switches, faute de plan de retour arrière.</p>`,
					`<p>Approuvé : renouvellement des certificats de l'intranet.</p>`,
					`<p>Approuvé : montée de version de l'ERP le week-end du 14.</p>`,
				},
			},
			{
				Titre: "Point avec l'ancien infogérant", Cachee: true, Depuis: 150 * jour, Periode: 30 * jour,
				Instances: []string{
					`<p>Fin du contrat. Restitution des accès et de la documentation.</p>`,
					`<p>Revue des pénalités de fin de contrat.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-mobile", Nom: "Application mobile v2", Age: 60 * jour,
		Taches: []tache{
			{Nom: "Corriger le plantage au démarrage sous Android 15", Importance: critique, Dans: dans(1 * heure),
				Description: "1,2 % des sessions touchées depuis la mise à jour de lundi."},
			{
				Nom: "Authentification biométrique", Importance: normale, Dans: dans(9 * jour),
				Enfants: []tache{
					{Nom: "Face ID sur iOS", Importance: normale, Terminee: true},
					{Nom: "Empreinte digitale sur Android", Importance: normale},
					{Nom: "Repli sur code PIN", Importance: haute},
				},
			},
			{
				Nom: "Mode hors ligne", Importance: haute,
				Enfants: []tache{
					{Nom: "Choisir la base locale", Importance: normale, Terminee: true},
					{
						Nom: "Synchronisation différentielle", Importance: haute,
						Enfants: []tache{
							{Nom: "Horodater les modifications côté serveur", Importance: haute},
							{Nom: "Gérer les conflits d'édition", Importance: haute},
						},
					},
				},
			},
			{Nom: "Passer à la nouvelle architecture React Native", Importance: normale},
			{
				Nom: "Préparer la publication sur les stores", Importance: normale, Dans: dans(21 * jour),
				Enfants: []tache{
					{Nom: "Captures d'écran", Importance: basse},
					{Nom: "Mettre à jour la politique de confidentialité", Importance: normale},
				},
			},
			{Nom: "Version tablette", Importance: basse, Annulee: true,
				Description: "Moins de 3 % d'usage tablette sur la v1."},
		},
		Notes: []note{
			{
				Titre:   "Retours de la bêta",
				Contenu: `<ul><li>Le parcours de connexion est jugé trop long (4 écrans)</li><li>Mode sombre très demandé</li><li>Plantages remontés sur Pixel 8 sous Android 15</li></ul>`,
			},
			{
				Titre:   "Charte UX mobile",
				Contenu: `<p>Zones tactiles de 44 points minimum. Pas plus de deux niveaux de navigation. Toute action destructive demande confirmation.</p>`,
			},
			{
				Titre: "Maquettes v1 abandonnées", Cachee: true,
				Contenu: `<p>Première série de maquettes à onglets, remplacée par la navigation à tiroir après les tests utilisateurs.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Planification de sprint mobile", Depuis: 2 * jour, Periode: 14 * jour,
				Instances: []string{
					`<p>Priorité absolue au plantage Android 15. Le mode hors ligne glisse au sprint suivant.</p>`,
					`<p>Sprint consacré à la biométrie et au choix de la base locale.</p>`,
					`<p>Premier sprint : socle technique, CI mobile, distribution de la bêta.</p>`,
				},
			},
			{
				Titre: "Démo client", Depuis: 10 * jour,
				Instances: []string{
					`<p>Démo de la bêta au client. Satisfait, demande le mode sombre pour la v2.0.</p>`,
				},
			},
		},
	},

	{
		ID: domain.DiversProjectID, Nom: domain.DiversProjectName, Age: 400 * jour,
		Taches: []tache{
			{
				Nom: "Préparer les entretiens annuels de l'équipe", Importance: haute, Dans: dans(5 * jour),
				Enfants: []tache{
					{Nom: "Envoyer les trames de préparation", Importance: normale, Terminee: true},
					{Nom: "Caler les créneaux", Importance: normale},
					{Nom: "Relire les objectifs de l'an dernier", Importance: normale},
				},
			},
			{Nom: "Préparer le budget IT 2027", Importance: haute, Dans: dans(14 * jour),
				Description: "Licences, renouvellement du parc, formation, prestations."},
			{Nom: "Renouveler les licences JetBrains", Importance: normale, Dans: dans(10 * jour)},
			{Nom: "Commander les portables des nouveaux arrivants", Importance: normale, Terminee: true},
			{Nom: "Lire le panorama de la cybermenace de l'ANSSI", Importance: basse},
			{Nom: "S'inscrire à la conférence DevOps d'automne", Importance: basse, Dans: dans(-6 * jour)},
		},
		Notes: []note{
			{
				Titre:   "Idées en vrac",
				Contenu: `<ul><li>Un wiki unique au lieu de trois</li><li>Journée « dette technique » une fois par trimestre</li><li>Mesurer le temps passé en réunion</li></ul>`,
			},
			{
				Titre:   "Budget prévisionnel 2027",
				Contenu: `<p>Postes principaux : licences Microsoft (+8 %), renouvellement d'un tiers du parc, formation Kubernetes pour l'équipe exploitation.</p>`,
			},
			{
				Titre: "Notes de la formation ITIL 4", Cachee: true,
				Contenu: `<p>Chaîne de valeur des services, sept principes directeurs. Certification obtenue.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Réunion d'équipe DSI", Depuis: 3 * jour,
				Instances: []string{
					`<p>Arrivée de deux développeurs le mois prochain. Point sur la charge de l'équipe exploitation, très sollicitée.</p>`,
					`<p>Présentation des résultats du POC EDR à toute l'équipe.</p>`,
					`<p>Organisation des congés de fin d'année.</p>`,
					`<p>Tour de table des projets. Kubernetes et facturation prennent l'essentiel de la charge.</p>`,
				},
			},
			{
				Titre: "Séminaire DSI 2025", Cachee: true, Depuis: 330 * jour,
				Instances: []string{
					`<p>Deux jours hors les murs. Feuille de route 2026 : conteneurs, observabilité, MFA.</p>`,
				},
			},
		},
	},

	/* ======== Projets terminés, cachés ======== */

	{
		ID: "dev-win11", Nom: "Migration Windows 11", Cache: true, Age: 300 * jour, Inactif: 120 * jour,
		Taches: []tache{
			{
				Nom: "Inventorier le parc", Importance: haute, Terminee: true, Dans: dans(-250 * jour),
				Enfants: []tache{
					{Nom: "Identifier les postes sans TPM 2.0", Importance: haute, Terminee: true},
					{Nom: "Lister les applications métier à tester", Importance: normale, Terminee: true},
				},
			},
			{Nom: "Construire le master", Importance: haute, Terminee: true, Dans: dans(-220 * jour)},
			{
				Nom: "Déployer par vagues", Importance: haute, Terminee: true, Dans: dans(-130 * jour),
				Enfants: []tache{
					{Nom: "Vague 1 : service informatique", Importance: normale, Terminee: true},
					{Nom: "Vague 2 : services support", Importance: normale, Terminee: true},
					{Nom: "Vague 3 : production", Importance: haute, Terminee: true},
				},
			},
			{Nom: "Remplacer les 40 postes incompatibles", Importance: normale, Terminee: true},
			{Nom: "Maintenir un pool Windows 10 sous ESU", Importance: basse, Annulee: true,
				Description: "Inutile : toutes les applications métier sont passées."},
		},
		Notes: []note{
			{
				Titre:   "Bilan de la migration",
				Contenu: `<p>612 postes migrés en 5 mois, 40 remplacés. 3 applications métier ont nécessité une mise à jour éditeur.</p>`,
			},
			{
				Titre: "Applications incompatibles", Cachee: true,
				Contenu: `<p>Ancien client de gestion du courrier, pilote de la badgeuse, macro Excel de la paie. Toutes traitées.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Comité de suivi Windows 11", Depuis: 125 * jour, Periode: 30 * jour,
				Instances: []string{
					`<p>Clôture du projet. Dernière vague terminée sans incident majeur.</p>`,
					`<p>Vague 2 terminée. Retours utilisateurs globalement positifs.</p>`,
					`<p>Validation du master et du calendrier des vagues.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-oracle", Nom: "Décommissionnement Oracle 11g", Cache: true, Age: 400 * jour, Inactif: 200 * jour,
		Taches: []tache{
			{
				Nom: "Migrer les schémas vers PostgreSQL", Importance: critique, Terminee: true, Dans: dans(-230 * jour),
				Enfants: []tache{
					{Nom: "Convertir les procédures PL/SQL", Importance: haute, Terminee: true},
					{Nom: "Reprendre les données historiques", Importance: haute, Terminee: true},
					{Nom: "Comparer les volumétries ligne à ligne", Importance: normale, Terminee: true},
				},
			},
			{Nom: "Rebrancher les applications clientes", Importance: haute, Terminee: true},
			{Nom: "Archiver les sauvegardes Oracle", Importance: normale, Terminee: true},
			{Nom: "Résilier le contrat de support Oracle", Importance: haute, Terminee: true, Dans: dans(-205 * jour)},
			{Nom: "Garder une instance en lecture seule", Importance: basse, Annulee: true},
		},
		Notes: []note{
			{
				Titre:   "Correspondance des types Oracle → PostgreSQL",
				Contenu: `<ul><li><code>NUMBER(p,s)</code> → <code>numeric(p,s)</code></li><li><code>VARCHAR2</code> → <code>varchar</code></li><li><code>DATE</code> → <code>timestamp(0)</code>, attention : le DATE Oracle porte l'heure</li></ul>`,
			},
			{
				Titre:   "Économies réalisées",
				Contenu: `<p>Support Oracle : 48 k€ par an. Deux serveurs physiques décommissionnés.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Point migration base de données", Depuis: 210 * jour, Periode: 14 * jour,
				Instances: []string{
					`<p>Bascule de production réussie. Oracle arrêté, conservé éteint un mois par précaution.</p>`,
					`<p>Répétition générale de la bascule : 3 h 40, sous la fenêtre de 6 h.</p>`,
				},
			},
			{
				Titre: "Négociation de sortie avec Oracle", Cachee: true, Depuis: 240 * jour,
				Instances: []string{
					`<p>Résiliation acceptée à l'échéance, sans pénalité.</p>`,
				},
			},
		},
	},

	{
		ID: "dev-dc", Nom: "Déménagement de la salle serveurs", Cache: true, Age: 250 * jour, Inactif: 60 * jour,
		Taches: []tache{
			{Nom: "Signer le contrat d'hébergement", Importance: critique, Terminee: true},
			{
				Nom: "Préparer le déménagement", Importance: haute, Terminee: true, Dans: dans(-90 * jour),
				Enfants: []tache{
					{Nom: "Étiqueter les câbles", Importance: normale, Terminee: true},
					{Nom: "Réserver le transporteur spécialisé", Importance: haute, Terminee: true},
					{Nom: "Vérifier les assurances", Importance: normale, Terminee: true},
				},
			},
			{Nom: "Déménager et redémarrer les baies", Importance: critique, Terminee: true, Dans: dans(-75 * jour)},
			// Tâche oubliée dans un projet caché : le masquage est purement
			// visuel, elle doit rester visible dans Priorités et la barre système.
			{Nom: "Récupérer les baies vides chez l'hébergeur", Importance: normale, Dans: dans(-10 * jour),
				Description: "Oubliée à la clôture du projet. Le propriétaire des locaux les réclame."},
			{Nom: "Garder l'ancienne salle en secours", Importance: basse, Annulee: true},
		},
		Notes: []note{
			{
				Titre:   "Plan d'implantation des baies",
				Contenu: `<p>Rangée B, baies 12 à 15. Alimentation redondée A+B, 6 kVA par baie.</p>`,
			},
			{
				Titre: "Devis transporteurs", Cachee: true,
				Contenu: `<p>Trois devis reçus, le moins cher n'assurait pas le matériel en transit. Retenu : le deuxième.</p>`,
			},
		},
		Reunions: []reunion{
			{
				Titre: "Comité de pilotage déménagement", Depuis: 65 * jour, Periode: 21 * jour,
				Instances: []string{
					`<p>Clôture. Tous les services ont redémarré en moins de 4 h.</p>`,
					`<p>Validation du plan de déménagement et du plan de retour arrière.</p>`,
					`<p>Choix de l'hébergeur.</p>`,
				},
			},
		},
	},
}
