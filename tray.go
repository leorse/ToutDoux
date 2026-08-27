package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/energye/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"toutdoux/domain"
	"toutdoux/domain/duedate"
	"toutdoux/domain/stats"
)

/*
Barre système (§2.10).

Rôle : montrer l'urgence par l'icône, donner accès aux tâches prioritaires, et
permettre de quitter. Un clic sur l'icône ramène la fenêtre au premier plan.

La fenêtre n'est jamais masquée : le bouton X ferme l'application pour de bon,
et la réduire la laisse dans la barre des tâches, comme n'importe quelle
application. C'est ce qui rend ce fichier simple — il n'y a aucun état
d'affichage à suivre.

Deux contraintes de la librairie systray restent structurantes, apprises à la
dure ; les ignorer produit une icône qui se fige :

 1. Les gestionnaires de clic sont appelés **de façon synchrone dans la
    procédure de fenêtre** (wndProc → WM_COMMAND → item.click()). Tout appel
    bloquant à cet endroit fige la boucle de messages, et l'icône cesse de
    répondre à tous les clics. Chaque gestionnaire délègue donc à une goroutine.

 2. ResetMenu() recrée le menu Windows sans détruire le précédent et ne purge
    jamais sa table d'entrées : une poignée de fenêtre fuit à chaque appel. Le
    menu est donc construit **une seule fois** ; seuls les libellés changent
    ensuite, et uniquement lorsqu'ils diffèrent réellement — écrire dans le menu
    est l'opération la plus risquée du lot, autant ne le faire qu'à bon escient.
*/

const (
	// Rythme de recalcul de l'état affiché, aligné sur celui de l'interface
	// (§2.3) : les deux doivent montrer la même chose au même moment.
	trayRefresh = 30 * time.Second

	// Demi-période du clignotement. Assez lent pour ne pas être fatigant,
	// assez rapide pour se remarquer du coin de l'œil.
	trayBlink = 700 * time.Millisecond

	// Nombre d'entrées de tâches dans le menu (§2.10, « Top 5 »).
	trayTopN = 5

	// Libellé des emplacements sans tâche à afficher.
	placeholderVide = "—"
)

// tray porte l'état de l'icône de barre système.
type tray struct {
	app *App

	mu       sync.Mutex
	state    stats.TrayState
	inverted bool // phase courante du clignotement
	cycles   int  // nombre de rafraîchissements, pour le journal

	// slotTasks est la tâche affichée par chaque entrée de menu. Les entrées
	// sont fixes, leur contenu change : un gestionnaire de clic lit la tâche ici
	// plutôt que de la capturer à la construction du menu.
	slotTasks []domain.Task

	// libelles retient le texte actuellement écrit dans chaque entrée, pour
	// n'écrire dans le menu que lorsque quelque chose a réellement changé.
	libelles []string

	// notified retient les tâches déjà signalées, pour ne pas répéter la même
	// notification à chaque rafraîchissement.
	notified map[string]bool

	slots []*systray.MenuItem

	stop chan struct{}
}

// startTray démarre la barre système (§2.10).
//
// systray.Run bloque, exactement comme wails.Run : les deux ne peuvent pas
// occuper le même fil, d'où la goroutine.
func (a *App) startTray() {
	a.tray = &tray{
		app:      a,
		notified: map[string]bool{},
		libelles: make([]string, trayTopN),
		stop:     make(chan struct{}),
	}
	go systray.Run(a.tray.onReady, func() {})
}

// stopTray arrête proprement la barre système.
func (a *App) stopTray() {
	if a.tray == nil {
		return
	}
	close(a.tray.stop)
	systray.Quit()
}

func (t *tray) onReady() {
	systray.SetIcon(TrayIcon(TrayNormal))
	systray.SetTitle("Tout Doux")
	systray.SetTooltip("Tout Doux")

	// Clic et double-clic ramènent la fenêtre au premier plan (§2.10). Il n'y a
	// pas de bascule : la fenêtre n'est jamais masquée par l'application.
	systray.SetOnClick(func(systray.IMenu) {
		log.Printf("[tray] ICÔNE clic gauche  %s", t.etat())
		t.enGoroutine("afficher", t.showWindow)
	})
	systray.SetOnDClick(func(systray.IMenu) {
		log.Printf("[tray] ICÔNE double-clic  %s", t.etat())
		t.enGoroutine("afficher", t.showWindow)
	})

	// Le clic droit n'est pas intercepté : la librairie ouvre le menu elle-même.
	// Reprendre cette responsabilité s'était soldé par un gel immédiat.

	t.buildMenu()

	// Le premier rafraîchissement part dans la boucle, pas ici : onReady
	// s'exécute sur le fil de la barre système, avant que sa boucle de messages
	// ne démarre. Y faire un accès base de données retarderait sa mise en service.
	go t.loop()
}

// buildMenu crée les entrées de menu, une fois pour toutes (§2.10).
//
// Cinq emplacements de tâches, puis « Quitter ». Leur nombre ne change jamais :
// seul leur libellé est réécrit, et uniquement quand il diffère.
func (t *tray) buildMenu() {
	for i := 0; i < trayTopN; i++ {
		item := systray.AddMenuItem(placeholderVide, "")
		t.libelles[i] = placeholderVide
		index := i
		item.Click(func() { t.enGoroutine("ouvrir tâche", func() { t.openSlot(index) }) })
		t.slots = append(t.slots, item)
	}

	systray.AddSeparator()

	quitter := systray.AddMenuItem("Quitter", "Fermer Tout Doux")
	quitter.Click(func() { t.enGoroutine("quitter", t.quitApp) })
}

// loop entretient l'icône, l'infobulle et les libellés du menu.
//
// Un seul fil pilote la barre système : les deux tickers y sont multiplexés
// plutôt que lancés séparément, ce qui évite d'avoir à synchroniser deux
// goroutines qui écriraient l'icône en même temps.
func (t *tray) loop() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[tray] PANIQUE dans la boucle : %v", r)
		}
	}()

	refresh := time.NewTicker(trayRefresh)
	blink := time.NewTicker(trayBlink)
	defer refresh.Stop()
	defer blink.Stop()

	t.refresh() // premier cycle immédiat

	for {
		select {
		case <-t.stop:
			return
		case <-refresh.C:
			t.refresh()
		case <-blink.C:
			t.tick()
		}
	}
}

// refresh recalcule l'état, met à jour l'icône, le menu et les notifications.
func (t *tray) refresh() {
	tasks, err := t.app.tasks.ListAll()
	if err != nil {
		// Une lecture qui échoue ne doit pas tuer la boucle : l'icône reste
		// telle quelle et le prochain cycle réessaiera.
		log.Printf("[tray] CYCLE lecture des tâches impossible : %v", err)
		return
	}
	state := stats.Tray(tasks, t.app.clock.Now())
	top := stats.TopPriority(tasks, trayTopN)

	t.mu.Lock()
	t.cycles++
	t.state = state
	t.slotTasks = top
	t.inverted = false
	t.mu.Unlock()

	systray.SetIcon(TrayIcon(iconFor(state.Level)))
	systray.SetTooltip(tooltip(state))
	t.majLibelles(top)
	t.notifyUrgent(state)
}

// majLibelles réécrit les entrées de menu dont le texte a changé (§2.10).
//
// La comparaison au texte courant n'est pas une optimisation : écrire dans le
// menu Windows est l'opération la plus délicate de ce fichier, et sur une liste
// de tâches stable elle devient presque toujours inutile. Moins on y touche,
// moins on s'expose.
func (t *tray) majLibelles(top []domain.Task) {
	for i, item := range t.slots {
		voulu := placeholderVide
		if i < len(top) {
			voulu = t.libelle(top[i])
		}
		if t.libelles[i] == voulu {
			continue
		}
		t.libelles[i] = voulu
		item.SetTitle(voulu)
	}
}

// libelle rend une entrée de menu « [Projet] Nom (dans 30 min) » (§2.10).
func (t *tray) libelle(task domain.Task) string {
	projet := ""
	if p, err := t.app.projects.Get(task.ProjectID); err == nil {
		projet = "[" + p.Name + "] "
	}
	if info := duedate.Format(task.DueDate, t.app.clock.Now()); info != nil {
		return fmt.Sprintf("%s%s (%s)", projet, task.Name, info.Text)
	}
	return projet + task.Name
}

// tick fait avancer le clignotement d'une demi-période (§2.10).
//
// L'alternance se fait entre l'icône d'horloge et celle du niveau courant —
// normale, haute ou critique. L'horloge est la même que celle affichée dans
// l'arbre et la sidebar : elle signale le temps, le niveau signale l'importance.
func (t *tray) tick() {
	t.mu.Lock()
	if !t.state.Blinking {
		t.mu.Unlock()
		return
	}
	t.inverted = !t.inverted
	niveau, inverse := t.state.Level, t.inverted
	t.mu.Unlock()

	if inverse {
		systray.SetIcon(TrayIcon(TrayClock))
		return
	}
	systray.SetIcon(TrayIcon(iconFor(niveau)))
}

// tooltip rend le texte de l'infobulle, avec les compteurs du §2.10.
func tooltip(s stats.TrayState) string {
	if s.CriticalCount == 0 && s.HighCount == 0 {
		return "Tout Doux"
	}
	return fmt.Sprintf("Tout Doux — %d critique(s), %d haute(s)", s.CriticalCount, s.HighCount)
}

// iconFor associe un niveau à son icône embarquée.
func iconFor(level stats.TrayLevel) string {
	switch level {
	case stats.TrayLevelCritical:
		return TrayCritical
	case stats.TrayLevelHigh:
		return TrayHigh
	default:
		return TrayNormal
	}
}

// notifyUrgent signale les tâches devenues urgentes (§2.10).
//
// Le suivi des tâches déjà notifiées est indispensable : sans lui, le cycle de
// 30 secondes rejouerait la même notification dix fois pendant les cinq minutes
// où la tâche reste urgente.
func (t *tray) notifyUrgent(state stats.TrayState) {
	if t.app.ctx == nil {
		return
	}

	encoreUrgentes := make(map[string]bool, len(state.TasksToNotify))
	for _, task := range state.TasksToNotify {
		encoreUrgentes[task.ID] = true
		if t.notified[task.ID] {
			continue
		}
		t.notified[task.ID] = true

		projet := ""
		if p, err := t.app.projects.Get(task.ProjectID); err == nil {
			projet = "[" + p.Name + "] "
		}
		corps := projet + task.Name
		if info := duedate.Format(task.DueDate, t.app.clock.Now()); info != nil {
			corps = fmt.Sprintf("%s (%s)", corps, info.Text)
		}
		_ = wailsruntime.SendNotification(t.app.ctx, wailsruntime.NotificationOptions{
			ID:    "urgent-" + task.ID,
			Title: "Tâche Urgente !",
			Body:  corps,
		})
	}

	// Oubli des tâches qui ne sont plus urgentes : une échéance repoussée puis
	// redevenue imminente doit pouvoir notifier de nouveau.
	for id := range t.notified {
		if !encoreUrgentes[id] {
			delete(t.notified, id)
		}
	}
}

/* ---- Actions déclenchées par un clic ----

Toutes s'exécutent en goroutine, jamais dans la boucle de messages du tray :
les gestionnaires de la librairie sont appelés de façon synchrone depuis la
procédure de fenêtre, et un appel bloquant y figerait l'icône. */

// enGoroutine exécute une action hors de la boucle de messages, en journalisant
// son entrée et sa sortie.
//
// Un « début » sans « fin » dans le journal désigne l'appel qui bloque.
func (t *tray) enGoroutine(nom string, action func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[tray] PANIQUE dans %s : %v", nom, r)
			}
		}()
		log.Printf("[tray] %s : début", nom)
		action()
		log.Printf("[tray] %s : fin", nom)
	}()
}

// showWindow ramène la fenêtre au premier plan (§2.10).
//
// WindowUnminimise en plus de WindowShow : la fenêtre peut être réduite dans la
// barre des tâches, et l'afficher sans la restaurer ne ferait rien de visible.
func (t *tray) showWindow() {
	if t.app.ctx == nil {
		return
	}
	wailsruntime.WindowUnminimise(t.app.ctx)
	wailsruntime.WindowShow(t.app.ctx)
}

func (t *tray) quitApp() {
	if t.app.ctx == nil {
		return
	}
	wailsruntime.Quit(t.app.ctx)
}

// openSlot ramène la fenêtre et demande au frontend d'ouvrir la tâche de
// l'emplacement de menu cliqué (§2.10).
//
// La tâche est relue au moment du clic, et non capturée à la construction du
// menu : les entrées sont permanentes, leur contenu change à chaque cycle.
func (t *tray) openSlot(index int) {
	t.mu.Lock()
	var task domain.Task
	if index < len(t.slotTasks) {
		task = t.slotTasks[index]
	}
	t.mu.Unlock()

	if task.ID == "" || t.app.ctx == nil {
		return
	}
	t.showWindow()
	wailsruntime.EventsEmit(t.app.ctx, "open-task", map[string]string{
		"projectId": task.ProjectID,
		"taskId":    task.ID,
	})
}

// etat rend un résumé compact joint à chaque ligne de journal.
func (t *tray) etat() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return fmt.Sprintf("[niveau=%s clignote=%v cycles=%d]",
		t.state.Level, t.state.Blinking, t.cycles)
}
