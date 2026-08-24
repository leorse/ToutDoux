// Package duedate porte les échéances intelligentes (§2.3).
//
// Toutes les fonctions reçoivent `now` en paramètre plutôt que d'appeler
// time.Now() : c'est ce qui rend testables les règles sensibles à l'heure
// (urgence à moins de 5 minutes, vendredi qui bascule au lundi). En production
// `now` vient du port Clock (§3.9).
package duedate

import (
	"fmt"
	"math"
	"time"
)

// Kind est un raccourci d'échéance proposé dans le détail d'une tâche (§2.3).
type Kind string

const (
	KindNow         Kind = "now"         // "Tout de suite"
	Kind10Min       Kind = "10min"       // "Dans 10 min"
	Kind1H          Kind = "1h"          // "Dans 1h"
	KindJournee     Kind = "journee"     // "Journée" → 18h
	KindLendemain9H Kind = "lendemain9h" // "Lendemain 9h", week-end sauté
)

// Info est le rendu d'une échéance relative à un instant donné.
//
// Urgent et Overdue ne sont jamais vrais en même temps, et c'est délibéré :
// Urgent signifie "bientôt" (moins de 5 minutes), Overdue signifie "trop tard".
// Les appelants testent `Urgent || Overdue` pour décider de l'icône ⏰ (§2.3) ;
// les fusionner en un seul booléen perdrait la distinction.
type Info struct {
	Text    string `json:"text"`
	Urgent  bool   `json:"urgent"`
	Overdue bool   `json:"overdue"`
}

// UrgentThreshold est le seuil sous lequel une échéance est dite urgente (§2.3).
const UrgentThreshold = 5 * time.Minute

// StartOfDay renvoie minuit du jour de t, dans son fuseau.
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// atTime renvoie le jour de t à l'heure h:m précise.
func atTime(t time.Time, h, m int) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, h, m, 0, 0, t.Location())
}

// sameDay indique si a et b tombent le même jour calendaire.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// daysBetween compte les jours calendaires séparant a de b.
//
// L'arrondi sur des journées tronquées à minuit absorbe les changements
// d'heure : un jour de basculement fait 23 ou 25 heures, pas 24.
func daysBetween(a, b time.Time) int {
	diff := StartOfDay(b).Sub(StartOfDay(a))
	return int(math.Round(diff.Hours() / 24))
}

// NextWeekday9h renvoie le lendemain de `from` à 9h, en sautant le week-end.
//
// Un vendredi rend donc le lundi suivant, un samedi le lundi également (§2.3).
func NextWeekday9h(from time.Time) time.Time {
	d := from.AddDate(0, 0, 1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return atTime(d, 9, 0)
}

// Quick calcule l'échéance correspondant à un raccourci rapide (§2.3).
//
// Renvoie nil si le raccourci est inconnu.
//
// Note : "Journée" rend 18h sans condition. La désactivation du bouton après
// 17h est une règle d'affichage (§2.3), elle appartient à l'UI et n'a pas sa
// place ici — l'y remonter ferait échouer toute échéance fixée en soirée.
func Quick(kind Kind, now time.Time) *time.Time {
	var t time.Time
	switch kind {
	case KindNow:
		t = now
	case Kind10Min:
		t = now.Add(10 * time.Minute)
	case Kind1H:
		t = now.Add(time.Hour)
	case KindJournee:
		t = atTime(now, 18, 0)
	case KindLendemain9H:
		t = NextWeekday9h(now)
	default:
		return nil
	}
	return &t
}

// Format rend le temps restant avant une échéance (§2.3).
//
// Renvoie nil si la tâche n'a pas d'échéance, ce qui laisse l'appelant
// distinguer "pas d'échéance" de "échéance à l'instant".
func Format(due *time.Time, now time.Time) *Info {
	if due == nil {
		return nil
	}
	diff := due.Sub(now)

	if diff < 0 {
		overdue := -diff
		if hours := ceilDiv(overdue, time.Hour); hours < 24 {
			return &Info{Text: fmt.Sprintf("En retard de %dh", hours), Overdue: true}
		}
		days := ceilDiv(overdue, 24*time.Hour)
		return &Info{Text: fmt.Sprintf("En retard de %dj", days), Overdue: true}
	}

	if sameDay(*due, now) {
		if diff < time.Hour {
			min := int(math.Round(diff.Minutes()))
			if min < 1 {
				min = 1
			}
			return &Info{Text: fmt.Sprintf("dans %d min", min), Urgent: diff < UrgentThreshold}
		}
		if diff < 2*time.Hour {
			halfHours := int(math.Round(diff.Minutes() / 30))
			h := halfHours / 2
			if halfHours%2 == 1 {
				return &Info{Text: fmt.Sprintf("~%dh30", h)}
			}
			return &Info{Text: fmt.Sprintf("~%dh", h)}
		}
		return &Info{Text: fmt.Sprintf("~%dh", int(math.Round(diff.Hours())))}
	}

	if sameDay(*due, now.AddDate(0, 0, 1)) {
		return &Info{Text: "Demain"}
	}

	days := daysBetween(now, *due)
	if days <= 6 {
		return &Info{Text: fmt.Sprintf("dans %d jours", days)}
	}
	if days <= 13 {
		return &Info{Text: "Semaine prochaine"}
	}

	weeks, rem := days/7, days%7
	text := fmt.Sprintf("%d semaine%s", weeks, plural(weeks))
	if rem > 0 {
		text += fmt.Sprintf(" et %d jour%s", rem, plural(rem))
	}
	return &Info{Text: text}
}

// ceilDiv divise deux durées en arrondissant au supérieur.
func ceilDiv(d, unit time.Duration) int {
	return int(math.Ceil(float64(d) / float64(unit)))
}

func plural(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
