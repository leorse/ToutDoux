# Tout Doux

Gestionnaire personnel de projets, tâches, notes et comptes rendus de réunion.
Application de bureau Windows, sans serveur et sans accès réseau : tout vit dans
un fichier SQLite sur le poste.

La spécification complète est dans
[specs/SPEC_COMPLETE_TOUTDOUX_v2_wails.md](specs/SPEC_COMPLETE_TOUTDOUX_v2_wails.md).
Ce fichier-ci ne décrit que ce qui sert à construire et à installer.

## Ce que fait l'application

**Projets.** Une liste à gauche, colorée selon la tâche la plus critique qu'elle
contient, avec le nombre de tâches actives et une horloge quand une échéance
approche. Un projet « Transverse / Divers » existe toujours et ne peut être ni
renommé ni supprimé : il sert de point de chute quand on supprime le projet
courant.

**Tâches.** Arbre à profondeur libre, réorganisable au glisser-déposer. Quatre
niveaux d'importance, chacun avec sa couleur de fond. Une tâche peut être
terminée ou annulée ; les deux états se propagent aux sous-tâches, et créer une
sous-tâche sous un parent terminé réactive toute la chaîne d'ancêtres. Les
filtres par statut, par importance et sur la présence d'une échéance gardent les
ancêtres des tâches retenues, pour que l'arbre reste navigable.

**Échéances.** Cinq raccourcis (tout de suite, dans 10 minutes, dans une heure,
fin de journée, lendemain 9 h) plus un sélecteur libre. « Lendemain 9 h » saute
le week-end. Un libellé relatif est recalculé toutes les trente secondes, avec
une horloge d'alerte sous cinq minutes ou en retard.

**Notes et réunions.** Éditeur riche avec listes, liens, blocs de code et images
collées, sauvegarde automatique après deux secondes d'inactivité et vidage
immédiat au changement de sélection. Une réunion regroupe des instances datées,
chacune avec son compte rendu.

**Priorités.** Vue transverse à tous les projets, en deux sections : les tâches
critiques ou hautes, et les échéances à venir. L'onglet affiche le nombre de
tâches concernées, dédoublonné — une tâche critique portant une échéance figure
dans les deux sections mais ne compte que pour une.

**Recherche.** Deux modes qui ne se mélangent jamais.

Le mode par mot-clé s'appuie sur FTS5 et couvre les noms de projets, les tâches
avec leur description, les notes et les comptes rendus. Il surligne le terme
trouvé dans l'extrait.

Le mode sémantique compare des vecteurs de sens : il retrouve un contenu
reformulé — chercher « client mécontent » ramène une note qui dit « le client
est furieux » — et tolère les fautes de frappe. Il fonctionne par adhésion : une
note, une tâche ou un compte rendu n'y entre que si on l'y ajoute
explicitement, parce qu'une vectorisation coûte quelques centaines de
millisecondes. Une fois une entité ajoutée, ses modifications de texte sont
suivies automatiquement. Les résultats sont classés par similarité, affichée
telle quelle à côté de chaque ligne.

**Barre système.** Les cinq tâches les plus urgentes dans le menu du clic droit,
une icône qui change de couleur selon le niveau d'alerte et clignote sous cinq
minutes, une notification pour les échéances imminentes. Un clic sur l'icône
ramène la fenêtre. Fermer la fenêtre arrête l'application.

**Raccourcis.** Tout ce qui crée, renomme ou supprime passe par le clic droit ;
il n'y a aucune barre d'outils. Un double-clic sur le fond d'une liste crée un
élément, un double-clic sur un projet le renomme.

## Construire

### Ce qu'il faut

| Outil | Version utilisée | Remarque |
|---|---|---|
| Go | 1.26.7 | version portable, sans droits administrateur |
| Wails CLI | 2.15.0 | `go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0` |
| Node et npm | 25.8.2 et 11.11.1 | pour le frontend |
| gcc | MSYS2 / mingw-w64 | voir ci-dessous |
| WebView2 | fourni par Windows | préinstallé sur Windows 10 (1809+) et 11 |

**gcc est obligatoire.** La liaison Go vers ONNX Runtime, utilisée par la
recherche sémantique, contient un fichier C et passe donc par cgo. Sans
compilateur C sur le PATH, le paquet `adapters/onnx` est exclu de la compilation
et le build échoue sur `build constraints exclude all Go files`.

Sur cette machine, gcc vient de MSYS2, installé sans droits administrateur. Il
n'est pas sur le PATH par défaut :

```powershell
$env:Path += ";C:\msys64\mingw64\bin"
```

Ajouter ce dossier au PATH utilisateur, dans les variables d'environnement
Windows, évite d'avoir à le refaire à chaque session.

Le reste de l'application ne demande rien de tel : le pilote SQLite est du Go
pur, et c'est délibéré. Si la recherche sémantique devait être abandonnée,
retirer `adapters/onnx` suffirait à revenir à une chaîne de construction sans
compilateur C.

### Développement

```powershell
wails dev
```

Recharge le frontend à chaud et expose les méthodes Go sur
http://localhost:34115 pour les inspecter depuis un navigateur.

### Production

```powershell
wails build
```

Produit `build\bin\ToutDoux.exe`, un exécutable unique.

### Tests

```powershell
go test ./...
cd frontend; npm test
```

Les tests du tokenizer et de l'inférence, dans `adapters/onnx`, se sautent
d'eux-mêmes si le modèle n'est pas déposé. Ils ne peuvent donc pas servir de
garde-fou en intégration continue.

### Jeu d'essai

```powershell
go run ./cmd/seed
```

Remplit la base avec cinq projets, quarante-six tâches, des notes et des
réunions, dont des échéances calées sur l'instant présent pour voir réagir la
barre système. La commande est rejouable et ne touche qu'à ses propres projets.
`go run ./cmd/seed -remove` les retire.

## Déployer

L'application est distribuée en archive à décompresser, sans installeur. Rien
n'est écrit dans la base de registre, aucun droit administrateur n'est requis, et
la désinstallation consiste à supprimer le dossier.

### Ce qui va dans l'archive

| Fichier | Origine |
|---|---|
| `ToutDoux.exe` | `build\bin\` après `wails build` |
| `onnxruntime.dll` | 16 Mo, récupérée dans le cache des modules Go, voir ci-dessous |

Les deux côte à côte, dans le dossier de votre choix.

`onnxruntime.dll` n'est nécessaire qu'à la recherche sémantique. Sans elle,
l'application démarre et fonctionne normalement ; seule cette fonctionnalité se
déclare indisponible. La bibliothèque est fournie par le module Go, ce qui évite
un téléchargement séparé :

```
%USERPROFILE%\go\pkg\mod\github.com\yalue\onnxruntime_go@v1.35.0\test_data\onnxruntime.dll
```

La version compte. `onnxruntime_go` v1.35.0 réclame l'API ONNX Runtime 29, soit
onnxruntime 1.29.0, et n'a pas de repli : une bibliothèque plus ancienne fait
échouer l'initialisation sur un message peu parlant.

L'application cherche la bibliothèque à côté de l'exécutable, puis dans le
répertoire courant, puis dans le dossier des modèles décrit plus bas.

### Le modèle sémantique

Il n'est ni distribué avec l'application, ni téléchargé : l'application n'émet
aucune requête réseau, jamais. C'est à l'utilisateur de le déposer.

Trois fichiers, à récupérer depuis un poste connecté sur
https://huggingface.co/Xenova/multilingual-e5-small/tree/main :

| À télécharger | À déposer sous | Taille |
|---|---|---|
| `onnx/model_int8.onnx` | `model.onnx`, renommé | 118 Mo |
| `tokenizer.json` | `tokenizer.json` | 17 Mo |
| `sentencepiece.bpe.model` | `sentencepiece.bpe.model` | 5 Mo |

Dans ce dossier, créé au premier démarrage :

```
%APPDATA%\ToutDoux\models\
```

La vue Préférences affiche ce chemin, l'état de chaque fichier et le lien de
téléchargement. Le lien y est en texte copiable et non cliquable : ouvrir une
adresse depuis l'application déclencherait une requête sortante.

Le modèle n'est chargé qu'à la première vectorisation, jamais au démarrage.
Chargé, il porte l'empreinte mémoire à environ 350 Mo, contre moins de 100 Mo
sans lui.

### Données utilisateur

```
%APPDATA%\ToutDoux\
    app.db          base SQLite, tout le contenu
    models\         le modèle sémantique, déposé à la main
```

Une sauvegarde consiste à copier ce dossier, application fermée. Les données ne
sont pas dans le dossier de l'exécutable, qui peut se trouver à un emplacement
non inscriptible.

## Organisation du code

```
domain/         règles métier, sans dépendance à l'infrastructure
ports/          interfaces que le domaine attend de l'extérieur
adapters/       SQLite, ONNX, horloge, détection du modèle
app*.go         méthodes exposées au frontend par Wails
tray.go         barre système
frontend/src/   React et TypeScript
cmd/seed/       jeu d'essai
```

L'architecture est hexagonale, appliquée avec discernement : un port n'est
introduit que si une seconde implémentation existe ou si un besoin de test réel
l'exige. `EmbeddingProvider` en est l'exemple type — sans lui, tester la
recherche sémantique imposerait de charger 120 Mo de modèle à chaque `go test`.
