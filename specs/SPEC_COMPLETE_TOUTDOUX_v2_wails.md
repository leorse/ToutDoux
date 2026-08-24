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

**Architecture (nouveau — section 3.9)**
- Décision d'architecture hexagonale **appliquée avec discernement**, pas systématique : voir section 3.9 pour le détail de ce qui est mis derrière un port et ce qui ne l'est pas, et pourquoi

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
- À gauche : les deux vues de premier niveau, sous forme d'onglets — **Projets** (par défaut) et **Priorités**
- À droite : la barre de recherche (2.9)

```
┌────────────────────────────────────────────────────┐
│ [Icône + nom app]                    [Projet actif] │ ← Header (natif Wails, voir note)
├────────────────────────────────────────────────────┤
│ [Projets] [Priorités]      🔍 [Rechercher...] [X]  │ ← Bande transverse
├──────────────────────────────────────────────────────┤
│  contenu de la vue active (Projets, Priorités,       │
│  ou résultats de recherche si une recherche est       │
│  active)                                              │
└────────────────────────────────────────────────────┘
```

**Vue Projets** : sidebar projets (gauche, largeur redimensionnable) + sous-onglets Tâches/Notes/Réunions (droite)

**Vue Priorités** : pleine largeur, pas de sidebar, split interne liste/détail (2.8)

**Recherche active** : remplace tout le contenu, pleine largeur, quelle que soit la vue de premier niveau affichée avant la recherche

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
│  ├─ ports/        (interfaces Repository, Clock)
│  ├─ adapters/     (SQLite, FTS5 — adaptateurs pilotés)
│  └─ errors.go                  │
└────────────────┬────────────────┘
                 │ SQL
                 ↓
        ┌────────────────┐
        │  SQLite (DB)   │
        │  /app-data/    │
        │  app.db        │
        └────────────────┘
```

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
```

---

### 3.3 STOCKAGE FICHIERS

*(Inchangé depuis la v1)*

```
/app-data/
├─ app.db
└─ images/
   ├─ note-uuid-1.png
   ├─ meeting-uuid-3.png
   └─ task-uuid-4.webp
```

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

### Phase 5 : Release
- [ ] Build installers
- [ ] Documentation utilisateur
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
