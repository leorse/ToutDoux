package main

import (
	"fmt"
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

Ce fichier est un adaptateur pilote, au même titre que app.go : il ne décide
rien. Le niveau de l'icône, le clignotement et la liste des tâches urgentes sont
calculés par stats.Tray ; ce code se contente de les afficher et de réagir aux
clics.
*/

const (
	// Rythme de recalcul du menu et de l'icône, aligné sur celui de l'interface
	// (§2.3) : les deux doivent montrer la même chose au même moment.
	trayRefresh = 30 * time.Second

	// Demi-période du clignotement. Assez lent pour ne pas être fatigant,
	// assez rapide pour se remarquer du coin de l'œil.
	trayBlink = 700 * time.Millisecond

	// Nombre d'entrées de tâches dans le menu (§2.10, « Top 5 »).
	trayTopN = 5
)

// tray porte l'état de l'icône de barre système.
type tray struct {
	app *App

	mu       sync.Mutex
	state    stats.TrayState
	tasks    []domain.Task
	inverted bool // phase courante du clignotement

	// notified retient les tâches déjà signalées, pour ne pas répéter la même
	// notification à chaque rafraîchissement. Une tâche en sort dès qu'elle
	// n'est plus urgente, ce qui la rend de nouveau notifiable si son échéance
	// est repoussée puis redevient imminente.
	notified map[string]bool

	// entrées de menu conservées pour être reconstruites à chaque cycle
	quit *systray.MenuItem

	stop chan struct{}
}

// startTray démarre la barre système (§2.10).
//
// systray.Run bloque, exactement comme wails.Run : les deux ne peuvent pas
// occuper le même fil, d'où la goroutine.
func (a *App) startTray() {
	a.tray = &tray{app: a, notified: map[string]bool{}, stop: make(chan struct{})}
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

	// Double-clic sur l'icône : affiche ou masque la fenêtre (§2.10).
	systray.SetOnDClick(func(systray.IMenu) { t.toggleWindow() })

	t.rebuildMenu()
	t.refresh()

	go t.loop()
}

// loop entretient l'icône et le menu.
//
// Un seul fil pilote la barre système : les deux tickers y sont multiplexés
// plutôt que lancés séparément, ce qui évite d'avoir à synchroniser deux
// goroutines qui écriraient l'icône en même temps.
func (t *tray) loop() {
	refresh := time.NewTicker(trayRefresh)
	blink := time.NewTicker(trayBlink)
	defer refresh.Stop()
	defer blink.Stop()

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
		return
	}
	state := stats.Tray(tasks, t.app.clock.Now())

	t.mu.Lock()
	t.state = state
	t.tasks = tasks
	t.inverted = false
	t.mu.Unlock()

	systray.SetIcon(TrayIcon(iconFor(state.Level)))
	systray.SetTooltip(tooltip(state))
	t.rebuildMenu()
	t.notifyUrgent(state)
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
	} else {
		systray.SetIcon(TrayIcon(iconFor(niveau)))
	}
}

// rebuildMenu reconstruit le menu contextuel (§2.10).
//
// systray n'offre pas de mise à jour partielle : on repart d'un menu vide à
// chaque cycle. C'est acceptable à ce rythme, et bien plus simple que de tenir
// à jour une correspondance entre entrées de menu et tâches.
func (t *tray) rebuildMenu() {
	t.mu.Lock()
	state, tasks := t.state, t.tasks
	t.mu.Unlock()

	systray.ResetMenu()

	top := stats.TopPriority(tasks, trayTopN)
	if len(top) == 0 {
		vide := systray.AddMenuItem("Aucune tâche prioritaire", "")
		vide.Disable()
	}
	for _, task := range top {
		task := task // capture par valeur : sans cela, tous les clics ouvriraient la dernière
		item := systray.AddMenuItem(t.libelle(task), task.Name)
		item.Click(func() { t.openTask(task) })
	}

	systray.AddSeparator()

	critiques := systray.AddMenuItem(fmt.Sprintf("Tâches Critiques (%d)", state.CriticalCount), "")
	critiques.Disable()
	hautes := systray.AddMenuItem(fmt.Sprintf("Tâches Hautes (%d)", state.HighCount), "")
	hautes.Disable()

	systray.AddSeparator()

	afficher := systray.AddMenuItem("Afficher", "Afficher la fenêtre")
	afficher.Click(func() { t.showWindow() })

	t.quit = systray.AddMenuItem("Quitter", "Fermer Tout Doux")
	t.quit.Click(func() {
		// Seule sortie de l'application : le bouton X ne fait que masquer la
		// fenêtre (§2.10). Sans cette entrée, le processus serait impossible à
		// arrêter depuis l'interface.
		wailsruntime.Quit(t.app.ctx)
	})
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

func (t *tray) showWindow() {
	if t.app.ctx == nil {
		return
	}
	wailsruntime.WindowShow(t.app.ctx)
}

// toggleWindow affiche ou masque la fenêtre selon son état courant (§2.10).
func (t *tray) toggleWindow() {
	if t.app.ctx == nil {
		return
	}
	if wailsruntime.WindowIsMinimised(t.app.ctx) || !wailsruntime.WindowIsNormal(t.app.ctx) {
		wailsruntime.WindowShow(t.app.ctx)
		return
	}
	wailsruntime.WindowHide(t.app.ctx)
}

// openTask affiche la fenêtre et demande au frontend d'ouvrir la tâche (§2.10).
//
// Le backend ne sait pas naviguer : il émet un événement, et React fait le reste
// avec le même chemin que le double-clic depuis Priorités ou la Recherche.
func (t *tray) openTask(task domain.Task) {
	if t.app.ctx == nil {
		return
	}
	wailsruntime.WindowShow(t.app.ctx)
	wailsruntime.EventsEmit(t.app.ctx, "open-task", map[string]string{
		"projectId": task.ProjectID,
		"taskId":    task.ID,
	})
}
