# 🧪 STRATÉGIE QA & TESTING - Tout Doux

## Problème à éviter

**Régression** = une feature qui fonctionnait avant casse après un changement

**Exemples du passé :**
- ❌ Bouton "Renommer" ne marche plus
- ❌ Interface change de layout (horizontal → vertical)
- ❌ Filtres tâches perdent leur état
- ❌ Drag-drop des tâches casse

**Cause racine :**
- Pas de tests automatisés
- Tests manuels incomplets
- Zéro checklist avant commit
- Pas de CI/CD

---

## STRATÉGIE : 3 couches de testing

```
┌─────────────────────────────────────┐
│ E2E Tests (Playwight)               │  ← User workflows complets
│ Tester comme un vrai utilisateur    │
└─────────────────────────────────────┘
           ↓
┌─────────────────────────────────────┐
│ Integration Tests (Vitest + React)  │  ← Features + BD interactions
│ Tester composants + leurs données   │
└─────────────────────────────────────┘
           ↓
┌─────────────────────────────────────┐
│ Unit Tests (Vitest)                 │  ← Fonctions isolées
│ Tester logique métier               │
└─────────────────────────────────────┘
```

---

## 1. TESTS UNITAIRES (Vitest)

### Quoi tester
**Backend (Rust)**
```
- Logique de filtrage tâches
- Calcul temps restant
- Validation anti-cycle
- Requêtes SQL complexes (ordering, filtering)
```

**Frontend (TypeScript)**
```
- Logique dates (smart due dates)
- Formatage temps restant
- Parsing des résultats search
- Transformations données
```

### Exemple : Smart due date
```
Test: "Si vendredi 17h, +1j doit être lundi 9h"
Input: vendredi 2026-02-20 17:30, clique "+demain 9h"
Expected: lundi 2026-02-23 09:00

Test: "Si samedi, +demain doit être lundi"
Input: samedi 2026-02-21, clique "+demain 9h"
Expected: lundi 2026-02-23 09:00
```

### Fréquence
- ✅ Après chaque changement de logique
- ✅ Avant commit (pré-commit hook)
- ✅ Coverage goal : > 60% (focus logique métier)

---

## 2. TESTS INTÉGRATION (Vitest + React Testing Library)

### Quoi tester
**Composants + données ensemble**
```
- Ajouter une tâche → apparaît dans l'arbre
- Filtrer tâches → liste change
- Éditer note → sauvegarde auto
- Drag-drop tâche → parent change
- Search → résultats corrects
```

### Exemple : Créer tâche et voir dans l'arbre
```gherkin
Given: Projet "Mig24" ouvert dans l'onglet Tâches
When: Je clique [+Tâche], remplis "Test task", Importance "Haute", clique OK
Then: 
  - Tâche apparaît dans l'arbre
  - Background est orange (Haute)
  - Compteur projet passe de 4 à 5
```

### Exemple : Filtrer par criticité
```gherkin
Given: Arbre avec 5 tâches (1 Critique, 2 Hautes, 2 Normales)
When: Je clique filtre "Critique"
Then:
  - Seule la tâche Critique est visible
  - Parents des tâches cachées restent visibles
```

### Fréquence
- ✅ Après chaque feature complète
- ✅ Avant PR/commit
- ✅ Coverage goal : > 40% (features main)

---

## 3. TESTS E2E (Playwright)

### Quoi tester
**Workflows utilisateur complets** (comme un vrai user)
```
1. Créer 3 projets
2. Créer tâches dans chaque projet
3. Ajouter échéances
4. Ajouter notes
5. Chercher un mot
6. Vérifier résultats search
7. Vérifier taskbar tray
8. Fermer app, rouvrir
9. Vérifier données persistent
```

### Scénarios critiques à couvrir

#### Workflow 1 : Gestion tâches
```
1. Créer projet "Test"
2. Ajouter tâche racine "Task 1"
3. Ajouter sous-tâche "Task 1.1"
4. Ajouter échéance "Dans 1h"
5. Marquer Task 1.1 complétée
6. Vérifier Task 1 devient complétée (parent)
7. Renommer Task 1 → "Renamed"
8. Vérifier nom change
9. Supprimer Task 1 → Confirm dialog
10. Vérifier Task 1 et Task 1.1 disparaissent
```

#### Workflow 2 : Notes
```
1. Onglet Notes
2. Créer note "Ma note"
3. Taper du texte riche (gras, italique)
4. Paste du texte depuis web
5. Vérifier "Sauvegardée à HH:mm"
6. Changer projet
7. Revenir à "Test"
8. Vérifier contenu note est intact
9. Supprimer note → Confirm
10. Vérifier disparaît
```

#### Workflow 3 : Recherche
```
1. Créer plusieurs notes/tâches/réunions avec mot "test"
2. Barre recherche : tape "test"
3. Mode Global → Voir tous les résultats
4. Clic sur résultat → Affiche détail
5. Mode Projet → Filtré au projet courant
6. Vérifier surbrillance jaune du mot
7. Fermer search → Revient à onglet précédent
```

#### Workflow 4 : Drag-drop tâches
```
1. Créer 3 tâches racines : A, B, C
2. Drag A sur B → A devient sous-tâche de B
3. Vérifier B est expanded
4. Vérifier hiérarchie correcte dans l'arbre
5. Drag A hors de B → Revient à racine
6. Fermer/Ouvrir app
7. Vérifier hiérarchie persiste
```

#### Workflow 5 : Taskbar
```
1. Créer tâche urgente (< 5min)
2. Vérifier taskbar icon clignote
3. Double-click taskbar → App apparaît
4. Vérifier tâche visible et sélectionnée
5. Click X → App se cache (pas ferme)
6. Vérifier app en arrière-plan
7. Right-click taskbar → Menu
8. Vérifier top 5 prioritaires listées
9. Click sur tâche → App montre et sélectionne
```

### Fréquence
- ✅ Avant chaque release
- ✅ Après feature importante
- ✅ 1x par semaine (run complet)
- ✅ Coverage : 100% des workflows critiques

---

## 4. CHECKLIST PRÉ-COMMIT

Avant chaque commit/PR, vérifier :

### Code
- [ ] Pas d'erreurs TypeScript (`npm run type-check`)
- [ ] Pas d'erreurs Rust (`cargo check`)
- [ ] Tests unitaires passent (`npm run test:unit`)
- [ ] Tests intégration passent (`npm run test:integration`)
- [ ] Linter OK (`npm run lint`)
- [ ] Pas de `console.log()` / `dbg!()`
- [ ] Pas de code commenté

### UI
- [ ] Aucun bouton cassé (renommer, supprimer, ajouter)
- [ ] Tous les filtres marchent
- [ ] Aucune erreur en console
- [ ] Layout responsive OK (resize window)
- [ ] Images s'affichent (si ajoutées)
- [ ] Texte formaté préservé (copier/coller)

### Data
- [ ] Données persistent (fermer/ouvrir app)
- [ ] Aucun orphelin BD (deletion cascade)
- [ ] Undo/Redo fonctionne
- [ ] Sauvegarde auto lance
- [ ] Search index à jour

### Features existantes
- [ ] Créer tâche fonctionne
- [ ] Drag-drop fonctionne
- [ ] Filtres fonctionne
- [ ] Notes fonctionne
- [ ] Réunions fonctionne
- [ ] Search fonctionne
- [ ] Taskbar fonctionne

---

## 5. OUTILS ET SETUP

### Tests unitaires
```bash
npm install -D vitest @vitest/ui @testing-library/react
```

**Config Vitest :** `vitest.config.ts`
```typescript
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      lines: 60,
      functions: 60,
    }
  }
})
```

**Scripts package.json :**
```json
{
  "scripts": {
    "test:unit": "vitest run",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage",
    "test:integration": "vitest run src/integration",
    "test:all": "npm run test:unit && npm run test:integration && npm run test:e2e"
  }
}
```

### Tests E2E
```bash
npm install -D @playwright/test
npx playwright install
```

**Config Playwright :** `playwright.config.ts`
```typescript
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
  },
})
```

**Scripts package.json :**
```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:debug": "playwright test --debug"
  }
}
```

### Pre-commit hook
```bash
npm install -D husky lint-staged
npx husky install
```

**File : `.husky/pre-commit`**
```bash
#!/bin/sh
. "$(dirname "$0")/_/husky.sh"

npm run type-check
npm run lint
npm run test:unit
npm run test:integration
```

---

## 6. WORKFLOW DÉVELOPPEMENT

### Pour chaque feature

```
1. PLAN
   └─ Créer branche feature/xyz

2. DEV
   ├─ Écrire tests (unit + integration)
   ├─ Écrire code
   ├─ Tests passent ✓
   └─ Code review auto (linter, types)

3. PRÉ-COMMIT CHECKLIST
   ├─ Tests passent ✓
   ├─ Pas d'erreurs
   ├─ Features existantes marchent ✓
   └─ UI OK

4. COMMIT
   └─ Hooks lancent tests automatiquement

5. REVIEW (si équipe)
   ├─ Code review
   ├─ Tester feature en local
   └─ Merge si OK

6. RELEASE
   ├─ Tests E2E complets
   ├─ Smoke test (démarrage, usage basic)
   └─ Build + installer
```

---

## 7. TEST MATRIX (ce qu'il faut couvrir)

### Par feature

| Feature | Unit | Integration | E2E |
|---------|------|---|---|
| **Tâches CRUD** | ✅ Logic | ✅ Arbre | ✅ Workflow complet |
| **Drag-drop** | - | ✅ Parent change | ✅ Persist après close |
| **Filtres** | ✅ Logic | ✅ Affichage | ✅ Multi-filter |
| **Échéances smart** | ✅ Dates | ✅ Format affichage | ✅ Vendredi→lundi |
| **Notes** | ✅ Save logic | ✅ Auto-save | ✅ Edit+persist |
| **Réunions** | ✅ CRUD | ✅ Instances | ✅ Historique |
| **Search** | ✅ Query | ✅ Results display | ✅ Highlight + navigate |
| **Taskbar** | ✅ Icons | ✅ Menu build | ✅ Click behavior |

---

## 8. FREQUENCY & AUTOMATION

### Daily
- ✅ Tests unitaires (dev runs)
- ✅ Pre-commit hooks (avant chaque commit)

### Per feature
- ✅ Tests intégration (avant PR)
- ✅ Manual checklist (avant merge)

### Weekly
- ✅ Tests E2E complets (Friday)
- ✅ Smoke test (démarrage + 5 workflows)

### Before release
- ✅ Full test suite (unit + integration + E2E)
- ✅ Manual regression testing
- ✅ Build & install test

---

## 9. REPORTING BUGS

Quand une régression trouvée :

```
Title: [REGRESSION] Feature X broken by PR #123

Description:
- What broke: Rename button doesn't work
- When: After merge of feature/task-editing
- How to reproduce:
  1. Open project
  2. Click rename button
  3. See error in console
- Expected: Dialog opens for renaming
- Actual: Nothing happens, console error "Cannot read property 'id'"

Impact: High (core feature broken)
Priority: Urgent (fix immediately)
Root cause: (after investigation) Missing import in TaskList.tsx
Fix: Add import { renameTask } from commands
```

---

## 10. MÉTRIQUES À TRACKER

```
- Test coverage : Goal > 60% (logic) + > 40% (integration)
- Tests passing : 100% (fail build si <100%)
- Regression bugs : < 1 per release
- Bugs time-to-fix : < 24h
- Feature complete rate : > 90% (tout ce qu'on code "fonctionne")
```

---

## RÉSUMÉ

**Pour éviter les régressions :**

1. ✅ **Tests automatisés** (unit + integration + E2E)
2. ✅ **Pre-commit hooks** (run tests avant de commiter)
3. ✅ **Checklist manuelle** (avant chaque feature)
4. ✅ **Test coverage** (60%+ code métier)
5. ✅ **Workflows E2E** (tester comme vrai user)
6. ✅ **CI local** (run tests avant merge)
7. ✅ **Weekly smoke tests** (regression catch early)

**Temps ajouté au dev :**
- Setup initial : ~2h (outils + config)
- Par feature : +10-20% (écrire tests)
- **ROI** : Zéro régression surprise ✅

