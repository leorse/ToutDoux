# 📋 SPÉCIFICATION COMPLÈTE - Tout Doux (v2)

> Cette version remplace `SPEC_COMPLETE_TOUTDOUX.md`. Elle intègre tout ce qui a été
> décidé et affiné pendant la phase de prototypage React (validation UX avant code
> app Wails réelle). Le prototype React n'est pas jeté : ses fonctions de logique pure
> (échéances, filtres, cascade, recherche) servent de référence directe pour le
> portage du domaine en Go — voir section 3.9.
>
> **Choix de stack final (décidé après le prototypage)** : Wails (Go) plutôt que
> Tauri (Rust). Raison pratique : l'environnement de développement ne dispose pas
> des droits admin nécessaires à l'installation de la toolchain Rust + MSVC Build
> Tools. Wails ne demande que Go (installable en portable) et un compilateur C via
> MSYS2/mingw-w64 (déjà présent), sans droits admin. Même philosophie que Tauri :
> webview système (pas de Chromium embarqué), exécutable léger. Voir section 3.5.

## CHANGELOG PAR RAPPORT À LA V1

**Navigation générale (nouveau)**
- L'app a maintenant deux vues de premier niveau : **Projets** et **Priorités**, plus une **barre de recherche** — les trois dans une seule bande transverse sous le header
- L'onglet **Priorités quitte le groupe Tâches/Notes/Réunions** : ce n'est pas un onglet dépendant d'un projet, c'est une vue transverse à part entière (section 2.8 réécrite)
- La **Recherche** n'a plus de mode "Projet" : elle est toujours globale (section 2.9 réécrite)
- Toutes les séparations entre panneaux (sidebar↔contenu, liste↔détail, colonnes de la vue Réunions) sont **redimensionnables au glisser-déposer**, de façon cohérente sur toute l'app
- La barre de titre "Tout Doux" affichée en haut du prototype web **ne doit pas être reprise dans l'app Wails** : elle sera remplacée par la barre de fenêtre native (icône déjà prête)

**Projets**
- Le projet "Transverse / Divers" reste **toujours en première position** dans la sidebar, quel que soit l'ordre de création des autres projets
- Il s'affiche en **gris fixe**, sans logique de couleur dynamique, et **sans icône cadenas** (le style grisé suffit à indiquer qu'il est spécial)
- Les couleurs des projets (rouge/orange si tâche Critique/Haute active) utilisent **exactement les mêmes teintes que les tâches** — plus de version "pastel" séparée
- Le compteur sidebar affiche uniquement le **nombre de tâches actives** (pas le triplet Total/Complétées/Actives) + petit texte gris avec nombre de notes et de réunions

**Tâches — interactions**
- Ajout de tâche/sous-tâche, suppression, et déplacement (monter/descendre/début/fin) : **exclusivement via le menu contextuel (clic droit)**, plus aucun bouton de toolbar
- Menu contextuel enrichi : **"Tout étendre" / "Tout réduire"** disponibles depuis le clic droit sur le fond ou sur une tâche
- Clic droit sur le fond de la colonne Instances (vue Réunions) sans réunion sélectionnée → **ne fait rien** (pas de menu)

**Tâches — cohérence des statuts (corrections de fond)**
- Ajouter une sous-tâche à un parent terminé/annulé **réactive automatiquement toute la chaîne d'ancêtres**, jusqu'au premier ancêtre déjà actif
- Même règle appliquée au **drag & drop** : déplacer une tâche active sous un parent terminé/annulé le réactive en cascade
- Décocher un enfant terminé fait aussi perdre le statut **annulé** du parent (pas seulement "terminé"), s'il ne peut plus être considéré comme fini

**Tâches — dépliement par défaut (nouveau comportement)**
- Au chargement et à chaque changement de projet : toute **sous-arborescence contenant au moins une tâche active** est dépliée par défaut — pas seulement les tâches elles-mêmes actives. Un parent terminé avec un enfant encore actif reste donc visible.
- Ce recalcul ne se déclenche **qu'au changement de projet**, jamais en continu pendant que l'utilisateur travaille (pour ne pas replier une tâche sous ses yeux)

**Filtres tâches — redesign complet**
- Le filtre **Statut devient multi-sélection** (Actives + Annulées en même temps, par exemple), comme l'était déjà le filtre Importance
- Représentation **icônes/pastilles plutôt que texte**, pour gagner de la place :
  - Statut : case vide (Active), case cochée verte (Complétée), case + texte barré "abc" (Annulée)
  - Importance : pastille colorée (rouge/orange/blanc/gris — mêmes couleurs que les tâches), contour noir dont l'**épaisseur** indique l'état actif/inactif (pas de changement de couleur de fond)
  - Échéance uniquement : icône horloge dans un cercle, même logique de contour
- Tous les filtres tiennent sur **une seule ligne**
- **Les filtres ne sont plus réinitialisés au changement de projet** — ils persistent tels quels (contrairement à la v1)

**Détail d'une tâche**
- Boutons d'importance : texte conservé, mais **fond coloré selon la criticité**, sélection unique, contour épais si actif
- **Nouveau : sélecteur de date/heure libre** (`datetime-local`), en complément des raccourcis rapides existants — pour fixer une échéance éloignée sans passer par les raccourcis
- Retirer une échéance : **croix rouge (icône)**, plus de lien texte "Retirer"
- Icône ⏰ affichée si urgent (< 5 min) **OU en retard** (la v1 ne l'affichait que pour "urgent", ce qui la faisait disparaître dès qu'une tâche passait en retard)

**Notes / Réunions — fiabilité de la sauvegarde auto (correction de fond)**
- En plus du délai habituel (2s notes/instances, 500ms tâches), la sauvegarde se déclenche **immédiatement** si : le champ perd le focus, l'utilisateur change de sélection (autre note/tâche/instance), ou change de projet. Objectif : zéro perte de frappe, y compris si l'utilisateur clique ailleurs avant la fin du délai.

**Priorités — repositionnée et redesignée (section 2.8 réécrite)**
- N'est plus un onglet sous un projet : devient une **vue de premier niveau**, pleine largeur, sans sidebar projets
- Toujours composée des deux mêmes sections (Critiques & Hautes, Échéances à venir), mais affichage en **pastille colorée** plutôt qu'en texte "[Crit]"/"[Haute]"
- **Nouveau : split interne liste (gauche) / détail en lecture seule (droite)**
  - Simple clic sur une tâche → aperçu en lecture seule à droite (même structure visuelle que le panneau de détail Tâches, mais non éditable)
  - Double-clic → bascule sur la vue **Projets**, sélectionne le bon projet, déplie les ancêtres nécessaires, sélectionne la tâche : comportement d'édition classique

**Recherche — repositionnée et simplifiée (section 2.9 réécrite)**
- Le **mode Projet est retiré** : la recherche est toujours globale (tous projets)
- La barre est fusionnée dans la **même bande** que les vues Projets/Priorités, pas empilée séparément au-dessus des onglets
- La vue résultats remplace tout le contenu principal, **quelle que soit la vue active** (Projets ou Priorités) au moment de la recherche
- Icônes par type : tâche = **vraie case à cocher** (reflète l'état complété) **+ pastille colorée d'importance**, au lieu du simple symbole "✓" de la v1 ; note = 📝 ; réunion/instance = 📞
- Simple clic = aperçu à droite ; double-clic (ou bouton "Ouvrir") = ferme la recherche et navigue vers l'élément dans son onglet natif (bascule automatiquement sur la vue Projets)

**Recherche sémantique (nouveau — section 2.12)**
- Second mode de recherche, **strictement séparé** de la recherche mot-clé : l'utilisateur choisit lequel il emploie, les deux ne sont jamais fusionnés
- Bascule par une **icône 🧠 seule** dans la barre de recherche, décochée par défaut
- **Opt-in par entité** : rien n'est vectorisé automatiquement. Un bouton « Ajouter à la recherche sémantique » sur les notes, les instances de réunion et les tâches
- Une fois activée sur une entité, la vectorisation **se maintient seule** : chaque modification du texte la recalcule en arrière-plan, au rythme de la sauvegarde auto existante
- **Zéro réseau, sans exception** : l'application ne télécharge rien et ne vérifie aucune mise à jour. Le modèle est déposé à la main par l'utilisateur

**Préférences (nouveau — section 2.13)**
- Nouvelle vue de premier niveau, icône ⚙️ : statut du modèle sémantique, chemin et noms de fichiers attendus, lien de téléchargement, bouton de vérification

**Architecture (nouveau — section 3.9)**
- Décision d'architecture hexagonale **appliquée avec discernement**, pas systématique : voir section 3.9 pour le détail de ce qui est mis derrière un port et ce qui ne l'est pas, et pourquoi
- Un port `EmbeddingProvider` s'ajoute en même temps que la recherche sémantique (section 2.12), pour la raison habituelle : sans lui, tester le domaine imposerait de charger un modèle de 120 Mo à chaque `go test`

---

## 1. VISION PRODUIT

**Tout Doux** est un gestionnaire de projets, tâches, notes et réunions desktop moderne, léger et rapide.

Une application pour organiser votre travail sans friction :
- Projets structurés
- Tâches hiérarchiques avec échéances intelligentes
- Notes riches par projet
- Suivi des réunions avec historique
- Recherche globale dans tous les contenus
- Vue transverse des priorités, indépendante des projets
- Intégration taskbar/tray pour rester en arrière-plan

**Cible :** Travailleurs individuels et petites équipes (1-10 personnes).

---

## 2. SPÉCIFICATION FONCTIONNELLE

### 2.1 GESTION DES PROJETS

**Créer projet**
- Nom unique, texte libre
- Validation d'unicité insensible à la casse
- Visible immédiatement dans la sidebar

**Renommer projet**
- Double-clic sur le nom
- Validation unicité du nom (insensible à la casse)

**Supprimer projet**
- Confirmation avec compte des tâches/notes/réunions (et leurs instances) qui seront supprimées
- Cascade complète : tâches, notes, réunions, instances de réunion

**Cas spécial : Projet "Transverse/Divers"**
- Créé automatiquement au démarrage, id stable
- **Verrouillé** : ne peut pas être renommé ni supprimé (pas de bouton de suppression au survol, double-clic sans effet)
- **Toujours affiché en premier** dans la sidebar, quel que soit l'ordre de création des autres projets
- **Affiché en gris fixe** (mêmes tons que le statut "Annulée" d'une tâche), sans logique de couleur dynamique liée à la criticité de ses tâches, et **sans icône cadenas**
- Espace de repli pour les tâches sans projet dédié
- Si le projet actif est supprimé (cas d'un projet non verrouillé), l'app bascule automatiquement sur Divers

**Affichage sidebar**
```
[Nom projet]              [N actives] [icône échéance]
[N notes · N réunions]
```
- Le compteur principal n'affiche que le **nombre de tâches actives** (pas le triplet Total/Complétées/Actives de la v1 — jugé inutilement verbeux)
- Une ligne secondaire, petite et grise, indique le nombre de notes et de réunions du projet

**Visuels projet**
- Fond **rouge** (même teinte que le fond "Critique" des tâches) si une tâche Critique active existe dans le projet
- Fond **orange** (même teinte que "Haute") si une tâche Haute active existe et aucune Critique
- Sinon fond blanc
- Icône ⏰ si une tâche active est urgente (< 5min) **ou en retard**
- Icône 🕐 si une tâche active a une échéance à venir (non urgente)
- Projet sélectionné : contour vert foncé

---

### 2.2 GESTION DES TÂCHES

**Structure**
- Hiérarchie infinie (parent → enfants → petits-enfants, etc.)
- Chaque tâche a : nom, description, importance, échéance, statut (actif/complété/annulé), projet, ordre au sein de ses frères

**Créer tâche**
- Exclusivement via **clic droit** :
  - Clic droit sur le **fond** de la zone de tâches (pas sur une tâche) → menu avec "+ Nouvelle tâche" (créée au niveau racine du projet actif)
  - Clic droit **sur une tâche** → menu avec "+ Sous-tâche" (créée comme enfant de cette tâche)
- Saisie du nom via une petite modale (nom uniquement — importance/échéance se règlent ensuite dans le détail)

**Éditer tâche**
- Nom (texte libre)
- Description (texte riche — voir 3.4 pour la cible finale, texte simple acceptable en transition)
- Importance : Basse / Normale / Haute / Critique — boutons colorés selon la criticité, sélection unique
- Échéance : optionnelle, via raccourcis rapides **ou sélecteur date/heure libre** (voir 2.3)

**Supprimer tâche**
- Uniquement via le menu contextuel (clic droit → "Supprimer")
- Confirmation affichant le nombre de sous-tâches qui seront supprimées en cascade

**Marquer terminée**
- Checkbox dans l'arbre
- Auto-update parent : si TOUS les enfants (directs) sont terminés, le parent devient terminé automatiquement, en cascade jusqu'à la racine
- Si un enfant repasse actif (décoché), le ou les parents concernés perdent leur statut terminé **et leur statut annulé** en cascade (un parent ne peut pas rester marqué "fini" au sens large s'il a un enfant actif)

**Annuler/Réactiver tâche**
- État "cancelled", distinct de "completed"
- Visuellement : barré, gris
- Disponible dans le détail de la tâche et dans le menu contextuel

**Règle de cohérence : réactivation en cascade des ancêtres**

C'est une règle de cohérence de données à respecter impérativement, identifiée pendant le prototypage :

> Une sous-tâche fraîchement ajoutée, ou déplacée sous un nouveau parent, est **active par nature**. Ses ancêtres ne peuvent donc pas rester marqués "terminé" ou "annulé".

- **Ajout d'une sous-tâche** à un parent terminé/annulé → ce parent (et toute la chaîne au-dessus, tant qu'elle est elle-même terminée/annulée) redevient actif automatiquement. La remontée s'arrête dès qu'un ancêtre est déjà actif.
- **Drag & drop** : déplacer une tâche **active** sous un parent terminé/annulé déclenche la même cascade de réactivation. (Déplacer une tâche déjà terminée/annulée ne déclenche rien, la cohérence du parent reste correcte.)

**Déplacer tâche**
- Drag & drop : reparenter (changer de parent), avec la cascade de réactivation ci-dessus si applicable
- Menu contextuel de la tâche : "Envoyer au début" / "Monter" / "Descendre" / "Envoyer à la fin" (au sein des frères du même parent) — **plus de boutons dédiés dans le panneau de détail**, uniquement accessible via clic droit
- Validation anti-cycle : impossible de déplacer une tâche dans un de ses propres descendants (message d'erreur explicite, pas d'échec silencieux)

**Tout étendre / Tout réduire**
- Disponible dans le menu contextuel, à la fois depuis le clic droit sur le fond et depuis le clic droit sur une tâche
- Agit sur l'ensemble des tâches du projet actuellement affiché

**Dépliement par défaut**
- Au chargement de l'app et à **chaque changement de projet** : toute tâche dont la sous-arborescence contient au moins une tâche active (elle-même active, ou un de ses descendants l'est) est dépliée par défaut
- Une tâche terminée ou annulée dont TOUS les descendants sont eux-mêmes terminés/annulés démarre repliée
- Ce calcul n'est **pas** recalculé en continu pendant que l'utilisateur travaille (par exemple en cochant une tâche) — seulement au chargement et au changement de projet, pour ne jamais replier une tâche sous les yeux de l'utilisateur en cours de manipulation

**Affichage arbre**
```
├─ 🕐 Tâche 1 (En retard de 2j)
│  ├─ Sous-tâche 1.1 ✓ (complétée, vert)
│  └─ Sous-tâche 1.2 (Critique, rouge bg)
│     └─ Sous-sous-tâche 1.2.1 (Haute, orange)
├─ Tâche 2 (Normal, blanc)
│  └─ Sous-tâche 2.1 (Cancelled, gris barré)
└─ Tâche 3 (Basse, gris clair)
```

**Visuels tâche**
- Fond rouge (#ff4d4d) si Critique
- Fond orange (#ffb84d) si Haute
- Fond blanc si Normale
- Fond gris clair (#d9d9d9) si Basse
- Fond vert clair si Complétée
- Gris + barré si Annulée
- Icône ⏰ si échéance < 5min **ou en retard**
- Icône 🕐 si échéance à venir (non urgente, non en retard)
- Texte : "Nom tâche (dans 2h)" ou "Nom tâche (En retard de 3j)"

---

### 2.3 ÉCHÉANCES INTELLIGENTES

**Ajouter échéance**
- Boutons rapides :
  - "Tout de suite" → maintenant
  - "Dans 10 min" → +10 minutes
  - "Dans 1h" → +1 heure
  - "Journée" → fin de journée (18h) ; désactivé après 17h (préférer "Lendemain 9h")
  - "Lendemain 9h" → J+1 à 9h ; si vendredi → lundi 9h automatiquement
- **Sélecteur de date/heure libre** (`datetime-local`), affiché en complément des raccourcis, pour fixer une échéance éloignée que les raccourcis ne couvrent pas

**Retirer une échéance**
- Croix rouge (icône ronde), à côté de l'échéance affichée — pas de lien texte

**Affichage temps restant**
```
- Si aujourd'hui :
  - < 1h → "dans 23 min"
  - 1-2h → "~1h" ou "~1h30"
  - > 2h → "~4h"

- Si demain → "Demain"
- Si cette semaine → "dans 3 jours"
- Si semaine prochaine → "Semaine prochaine"
- Si au-delà → "2 semaines et 3 jours"
- Si dépassée → "En retard de 2h" / "En retard de 3j"
```

**Icônes d'urgence**
- ⏰ si échéance < 5 min **ou déjà en retard** (quelle que soit l'ampleur du retard)
- 🕐 si échéance à venir, non urgente

**Refresh temps réel**
- Timer toutes les 30s met à jour l'affichage du temps restant partout dans l'app (arbre, sidebar, priorités)
- Taskbar notifiée si une tâche devient urgente (< 5min) — voir 2.10

---

### 2.4 FILTRAGE TÂCHES

**Filtres combinables, tous sur une seule ligne**

- **Statut** — multi-sélection (et non plus radio exclusif) : Active / Complétée / Annulée, combinables librement (ex. Actives + Annulées visibles ensemble)
  - Représentation : icônes plutôt que texte
    - Active : case à cocher vide
    - Complétée : case cochée, fond vert
    - Annulée : case vide + texte d'exemple barré ("abc"), pour évoquer visuellement le style barré d'une tâche annulée
  - Au moins un statut doit rester sélectionné (le dernier décoché se re-coche automatiquement, pour éviter un arbre vide sans explication)
- **Importance** — multi-sélection : Critique / Haute / Normale / Basse
  - Représentation : pastille colorée (rouge/orange/blanc/gris, mêmes teintes que les tâches), contour noir constant, **épaisseur du contour** = actif/inactif (pas de changement de couleur de fond)
- **Échéance uniquement** — toggle : icône horloge dans un cercle, même logique d'épaisseur de contour

**Comportement filtrage**
- Les tâches ne matchant aucun filtre actif sont cachées
- **Important** : si une tâche ne matche pas mais qu'un de ses descendants matche → la tâche parente reste visible (règle inchangée depuis la v1)
- Clic sur une tâche filtrée (masquée) → panneau de détail vide

**Persistance**
- Les filtres sont **conservés tels quels** d'un projet à l'autre — **changement par rapport à la v1**, qui les réinitialisait par défaut à chaque changement de projet. Décision utilisateur explicite : la préférence de filtrage est jugée plus stable que le projet affiché.
- Seule exception : la navigation automatique depuis Priorités ou la Recherche peut compléter (jamais remplacer) le filtre courant pour garantir que l'élément ciblé reste visible (ajout du statut "active" et de l'importance de la tâche visée si nécessaire, sans toucher au reste des préférences).

---

### 2.5 ONGLET TÂCHES (au sein de la vue Projets)

**Layout**
```
┌──────────────────────────────────────────────────────────┐
│ [Statut●●●] │ [Importance ●●●●] │ [🕐]                   │ ← Filtres, une ligne, icônes
├────────────────────────────┬─────────────────────────────┤
│ [Arbre tâches hiérarchique]│ [Detail panel : nom /       │
│                            │  description / importance / │
│                            │  échéance]                  │
│                            │                             │
└────────────────────────────┴─────────────────────────────┘
                             ↑ séparation VERTICALE, redimensionnable
```
- Plus aucun bouton de toolbar ("+Tâche", "+Sous-tâche", "Supprimer") — tout passe par le clic droit
- **L'arbre et le détail sont côte à côte**, séparés par une barre verticale redimensionnable au glisser-déposer — et non empilés l'un au-dessus de l'autre
  - *Correction du 2026-08-22* : le schéma de cette section montrait auparavant les deux panneaux empilés, ce qui était une erreur de la spec. L'agencement côte à côte est celui retenu, cohérent avec les autres onglets (Notes en 2.6, Réunions en 2.7) et avec la vue Priorités (2.8), qui placent tous le détail à droite de la liste.

---

### 2.6 ONGLET NOTES (au sein de la vue Projets)

**Structure**
- Notes associées à un projet
- Une note = titre + contenu riche
- Création (clic droit sur le fond de la liste) / suppression / renommage (double-clic ou clic droit) via menu contextuel
- Sauvegarde automatique (2s d'inactivité)

**Fiabilité de la sauvegarde (précision v2)**
En plus du délai de 2s, la sauvegarde se déclenche **immédiatement**, sans attendre le délai, dans ces cas :
- Perte de focus du champ titre ou contenu
- Changement de note sélectionnée
- Changement de projet

Objectif : aucune modification en cours ne doit pouvoir être perdue, quelle que soit la façon dont l'utilisateur quitte l'édition.

**Statut sauvegarde**
- Texte : "Modifications en cours..."
- Puis : "Note sauvegardée à 14:32:15"

**Layout**
```
│ [Liste notes] │ [Éditeur riche]         │
│               │                          │
│ • Note 1      │ Titre: Note 1            │
│ • Note 2      │ [Contenu...]             │
│ • Note 3      │ Sauvegardée à 14:32:15   │
```
- Plus de boutons [+] [-] [Renommer] dédiés — clic droit sur le fond (créer) ou sur une note (renommer/supprimer)
- Séparation liste ↔ éditeur redimensionnable

---

### 2.7 ONGLET RÉUNIONS (au sein de la vue Projets)

**Structure**
- Réunions associées à un projet (pas globales)
- Une réunion = titre + instances (historique)
- Une instance = timestamp + notes riches

**Créer réunion**
- Clic droit sur le fond de la colonne Réunions → "+ Nouvelle réunion" → modale titre libre

**Ajouter instance**
- Clic droit sur le fond de la colonne Instances → "+ Nouvelle instance"
- **Uniquement disponible si une réunion est sélectionnée** : si aucune réunion n'est sélectionnée, le clic droit sur cette colonne ne fait rien (pas de menu affiché) — précision v2
- Crée automatiquement timestamp (maintenant) + notes vides, sélectionne la nouvelle instance

**Sélection automatique**
- Sélectionner une réunion sélectionne automatiquement sa dernière instance (la plus récente), pour éviter un clic supplémentaire

**Éditer notes instance**
- Éditeur riche (même mécanique que les notes)
- Sauvegarde auto (2s), avec la même fiabilité de flush immédiat qu'en 2.6 (blur, changement d'instance/réunion/projet)

**Layout**
```
│ [Réunions] │ [Instances] │ [Éditeur]     │
│            │             │               │
│ • Réunion 1│ • 23/02 14:30 │ [Notes]     │
│ • Réunion 2│ • 20/02 09:00 │ Sauv. 14:32 │
```
- Trois colonnes, deux séparations redimensionnables
- Plus de boutons [+] [-] dédiés — clic droit uniquement

**Instances triées**
- Par date décroissante (plus récente d'abord)

---

### 2.8 VUE PRIORITÉS (vue de premier niveau, transverse)

**Changement majeur par rapport à la v1** : Priorités n'est plus un onglet sous un projet. C'est une **vue de premier niveau**, au même titre que la vue Projets, sélectionnable depuis la bande de navigation transverse (voir 2.11). Elle n'a pas de sens rattachée à un seul projet puisqu'elle balaie tous les projets par nature.

**Contenu** (inchangé dans le fond)
- Section "Tâches Critiques & Hautes" : toutes les tâches actives (non complétées, non annulées) d'importance Critique ou Haute, tous projets confondus
- Section "Échéances à venir" : toutes les tâches actives avec échéance, tous projets confondus, triées par urgence (en retard/imminentes en premier)

**Layout — pleine largeur, split interne**
```
┌────────────────────────────────────────────────┐
│ Tâches Critiques & Hautes (4)                  │
│ ┌──────────────────────────┐ ┌────────────────┐│
│ │ ● Task 1  [MIG-2026] [..]│ │  DÉTAIL         ││
│ │ ● Task 2  [US-3456]  [..]│ │  (lecture seule)││
│ └──────────────────────────┘ │  Nom, importance││
│ Échéances à venir (8)         │  échéance,      ││
│ ┌──────────────────────────┐ │  description    ││
│ │ ● Task A  [DR]  [30 min] │ │  [Ouvrir dans   ││
│ │ ● Task B  [MIG] [Demain] │ │   Tâches]       ││
│ └──────────────────────────┘ └────────────────┘│
└────────────────────────────────────────────────┘
```
- **Pas de sidebar projets** dans cette vue (transverse par nature)
- Chaque ligne : **pastille colorée d'importance** (pas de texte "[Crit]"/"[Haute]"), nom, tag projet, échéance
- Colonne gauche (liste) / colonne droite (détail), séparation redimensionnable

**Interaction**
- **Simple clic** sur une tâche → la sélectionne, affiche son détail en **lecture seule** à droite (même structure visuelle que le panneau de détail de l'onglet Tâches — nom, importance, échéance, description — mais aucun champ éditable), avec un bouton "Ouvrir dans Tâches"
- **Double-clic** (ou bouton "Ouvrir") → ferme la vue Priorités, bascule sur la vue **Projets**, sélectionne le bon projet, déplie tous les ancêtres nécessaires dans l'arbre, sélectionne la tâche : comportement d'édition classique, comme si l'utilisateur l'avait ouverte lui-même

---

### 2.9 RECHERCHE GLOBALE

**Changements majeurs par rapport à la v1**
- **Toujours globale** : le mode "Projet" est retiré, il n'y a plus de choix Global/Projet
- **Positionnement** : la barre de recherche est dans la **même bande transverse** que les vues Projets/Priorités (voir 2.11), pas empilée séparément au-dessus des onglets

**Barre de recherche**
- Icône loupe + input + croix pour effacer (visible seulement si texte tapé)
- Placeholder explicite du périmètre ("Rechercher dans les tâches, notes, réunions…")
- **Bascule de mode** : icône 🧠 seule, à droite du champ, sans libellé — cohérent avec le reste de l'interface, où les filtres du §2.4 sont déjà en icônes. Décochée par défaut : la recherche mot-clé reste le mode normal
  - Décochée → recherche mot-clé, décrite dans cette section
  - Cochée → recherche sémantique, décrite en §2.12
- Les deux modes ne sont **jamais fusionnés ni mélangés dans une même liste de résultats**. Ce sont deux façons différentes de chercher, avec des forces différentes ; les entrelacer produirait un classement que personne ne saurait interpréter

**Déclenchement**
- Typing > 2 caractères (à partir du 3ᵉ caractère)
- Debounce 300ms
- Résultats recalculés en temps réel

**Vue résultats** (remplace tout le contenu principal, quelle que soit la vue active — Projets ou Priorités — au moment de la recherche)
```
┌──────────────────────────────────────┐
│ [5 résultats]                 [Fermer ⊗]
├─────────────────┬────────────────────┤
│ RÉSULTATS       │ DÉTAIL             │
├─────────────────┼────────────────────┤
│ ☑ ● Task 3      │ Title: "Task 3"    │
│ 📝 Note 1       │ "...<<mot>>..."    │
│ 📞 Meeting 2    │ (jaune highlight)  │
│ [Scroll...]     │ Projet: MIG-2026   │
│                 │ [Ouvrir]           │
└─────────────────┴────────────────────┘
```
- Résultats triés par date décroissante (plus récent en premier ; sans date en dernier)

**Icônes par type**
- Tâche : **case à cocher réelle** reflétant l'état complété/actif de la tâche **+ pastille colorée d'importance** (remplace le simple "✓" de la v1, jugé peu informatif)
- 📝 Note
- 📞 Réunion / instance

**Surbrillance**
- Mot-clé cherché en **jaune** dans l'extrait affiché en détail

**Interaction**
- Simple clic sur un résultat → aperçu en détail à droite
- Double-clic (ou bouton "Ouvrir") → ferme la recherche, navigue vers l'élément dans son onglet natif, et **bascule automatiquement sur la vue Projets** (puisque tout ce que la recherche peut ouvrir — tâche, note, réunion — vit dans cette vue)

**Fermeture**
- Bouton [Fermer ⊗]
- Touche Échap
- Effacement complet du champ de recherche
- Revient à la vue précédemment affichée (Projets ou Priorités)

---

### 2.10 BARRE SYSTÈME (Windows Taskbar)

*(Section inchangée depuis la v1 — hors-scope du prototype web, nécessite l'intégration native Wails)*

**Icônes**
- Normal : icône standard app
- Haute urgence : icône "haute priorité" (orange)
- Urgence critique : icône "critique" (rouge)
- Clignotement si tâche < 5min

**Menu contextuel (clic droit)**
```
Top 5 tâches prioritaires (cliquables)
├─ [Projet] Tâche 1 (dans 30 min)
├─ [Projet] Tâche 2 (Demain)
├─ [Projet] Tâche 3 (Semaine prochaine)
├─ [Projet] Tâche 4 (En retard)
└─ [Projet] Tâche 5 (Critique)

Tâches Critiques (4)
Tâches Hautes (8)
─────────
Afficher
Quitter
```
- Le contenu de ce menu (top 5, critiques, hautes) peut réutiliser directement la même logique que la vue Priorités (2.8) côté domaine.

**Click sur tâche dans le menu**
- Affiche l'app, sélectionne le projet, ouvre la tâche dans l'onglet Tâches — même comportement que le double-clic en 2.8/2.9

**Double-clic icône taskbar**
- Affiche/masque l'app

**Notification**
- Si une tâche devient urgente (< 5min) : titre "Tâche Urgente!", message "[Projet] Tâche (dans 2 min)", son système

**Comportement du bouton "X" de la fenêtre**
- Ne ferme PAS l'app : cache la fenêtre, l'app continue en arrière-plan
- Notification : "App réduite en arrière-plan"

---

### 2.11 NAVIGATION GÉNÉRALE ET LAYOUT (nouvelle section)

**Bande transverse unique**, juste sous le header, contenant :
- À gauche : les vues de premier niveau, sous forme d'onglets — **Projets** (par défaut) et **Priorités**
- À droite : la barre de recherche (2.9), sa bascule sémantique 🧠 (2.12), puis l'accès aux **Préférences** ⚙️ (2.13)

```
┌──────────────────────────────────────────────────────────┐
│ [Icône + nom app]                          [Projet actif] │ ← Header (natif Wails, voir note)
├──────────────────────────────────────────────────────────┤
│ [Projets] [Priorités]   🔍 [Rechercher...] [X] [🧠] [⚙️] │ ← Bande transverse
├──────────────────────────────────────────────────────────┤
│  contenu de la vue active (Projets, Priorités,            │
│  Préférences, ou résultats de recherche si une            │
│  recherche est active)                                     │
└──────────────────────────────────────────────────────────┘
```

Les deux icônes de droite ne sont **pas de même nature**, et le schéma ne doit pas le laisser croire :
- **🧠 est une bascule d'état** — elle modifie le comportement de la barre de recherche à côté de laquelle elle se trouve, et reste décochée par défaut
- **⚙️ est une navigation** — elle ouvre la vue Préférences, au même titre que les onglets de gauche

**Vue Projets** : sidebar projets (gauche, largeur redimensionnable) + sous-onglets Tâches/Notes/Réunions (droite)

**Vue Priorités** : pleine largeur, pas de sidebar, split interne liste/détail (2.8)

**Vue Préférences** : pleine largeur, pas de sidebar, contenu statique (2.13)

**Recherche active** : remplace tout le contenu, pleine largeur, quelle que soit la vue de premier niveau affichée avant la recherche — Projets, Priorités ou Préférences

**Panneaux redimensionnables**
Toutes les séparations entre deux zones de contenu sont redimensionnables au glisser-déposer, de façon cohérente dans toute l'app :
- Sidebar projets ↔ contenu (vue Projets)
- Arbre de tâches ↔ panneau de détail (onglet Tâches)
- Liste de notes ↔ éditeur (onglet Notes)
- Réunions ↔ Instances ↔ Éditeur (onglet Réunions, deux séparations)
- Liste ↔ détail (vue Priorités)
- Résultats ↔ détail (vue Recherche)

**⚠️ Note importante pour le portage Wails**

La barre de titre "Tout Doux" affichée en haut du prototype web (avec le nom de l'app) **ne doit pas être reprise telle quelle dans l'application Wails finale**. Elle sera remplacée par la barre de fenêtre native du système d'exploitation (icône de l'app déjà prête séparément). Cette barre HTML custom n'existe que parce que le prototype tourne dans un simple onglet de navigateur, sans fenêtre native autour.

---

### 2.12 RECHERCHE SÉMANTIQUE (nouvelle section)

**Le problème résolu**

La recherche mot-clé du §2.9, adossée à FTS5, rate deux situations fréquentes :

1. **Une faute de frappe**, dans la requête ou dans le contenu indexé. « réuion » ne trouve rien.
2. **Une reformulation** : les mêmes idées avec d'autres mots. Chercher « client mécontent » ne remonte pas une note qui dit « client en colère ».

La recherche sémantique compare des **vecteurs de sens** plutôt que des chaînes de caractères. Elle ne remplace pas la recherche mot-clé, qui reste plus rapide et plus précise quand on sait exactement quel mot on cherche : les deux coexistent, l'utilisateur choisit.

**Opt-in strict, entité par entité**

Rien n'est vectorisé automatiquement. Calculer un vecteur coûte quelques centaines de millisecondes de CPU ; le faire sur tout le contenu sans qu'on l'ait demandé serait un coût imposé pour un bénéfice non désiré.

- Un bouton **« Ajouter à la recherche sémantique »** est présent sur :
  - les **notes** (§2.6) — titre et contenu
  - les **instances de réunion** (§2.7) — notes de l'instance
  - les **tâches** (§2.2) — nom et description
- L'entité passe alors dans l'index sémantique, et le bouton reflète cet état

**Maintien automatique après activation**

Une fois une entité activée, l'utilisateur n'a plus rien à faire. Chaque modification de son **texte** déclenche une revectorisation silencieuse en arrière-plan, au même rythme que la sauvegarde automatique déjà en place (§2.6, §2.7) — délai d'inactivité, plus vidage immédiat au changement de sélection.

> **Point de vigilance pour l'implémentation.** La revectorisation ne doit se déclencher que sur les champs **réellement textuels** : nom et description pour une tâche, titre et contenu pour une note, notes pour une instance. Les changements de **métadonnées** — importance, échéance, statut terminé ou annulé, ordre — ne modifient pas le sens du texte et ne doivent rien recalculer. Sans ce filtre, cocher une case déclencherait un calcul de plusieurs centaines de millisecondes pour un résultat identique. C'est un piège spécifique à cette fonctionnalité : dans le prototype React, recalculer un dérivé était gratuit ; ici, ça ne l'est pas.

**Résultats**

La vue résultats est celle du §2.9 — même liste à gauche, même aperçu à droite, mêmes icônes par type, même navigation au double-clic. Deux différences :

- Le tri se fait par **similarité décroissante** et non par date : en recherche sémantique, la pertinence est un score, pas une chronologie
- Il n'y a **pas de surbrillance** dans l'extrait : aucun mot exact n'a été mis en correspondance, il n'y a donc rien à surligner. L'extrait montre le début du contenu

Seules les entités explicitement ajoutées à l'index sémantique peuvent remonter. Quand l'index est vide, la vue le dit clairement plutôt que d'afficher « aucun résultat », qui ferait croire à une absence de contenu correspondant.

**Zéro réseau — contrainte non négociable**

L'application n'accède jamais à Internet. Le poste de travail est derrière un proxy d'entreprise, et le §6 pose déjà l'application comme hors-ligne.

Cela signifie, sans exception :

- **Aucun téléchargement automatique** du modèle, ni au premier lancement, ni à la demande
- **Aucune vérification de mise à jour**, du modèle comme du runtime
- **Aucune télémétrie**

**Le modèle est déposé par l'utilisateur**

Le modèle n'est **ni embarqué dans l'installeur, ni téléchargé**. L'utilisateur le récupère lui-même et le dépose dans un dossier précis (§3.3).

Si le modèle est absent au moment où on active la recherche sémantique, une fenêtre l'explique en donnant tout ce qu'il faut pour agir :

- le **chemin exact** du dossier attendu
- les **noms de fichiers exacts** attendus
- le **lien de téléchargement**, à ouvrir depuis un poste qui a accès au réseau
- un bouton **« Réessayer »** — jamais de tentative automatique

**Limite connue et assumée**

Si le modèle est absent au moment précis où une entité déjà vectorisée est modifiée, la mise à jour silencieuse échoue silencieusement : le vecteur reste celui de la version précédente du texte. Aucun indicateur ne le signale pour l'instant.

C'est **documenté comme amélioration future possible, pas comme un point bloquant** à traiter maintenant. Le cas suppose que l'utilisateur retire le modèle après l'avoir installé, ce qui est rare, et la conséquence est un résultat de recherche légèrement périmé — pas une perte de données.

---

### 2.13 VUE PRÉFÉRENCES (nouvelle section)

Troisième vue de premier niveau, accessible par une **icône ⚙️** dans la bande transverse (§2.11), à droite de la barre de recherche.

**Contenu**

- **Statut du modèle sémantique** : présent ou absent, avec le détail fichier par fichier
- **Chemin attendu** du dossier, affiché en toutes lettres et copiable
- **Noms de fichiers attendus**, listés explicitement
- **Lien de téléchargement** du modèle
- **Bouton de vérification** : relance la détection et met le statut à jour

**Pourquoi une vue et pas seulement une fenêtre**

La fenêtre du §2.12 ne s'affiche qu'au moment où l'utilisateur active la recherche sémantique sans modèle. Elle disparaît ensuite. Cette vue sert de **référence permanente** : on doit pouvoir y revenir des semaines plus tard pour retrouver le chemin ou le lien, sans avoir à reproduire la condition d'erreur.

---

## 3. SPÉCIFICATION TECHNIQUE

### 3.1 ARCHITECTURE GLOBALE

*(Structure générale inchangée dans son principe depuis la v1 — mise à jour du langage backend suite au choix Wails/Go, voir 3.5. La philosophie hexagonale de 3.9 s'applique à l'identique.)*

```
┌─────────────────────────────────┐
│  FRONTEND (React + TypeScript)   │
│  ├─ Components/                 │
│  ├─ Pages/                      │
│  ├─ Hooks/                      │
│  ├─ Store/ (Zustand)            │
│  └─ Styles/ (Tailwind)          │
└────────────────┬────────────────┘
                 │ Bindings Wails (JS ↔ Go)
                 ↓
┌─────────────────────────────────┐
│  BACKEND (Go + Wails)           │
│  ├─ app/          (adaptateur pilote, méthodes exposées au frontend)
│  ├─ domain/       (cœur métier, pur)
│  ├─ ports/        (interfaces Repository, Clock, SearchIndex, EmbeddingProvider)
│  ├─ adapters/     (SQLite, FTS5, ONNX — adaptateurs pilotés)
│  └─ errors.go                  │
└──────┬──────────────────┬───────┘
       │ SQL              │ inférence locale (2.12)
       ↓                  ↓
┌────────────────┐  ┌──────────────────────┐
│  SQLite (DB)   │  │ onnxruntime.dll      │ ← livrée avec l'app
│  /app-data/    │  │ + modèle ONNX        │ ← déposé par l'utilisateur
│  app.db        │  │ /app-data/models/    │
└────────────────┘  └──────────────────────┘
```

Les deux dépendances de droite sont **locales**, jamais distantes : aucune flèche ne sort
de ce schéma vers l'extérieur, et c'est une contrainte, pas un oubli (§6).

---

### 3.2 BASE DE DONNÉES (SQLite)

*(Schéma inchangé depuis la v1 — déjà cohérent avec le prototype, y compris `order_index` sur les tâches)*

```sql
-- Projets
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    locked BOOLEAN DEFAULT 0,   -- true uniquement pour "Transverse/Divers"
    created_at DATETIME,
    updated_at DATETIME
);

-- Tâches
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    parent_id TEXT,  -- NULL si racine
    name TEXT NOT NULL,
    description TEXT,
    importance TEXT CHECK(importance IN ('Basse', 'Normale', 'Haute', 'Critique')),
    completed BOOLEAN DEFAULT 0,
    cancelled BOOLEAN DEFAULT 0,
    due_date DATETIME,
    order_index INTEGER,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id),
    FOREIGN KEY(parent_id) REFERENCES tasks(id)
);

-- Notes
CREATE TABLE notes (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

-- Réunions
CREATE TABLE meetings (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

-- Instances de réunion
CREATE TABLE meeting_instances (
    id TEXT PRIMARY KEY,
    meeting_id TEXT NOT NULL,
    notes TEXT,
    timestamp DATETIME,
    created_at DATETIME,
    updated_at DATETIME,
    FOREIGN KEY(meeting_id) REFERENCES meetings(id)
);

-- Images
CREATE TABLE images (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    owner_type TEXT,  -- 'note' | 'meeting' | 'task'
    owner_id TEXT,
    size INT,
    format TEXT,
    created_at DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

-- Settings
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- Full-text search index (FTS5)
CREATE VIRTUAL TABLE search_index USING fts5(
    type,       -- 'note' | 'meeting' | 'task'
    title,
    content,
    project_id,
    entity_id,
    created_at
);

-- Index sémantique (2.12). Table ordinaire, pas une table virtuelle : la
-- similarité se calcule en Go à la lecture, il n'y a rien à indexer côté SQLite.
CREATE TABLE embeddings (
    entity_id   TEXT PRIMARY KEY,   -- une entité, un vecteur
    type        TEXT NOT NULL,      -- 'note' | 'meeting' | 'task'
    project_id  TEXT NOT NULL,
    vector      BLOB NOT NULL,      -- float32 sérialisés
    dimensions  INTEGER NOT NULL,   -- garde-fou : refuser de comparer deux
                                    -- vecteurs de tailles différentes, ce qui
                                    -- arriverait si le modèle changeait
    source_hash TEXT NOT NULL,      -- empreinte du texte vectorisé, pour ne pas
                                    -- recalculer un vecteur inchangé
    updated_at  DATETIME,
    FOREIGN KEY(project_id) REFERENCES projects(id)
);

CREATE INDEX idx_embeddings_project ON embeddings(project_id);
```

**Pourquoi pas d'extension vectorielle**

`sqlite-vec` et consorts apportent une recherche vectorielle approchée, utile à partir de centaines de milliers de vecteurs. Une application personnelle en compte quelques milliers au plus : un balayage complet avec similarité cosinus calculée en Go pur reste sous la milliseconde, et évite d'ajouter une extension native à charger — ce qui irait à l'encontre du choix « sans CGO » du §3.5.

`source_hash` mérite un mot : c'est ce qui évite de revectoriser un texte qu'on vient de réenregistrer sans l'avoir modifié. Combiné au filtrage sur les champs textuels (§2.12), il supprime l'essentiel des calculs inutiles.

---

### 3.3 STOCKAGE FICHIERS

*(Inchangé depuis la v1)*

```
/app-data/
├─ app.db
├─ images/
│  ├─ note-uuid-1.png
│  ├─ meeting-uuid-3.png
│  └─ task-uuid-4.webp
└─ models/                        ← déposé par l'utilisateur (2.12)
   ├─ model.onnx
   ├─ tokenizer.json
   └─ sentencepiece.bpe.model
```

**Dossier du modèle sémantique (2.12)**

Les trois fichiers sont attendus sous ce nom exact, dans `/app-data/models/`. Le dossier est celui des données utilisateur — pas celui de l'exécutable, qui peut être installé dans un emplacement non inscriptible.

L'application **ne crée pas** ces fichiers et ne les télécharge jamais. Elle se contente de vérifier leur présence et d'afficher le chemin attendu quand ils manquent (§2.13).

`onnxruntime.dll` fait exception et n'est **pas** dans ce dossier : c'est une bibliothèque d'exécution, livrée avec l'application, à côté de l'exécutable (§3.5).

**Stratégie images**
- Image < 50KB → BLOB en base64 dans le contenu
- Image ≥ 50KB → fichier `/app-data/images/uuid.{format}`, référencé via `app://images/uuid.png`

**Formats acceptés** : PNG, JPG, WEBP, GIF — max 10MB, auto-compress si > 2MB (WebP)

---

### 3.4 ÉDITEUR RICHE (TipTap) — reporté

**Décision prise pendant le prototypage** : le prototype React utilise volontairement de simples `<textarea>` pour Notes/Réunions/Description de tâche, plutôt que TipTap. Raison : TipTap n'est pas disponible dans le bac à sable de prototypage utilisé, et l'objectif du prototype était de valider les interactions et le layout, pas le rendu riche.

**Ce qui reste vrai pour la version Wails finale** :
- TipTap est une librairie JS pure (React + ProseMirror), aucune dépendance au langage backend, aucun lien avec Wails lui-même
- Le seul point de contact avec le backend : TipTap produit du HTML, stocké tel quel dans les colonnes `TEXT` de SQLite
- Point de vigilance déjà identifié : les images collées avec URL externe casseront visuellement en offline — nécessite une interception du paste pour rapatrier l'image en local et réécrire l'URL en `app://images/...`

Extensions cibles, barre d'outils, comportements (drag & drop images, paste Word/Docs, undo/redo) : voir contenu détaillé de la v1, inchangé.

---

### 3.5 STACK TECHNIQUE

**Choix Wails/Go plutôt que Tauri/Rust** — décidé après le prototypage, pour une raison purement pratique : la machine de développement n'a pas les droits admin nécessaires à l'installation de Rust + MSVC Build Tools (plusieurs Go de dépendances, installeur qui demande l'élévation). Wails n'exige que :
- Go, installable en version portable (zip à extraire, PATH utilisateur) — aucun droit admin
- Un compilateur C (déjà disponible via MSYS2/mingw-w64 sur cette machine) — aucun droit admin non plus une fois MSYS2 en place
- Le runtime WebView2, déjà préinstallé nativement sur Windows 10 (≥1809) et Windows 11

Même philosophie que Tauri : webview système (pas de Chromium embarqué comme Electron), exécutable léger, binding typé entre frontend et backend natif. La seule différence structurante : le langage backend est Go plutôt que Rust, ce qui ne remet rien en cause dans la philosophie hexagonale (section 3.9) — une interface Go joue exactement le rôle d'un trait Rust pour définir un port.

**Frontend** : React 18+, TypeScript, Vite, Tailwind CSS, TanStack Query, Zustand, TipTap 2.x — inchangé

**Backend** : Go 1.21+, Wails v2/v3, SQLite via `mattn/go-sqlite3` (nécessite CGO, déjà disponible) ou `modernc.org/sqlite` (pur Go, sans CGO — option à évaluer si on veut supprimer la dépendance au compilateur C même pour la persistance), FTS5 (fonctionnalité SQLite, indépendante du langage hôte)

**Desktop** : WebKit2GTK (Linux) / WebView2 (Windows) / WKWebView (macOS) — identique à Tauri, c'est la même famille d'approche "webview système". Bundler Wails (`wails build`) pour générer l'exécutable.

**Diagnostic d'environnement** : `wails doctor` vérifie automatiquement la présence du compilateur C et du runtime WebView2 avant de lancer un build — déjà validé sur cette machine.

**Recherche sémantique (2.12)**

| Élément | Choix | Taille | Distribution |
|---|---|---|---|
| Modèle | `intfloat/multilingual-e5-small`, export ONNX, **quantifié int8** | ~120 Mo | **déposé par l'utilisateur**, jamais dans l'installeur |
| Runtime | `onnxruntime.dll` | ~15-20 Mo | **inclus dans l'installeur** |
| Liaison Go | `github.com/yalue/onnxruntime_go` | — | dépendance Go |

Trois décisions, et leurs raisons :

**Le modèle est multilingue et quantifié.** `multilingual-e5-small` couvre le français, ce qu'un modèle anglophone ne ferait pas sur ce corpus. La version int8 pèse ~120 Mo contre ~470 Mo en fp32, pour une perte de qualité marginale sur de la recherche de similarité — le rapport est sans comparaison à cette échelle.

**Le modèle n'est pas dans l'installeur, le runtime si.** La distinction est volontaire et tient à la nature des deux : `onnxruntime.dll` est une bibliothèque d'exécution, du même ordre que le runtime WebView2, et l'inclure ne pose pas plus de question que de livrer n'importe quelle dépendance. Le modèle, lui, est une donnée de 120 Mo qui quadruplerait le poids de l'installeur pour une fonctionnalité optionnelle que tous les utilisateurs n'activeront pas.

**`yalue/onnxruntime_go` charge la DLL dynamiquement**, à l'exécution, et n'impose donc **pas de CGO à la compilation**. C'est ce qui permet à cette fonctionnalité de ne pas remettre en cause la chaîne de build sans compilateur C établie plus haut — c'était une condition d'acceptation, pas un heureux hasard. Concrètement, il y a un fichier de plus à côté de l'exécutable, et rien ne change au `wails build`.

---

### 3.6 COMMANDES BACKEND (API Wails)

*(Complétées par rapport à la v1 : ajout des commandes que le prototype a révélées comme nécessaires — cascade de réactivation, tout étendre/réduire côté frontend uniquement donc pas de commande dédiée)*

**Projets**
- `create_project(name)`
- `rename_project(id, new_name)`
- `delete_project(id)` — cascade tâches/notes/réunions/instances
- `list_projects()` → `[]Project`

**Tâches**
- `create_task(project_id, parent_id?, name)` → `Task` — réactive les ancêtres si besoin (règle 2.2)
- `update_task(task_id, ...fields)` → `Task`
- `delete_task(task_id)` — cascade sous-tâches
- `get_tasks(project_id)` → `[]Task`
- `toggle_completed(task_id)` → `[]Task` (la tâche + tous les ancêtres/descendants impactés par la cascade)
- `toggle_cancel(task_id)` → `Task`
- `move_task(task_id, direction)` — `'up' | 'down' | 'top' | 'bottom'`
- `reparent_task(task_id, new_parent_id?)` → `[]Task` (réactive les ancêtres si besoin)

**Notes**
- `create_note(project_id, title)` → `Note`
- `update_note(note_id, title, content)` → `Note`
- `delete_note(note_id)`
- `get_notes(project_id)` → `[]Note`

**Réunions**
- `create_meeting(project_id, title)` → `Meeting`
- `rename_meeting(meeting_id, new_title)` → `Meeting`
- `delete_meeting(meeting_id)` — cascade instances
- `get_meetings(project_id)` → `[]Meeting`
- `add_meeting_instance(meeting_id)` → `MeetingInstance`
- `update_instance_notes(instance_id, notes)` → `MeetingInstance`
- `delete_instance(instance_id)`
- `get_instances(meeting_id)` → `[]MeetingInstance`

**Priorités**
- `get_priority_tasks()` → `{ CriticalOrHigh: []Task, DueSoon: []Task }` — lecture transverse tous projets, réutilisable telle quelle par le menu taskbar (2.10)

**Recherche**
- `search_global(query)` → `[]SearchResult` — toujours globale, plus de variante `search_project` (le mode Projet a été retiré côté produit)

**Images**
- `upload_image(project_id, image_data, format)` → `image_url`
- `get_image(image_id)` → `image_data`

**Recherche sémantique (2.12)**
- `search_semantic(query)` → `[]SearchResult` — triés par similarité décroissante, jamais mélangés aux résultats mot-clé
- `add_to_semantic_index(entity_type, entity_id)` → vectorise et enregistre ; c'est l'action du bouton « Ajouter à la recherche sémantique »
- `remove_from_semantic_index(entity_id)`
- `is_in_semantic_index(entity_id)` → `bool` — alimente l'état du bouton
- `refresh_embedding(entity_id)` — revectorisation silencieuse après modification du texte. Sans effet si l'entité n'est pas dans l'index, et sans effet si `source_hash` est inchangé

**Modèle sémantique (2.13)**
- `get_model_status()` → `{ Available: bool, Directory: string, ExpectedFiles: []string, MissingFiles: []string, DownloadURL: string }` — une seule commande alimente à la fois la fenêtre du §2.12 et la vue Préférences du §2.13, pour que les deux ne puissent pas diverger

**Settings**
- `get_setting(key)` → `value`
- `set_setting(key, value)`

---

### 3.7 PERSISTENCE & DURABILITY

*(Inchangée depuis la v1)*

- Sauvegarde auto sur chaque changement, transactions ACID, pas de "Save" manuel
- Backup : copie de `/app-data/app.db` + dossier images
- Récupération : restore des deux + redémarrage

---

### 3.8 PERFORMANCE

*(Inchangée depuis la v1)*

- Startup < 500ms, recherche globale < 100ms (10k docs), rendu arbre < 100ms (500 tâches), mémoire < 100MB
- FTS5, requêtes ciblées, virtualisation si besoin, debounce 300ms, lazy-load images > 50KB

**Recherche sémantique (2.12) — objectifs distincts**

Cette fonctionnalité ne tient pas les chiffres ci-dessus, et ce n'est pas une régression : les ordres de grandeur d'une inférence de modèle n'ont rien à voir avec ceux d'une requête SQL.

| Opération | Ordre de grandeur | Remarque |
|---|---|---|
| Chargement du modèle | quelques secondes | une fois, au premier usage — **jamais au démarrage de l'app**, sinon le budget des 500 ms saute |
| Vectorisation d'un texte | quelques centaines de ms | c'est ce qui justifie l'opt-in et le filtrage sur les champs textuels (§2.12) |
| Recherche sur l'index | < 10 ms | une vectorisation de la requête, puis un balayage de quelques milliers de vecteurs |
| Mémoire, modèle chargé | +150 à 250 Mo | dépasse le budget des 100 Mo, assumé : c'est le prix d'une fonctionnalité optionnelle, et l'app y revient dès qu'elle est inutilisée |

Deux conséquences à respecter : le modèle est chargé **paresseusement**, à la première utilisation réelle et pas au lancement ; et la vectorisation se fait **hors du fil de l'interface**, sinon la frappe se figerait pendant le calcul.

---

### 3.9 ARCHITECTURE HEXAGONALE — PRINCIPES ET PÉRIMÈTRE

**Décision de fond** : architecture hexagonale (ports & adapters), mais **appliquée avec discernement**, pas de façon systématique. Une interface (port) n'est introduite que si l'une de ces deux conditions est vraie :
1. Une deuxième implémentation concrète existe déjà ou est concrètement prévue (pas hypothétique)
2. Un besoin de test réel, sans infrastructure lourde, serait autrement pénible ou lent

Ces principes ont été définis pendant le prototypage en pensant à Rust/Tauri, mais **rien n'en est invalidé par le passage à Go/Wails** : une `interface` Go joue exactement le rôle d'un `trait` Rust pour définir un port, et un `struct` qui l'implémente joue le rôle d'un adaptateur. Seule la colonne "langage cible" change ci-dessous.

**Domaine (Go, pur, zéro dépendance à SQLite/Wails/HTTP)**

Les fonctions suivantes, déjà isolées et testées comme fonctions pures dans le prototype React, sont directement portables en Go et constituent le cœur du domaine :

| Fonction du prototype | Rôle | Devient en Go |
|---|---|---|
| `formatDueDate`, `quickDueDate`, `nextWeekday9h` | Échéances intelligentes | Package `domain/duedate` |
| `buildChildrenMap`, `computeVisibility`, `matchesFilter` | Arbre et filtres | Package `domain/tasktree` |
| `defaultExpandedForProject` | Dépliement par défaut | Package `domain/tasktree` |
| Cascade de réactivation (dans `addTask`, `reparentTask`, `toggleTaskCompleted`) | Cohérence parent/enfant | Package `domain/cascade` — règle métier à part entière, pas un détail d'implémentation UI |
| `getSidebarStats`, `urgencySort` | Agrégations sidebar/priorités | Package `domain/stats` |
| `computeSearchResults`, `highlightSnippet` | Recherche | Package `domain/search` |

**Ports (interfaces Go) à prévoir**

| Port | Pourquoi un port ici | Adaptateur(s) |
|---|---|---|
| `TaskRepository`, `ProjectRepository`, `NoteRepository`, `MeetingRepository` | SQLite est amené à évoluer (format, sync) ; tests du domaine sans vraie base | `SqliteXxxRepository` (réel) ; `InMemoryXxxRepository` (tests unitaires domaine) |
| `SearchIndex` | Le moteur d'indexation pourrait changer ; tests de matching/surbrillance sans FTS5 réel | `Fts5SearchAdapter` |
| `Clock` (`Now() time.Time`) | Les règles d'échéance sont sensibles au temps (urgence < 5min, vendredi → lundi) ; sans ce port, les tester proprement est pénible | `SystemClock` (réel, `time.Now()`) ; horloge figée pour les tests |
| `EmbeddingProvider` (`Embed(text) ([]float32, error)`) | Sans ce port, tester la moindre règle touchant à la recherche sémantique imposerait de charger un modèle de 120 Mo à **chaque** `go test` — la suite passerait de moins d'une seconde à plusieurs dizaines. C'est exactement le second critère : un besoin de test réel, autrement pénible | `OnnxEmbeddingProvider` (réel) ; `FakeEmbeddingProvider` (vecteur déterministe, tests) |

Le `FakeEmbeddingProvider` rend un vecteur **déterministe dérivé du texte** : deux textes identiques donnent le même vecteur, deux textes différents des vecteurs différents. C'est suffisant pour tester tout ce qui entoure le modèle — l'opt-in, la revectorisation sur changement de texte, le non-recalcul sur changement de métadonnée, le tri par similarité, la séparation des deux modes de recherche. Ce qu'il ne teste pas, et ne prétend pas tester, c'est la **qualité sémantique** du vrai modèle : que « client en colère » soit proche de « client mécontent » relève du modèle, pas de notre code.

**Explicitement PAS mis derrière un port**

- **État UI pur** (largeur des panneaux, arbre déplié/replié, onglet/vue active, menu contextuel ouvert) — reste du `useState` React, ne remonte jamais au domaine
- **Settings** — CRUD trivial, appel direct, aucune règle métier dessus
- **Le pont de binding Wails côté frontend** (méthodes Go exposées automatiquement en JS) — pas d'interface `TaskApi` introduite tant qu'aucune migration de techno n'est planifiée ; le besoin réel identifié (mock pour les tests de composants React) est déjà couvert par le fait que les composants reçoivent leurs données en props, sans jamais appeler les bindings Wails eux-mêmes (voir 3.10)

**Le point le plus important à retenir pour le développement à venir**

Le domaine ne connaît que des interfaces (le tableau ci-dessus), jamais leurs implémentations concrètes. SQLite peut être remplacé un jour (autre moteur, sync cloud) sans toucher une ligne de règle métier — c'est SQLite qui *implémente* le contrat défini par le domaine, pas l'inverse.

---

### 3.10 STRATÉGIE DE TEST (précision par rapport à `STRATEGIE_QA_TESTING.md`)

La pyramide de test décrite dans `STRATEGIE_QA_TESTING.md` reste valable. Cette section précise **où** chaque type de test s'exécute concrètement, décidé pendant la discussion d'architecture :

1. **Tests domaine (Go)** — contre des fonctions pures uniquement, zéro infrastructure (`go test` standard). Exemple : la cascade de réactivation des parents, les calculs d'échéance.
2. **Tests d'intégration repository (Go)** — contre une **vraie SQLite en mémoire** (`:memory:`), pas de mock ni de port simulé. Objectif : vérifier qu'ajouter puis relire une donnée renvoie bien la même chose (round-trip), **pas** que SQLite fonctionne. Aucun port n'est nécessaire pour ce niveau : on instancie directement l'adaptateur réel.
3. **Tests de composants (React Testing Library)** — les données sont passées en **props depuis des fixtures JSON**, jamais lues depuis un vrai backend. Déjà permis par l'architecture du prototype : aucun composant (`TaskNode`, `DetailPanel`, `NotesTab`...) n'appelle un binding Wails ou le stockage directement — seul le composant racine le fait. Ce découplage suffit, sans introduire de port frontend dédié.
4. **Tests E2E (Playwright)** — contre l'app Wails complète, incluant la régression visuelle de layout (captures d'écran comparées à une référence).

**Recherche sémantique (2.12)**

Toute la suite tourne contre `FakeEmbeddingProvider`, jamais contre le vrai modèle. Charger 120 Mo à chaque `go test` ferait passer la suite de moins d'une seconde à plusieurs dizaines, et rendrait les tests impossibles à exécuter sur une machine où le modèle n'est pas déposé — c'est-à-dire toute machine d'intégration continue, puisque le modèle n'est ni dans le dépôt ni téléchargeable.

Ce qui se teste ainsi : l'opt-in par entité, la revectorisation déclenchée par un changement de texte, l'**absence** de revectorisation sur un changement de métadonnée, le rôle de `source_hash`, le tri par similarité, la séparation stricte des deux modes de recherche, et le comportement quand le modèle est absent.

Ce qui ne se teste pas automatiquement, et doit être vérifié à la main une fois le modèle en place : la **qualité sémantique** des résultats. Que « client en colère » ressorte sur « client mécontent » dépend du modèle, pas de notre code — un test qui l'affirmerait ne testerait que le `Fake`.

---

## 4. DATA FLOW

*(Flows inchangés dans leur principe depuis la v1 ; celui de la sauvegarde note est précisé pour refléter le flush immédiat)*

### Créer une tâche
```
1. Clic droit → "+ Nouvelle tâche" → modale nom
2. Frontend → Backend : create_task(project_id, parent_id?, name)
3. Backend (domaine) : applique la cascade de réactivation des ancêtres si besoin
4. Backend : INSERT INTO tasks (...)
5. Backend → Frontend : Task (+ ancêtres modifiés le cas échéant)
6. Frontend : met à jour l'arbre, déplie le parent, sélectionne la nouvelle tâche
```

### Sauvegarder une note (auto, avec flush immédiat)
```
1. User tape dans l'éditeur
2. Frontend : lance un timer de 2s (debounce)
3a. Timer expire SANS interruption → update_note(id, content) → "Note sauvegardée à 14:32:15"
3b. OU l'utilisateur change de note / change de projet / le champ perd le focus
    AVANT la fin du timer → flush immédiat : update_note(id, content) déclenché
    sur-le-champ, sans attendre le timer
4. Backend : UPDATE notes SET content = ?, updated_at = now()
```

### Chercher "client feedback"
```
1. User tape dans la barre de recherche (bande transverse, toujours visible)
2. > 2 caractères → debounce 300ms → search_global("client feedback")
3. Backend : SELECT * FROM search_index WHERE search_index MATCH 'client feedback'
4. Frontend affiche la vue résultats (remplace tout le contenu principal,
   quelle que soit la vue affichée avant — Projets ou Priorités)
5. Clic simple sur un résultat → aperçu à droite (extrait, surbrillance jaune)
6. Double-clic (ou "Ouvrir") → ferme la recherche, bascule sur la vue Projets,
   sélectionne le bon projet/onglet/élément
7. Échap ou "Fermer" → revient à la vue affichée avant la recherche
```

### Chercher "client mécontent" en mode sémantique (2.12)
```
1. User a coché 🧠 dans la bande transverse
2. > 2 caractères → debounce 300ms → search_semantic("client mécontent")
3. Backend : charge le modèle si ce n'est pas déjà fait (paresseux, quelques
   secondes la première fois), puis vectorise la requête
4. Backend : SELECT entity_id, vector FROM embeddings ; similarité cosinus
   calculée en Go pour chaque vecteur ; tri par score décroissant
5. Frontend affiche la même vue résultats qu'en 2.9, mais SANS surbrillance —
   aucun mot exact n'a été mis en correspondance, il n'y a rien à surligner
6. Une note disant "client en colère" remonte, là où le mode mot-clé ne
   trouvait rien
```

### Ajouter une note à l'index sémantique, puis la modifier
```
1. User clique "Ajouter à la recherche sémantique" sur une note
2. add_to_semantic_index("note", id) → vectorise → INSERT INTO embeddings
   (avec source_hash = empreinte du texte vectorisé)
3. Plus tard, user modifie le CONTENU de la note
4. La sauvegarde auto écrit la note (2.6), puis appelle refresh_embedding(id)
5. Backend : l'entité est dans l'index ET source_hash a changé → revectorise
6. En revanche, si user change seulement l'importance d'une tâche indexée :
   aucun champ textuel touché → AUCUN appel, aucun calcul (2.12)
```

---

## 5. INTERFACES

### 5.1 Layout principal (mis à jour v2)
```
┌────────────────────────────────────────────────────┐
│ [icône + Tout Doux]                                  │ ← barre de fenêtre NATIVE Wails (pas de HTML custom)
├────────────────────────────────────────────────────┤
│ [Projets] [Priorités]        🔍 [Rechercher...] [X] │ ← bande transverse unique
├──────────────────────────────────────────────────────┤
│  Vue Projets :                                        │
│  ┌─────────┬──────────────────────────────────────┐ │
│  │ Projets │ Tâches │ Notes │ Réunions             │ │
│  │ • Divers│ [Filtres, une ligne, icônes]          │ │
│  │ • Proj A│ [Arbre / Liste / Colonnes]            │ │
│  │ • Proj B│ [Détail]                              │ │
│  └─────────┴──────────────────────────────────────┘ │
│                                                        │
│  OU Vue Priorités (pleine largeur, pas de sidebar) :  │
│  ┌──────────────────────┬───────────────────────┐   │
│  │ Liste (2 sections)    │ Détail lecture seule  │   │
│  └──────────────────────┴───────────────────────┘   │
└────────────────────────────────────────────────────┘
```

### 5.2 Barre de recherche (mise à jour v2)
```
🔍 [Rechercher dans les tâches, notes, réunions…]  [X]
```
*(Plus de toggle Global/Projet — toujours globale)*

### 5.3 Éditeur riche
*(Inchangé depuis la v1 — cible TipTap, voir 3.4 pour le statut actuel)*

---

## 6. SPÉCIFICATIONS NON-FONCTIONNELLES

*(Inchangées depuis la v1)*

- **Accessibilité** : alt text images, HTML sémantique, navigation clavier, compatible lecteur d'écran
- **Localisation** : interface française, dates/heures et formats numériques FR
- **Sécurité** : offline-first, données locales uniquement, rendu HTML sécurisé, pas de script utilisateur exécuté
- **Compatibilité** : Windows 10+, macOS 11+, Linux — 4GB RAM / 100MB disque minimum
- **Support** : français uniquement, pas de sync cloud, pas de collaboration multi-utilisateur

**Précision v2 — « hors-ligne » au sens strict (2.12)**

L'arrivée de la recherche sémantique durcit ce point, qui restait implicite tant qu'aucune fonctionnalité n'avait de raison d'accéder au réseau. Le poste de travail est derrière un proxy d'entreprise, et l'application ne doit **jamais** émettre de requête sortante :

- pas de téléchargement de modèle, même à la demande explicite de l'utilisateur
- pas de vérification de mise à jour, ni du modèle ni du runtime
- pas de télémétrie, pas de rapport d'erreur distant

Toute dépendance externe est donc **déposée à la main** (le modèle, §3.3) ou **livrée avec l'application** (`onnxruntime.dll`, §3.5). L'interface se limite à indiquer où trouver ce qui manque ; c'est à l'utilisateur d'aller le chercher depuis un poste qui a accès au réseau.

**Empreinte mémoire.** Le budget de 100 Mo ci-dessus vaut pour l'application seule. Modèle sémantique chargé, elle monte à 250-350 Mo (§3.8). C'est assumé pour une fonctionnalité optionnelle, activée entité par entité.

---

## 7. PHASES DE DÉVELOPPEMENT (mise à jour v2)

### Phase 0 : Prototypage — ✅ terminée
- [x] Prototype React complet (Projets, Tâches, Notes, Réunions, Priorités, Recherche)
- [x] Validation UX et layout avec panneaux redimensionnables
- [x] Logique métier isolée en fonctions pures, prête à être portée

### Phase 1 : Fondations
- [ ] Init projet Wails + React
- [ ] Schéma SQLite (3.2)
- [ ] Domaine Go : port des fonctions pures identifiées en 3.9
- [ ] Ports/traits Repository + adaptateurs SQLite
- [ ] CRUD basique (projets, tâches, notes)
- [ ] UI layout principal (2.11), fenêtre native (sans la barre custom du prototype)

### Phase 2 : Core
- [ ] Hiérarchie tâches + drag-drop + cascade de réactivation (2.2)
- [ ] Filtres icônes/pastilles (2.4)
- [ ] Éditeur TipTap
- [ ] Sauvegarde auto avec flush immédiat (2.6/2.7)

### Phase 3 : Vues transverses + Search
- [ ] Vue Priorités en tant que vue de premier niveau, split liste/détail (2.8)
- [ ] Gestion réunions (par projet)
- [ ] Index FTS5 + `search_global`
- [ ] Barre recherche fusionnée + vue résultats (2.9)

### Phase 4 : Polish
- [ ] Taskbar integration (2.10), réutilisant `get_priority_tasks`
- [ ] Notifications urgence
- [ ] Tests (3.10) : domaine, repository (SQLite en mémoire), composants (fixtures), E2E
- [ ] Styling final

### Phase 5 : Recherche sémantique (2.12, 2.13)
Placée après le Polish et non dans la Phase 3 avec la recherche mot-clé : elle en est
indépendante — les deux modes ne partagent ni index, ni stockage, ni chemin de code —
et elle est optionnelle par nature. La livrer plus tôt retarderait une application
complète pour une fonctionnalité qu'on peut ajouter sans rien casser.

- [ ] Port `EmbeddingProvider` + `FakeEmbeddingProvider` (3.9), **avec ses tests d'abord**
- [ ] Table `embeddings` (3.2) et similarité cosinus en Go pur
- [ ] Adaptateur `OnnxEmbeddingProvider` : chargement paresseux, `yalue/onnxruntime_go`
- [ ] Détection du modèle et `get_model_status` (3.6)
- [ ] Bouton « Ajouter à la recherche sémantique » sur notes, instances et tâches
- [ ] Revectorisation silencieuse, **filtrée sur les champs textuels uniquement** (2.12)
- [ ] Bascule 🧠 dans la barre et vue résultats en mode sémantique (2.9, 2.12)
- [ ] Vue Préférences ⚙️ (2.13)
- [ ] Vérification manuelle de la qualité sémantique, modèle réel en place (3.10)

### Phase 6 : Release
- [ ] Build installers — **avec `onnxruntime.dll`, sans le modèle** (3.5)
- [ ] Documentation utilisateur — dont la procédure de dépôt du modèle
- [ ] Release notes
- [ ] Release publique

---

## 8. DÉFINITION DONE

*(Inchangée depuis la v1)*

✅ Feature "done" quand :
- Code écrit
- Tests passent (domaine / repository / composants / E2E selon 3.10)
- UI/UX validée (cohérente avec le comportement du prototype de référence)
- Docs mise à jour
- Perf acceptable (< objectifs de 3.8)
- Aucune erreur console
- Données persistent correctement
