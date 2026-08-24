package duedate

import (
	"testing"
	"time"
)

// ref est un mercredi, pour que les décalages de jours restent en semaine.
var ref = time.Date(2026, 8, 19, 10, 0, 0, 0, time.UTC)

func at(d time.Time) *time.Time { return &d }

func TestFormat_NilDue(t *testing.T) {
	if got := Format(nil, ref); got != nil {
		t.Fatalf("une tâche sans échéance doit rendre nil, obtenu %+v", got)
	}
}

func TestFormat_Overdue(t *testing.T) {
	cases := []struct {
		name string
		due  time.Time
		want string
	}{
		{"deux heures de retard", ref.Add(-2 * time.Hour), "En retard de 2h"},
		{"trois jours de retard", ref.Add(-72 * time.Hour), "En retard de 3j"},
		{"juste dépassée", ref.Add(-time.Second), "En retard de 1h"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Format(at(c.due), ref)
			if got.Text != c.want {
				t.Errorf("texte = %q, attendu %q", got.Text, c.want)
			}
			if !got.Overdue {
				t.Error("Overdue doit être vrai")
			}
			// §2.3 : les deux drapeaux sont disjoints.
			if got.Urgent {
				t.Error("Urgent doit rester faux sur une échéance dépassée")
			}
		})
	}
}

func TestFormat_UrgentSeparateFromOverdue(t *testing.T) {
	got := Format(at(ref.Add(3*time.Minute)), ref)
	if !got.Urgent {
		t.Error("moins de 5 min doit être urgent")
	}
	if got.Overdue {
		t.Error("une échéance à venir n'est pas en retard")
	}
	if got.Text != "dans 3 min" {
		t.Errorf("texte = %q", got.Text)
	}
}

func TestFormat_UrgentThresholdBoundary(t *testing.T) {
	if Format(at(ref.Add(UrgentThreshold)), ref).Urgent {
		t.Error("exactement 5 min ne doit pas être urgent (seuil strict)")
	}
	if !Format(at(ref.Add(UrgentThreshold-time.Second)), ref).Urgent {
		t.Error("juste sous 5 min doit être urgent")
	}
}

func TestFormat_SameDay(t *testing.T) {
	cases := []struct {
		delta time.Duration
		want  string
	}{
		{90 * time.Minute, "~1h30"},
		{60 * time.Minute, "~1h"},
		{4 * time.Hour, "~4h"},
	}
	for _, c := range cases {
		if got := Format(at(ref.Add(c.delta)), ref); got.Text != c.want {
			t.Errorf("+%v : texte = %q, attendu %q", c.delta, got.Text, c.want)
		}
	}
}

func TestFormat_FutureDays(t *testing.T) {
	cases := []struct {
		name string
		due  time.Time
		want string
	}{
		{"demain", ref.AddDate(0, 0, 1), "Demain"},
		{"dans trois jours", ref.AddDate(0, 0, 3), "dans 3 jours"},
		{"semaine prochaine", ref.AddDate(0, 0, 9), "Semaine prochaine"},
		{"deux semaines et trois jours", ref.AddDate(0, 0, 17), "2 semaines et 3 jours"},
		{"deux semaines pile", ref.AddDate(0, 0, 14), "2 semaines"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Format(at(c.due), ref); got.Text != c.want {
				t.Errorf("texte = %q, attendu %q", got.Text, c.want)
			}
		})
	}
}

// "Demain" doit se lire sur le calendrier, pas sur 24h glissantes :
// ce soir 23h et demain 00h30 ne sont séparés que d'une heure et demie.
func TestFormat_TomorrowIsCalendarBased(t *testing.T) {
	late := time.Date(2026, 8, 19, 23, 0, 0, 0, time.UTC)
	justAfterMidnight := time.Date(2026, 8, 20, 0, 30, 0, 0, time.UTC)
	if got := Format(at(justAfterMidnight), late); got.Text != "Demain" {
		t.Errorf("texte = %q, attendu \"Demain\"", got.Text)
	}
}

func TestNextWeekday9h_SkipsWeekend(t *testing.T) {
	cases := []struct {
		name    string
		from    time.Time
		wantDay int
	}{
		{"mercredi donne jeudi", time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC), 20},
		{"vendredi donne lundi", time.Date(2026, 8, 21, 14, 0, 0, 0, time.UTC), 24},
		{"samedi donne lundi", time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC), 24},
		{"dimanche donne lundi", time.Date(2026, 8, 23, 14, 0, 0, 0, time.UTC), 24},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NextWeekday9h(c.from)
			if got.Day() != c.wantDay {
				t.Errorf("jour = %d, attendu %d", got.Day(), c.wantDay)
			}
			if got.Hour() != 9 || got.Minute() != 0 {
				t.Errorf("heure = %02d:%02d, attendu 09:00", got.Hour(), got.Minute())
			}
			if wd := got.Weekday(); wd == time.Saturday || wd == time.Sunday {
				t.Errorf("le résultat tombe un %v", wd)
			}
		})
	}
}

func TestQuick(t *testing.T) {
	if got := Quick(KindJournee, ref); got.Hour() != 18 || got.Minute() != 0 {
		t.Errorf("\"Journée\" = %02d:%02d, attendu 18:00", got.Hour(), got.Minute())
	}
	if got := Quick(Kind10Min, ref); !got.Equal(ref.Add(10 * time.Minute)) {
		t.Errorf("\"Dans 10 min\" = %v", got)
	}
	if got := Quick(KindNow, ref); !got.Equal(ref) {
		t.Errorf("\"Tout de suite\" = %v", got)
	}
	if got := Quick(Kind("inconnu"), ref); got != nil {
		t.Errorf("un raccourci inconnu doit rendre nil, obtenu %v", got)
	}
}

// §2.3 : "Journée" rend 18h même appelé après 17h. La désactivation du bouton
// est une règle d'UI, elle ne doit pas contaminer le domaine.
func TestQuick_JourneeIgnoresCutoff(t *testing.T) {
	evening := time.Date(2026, 8, 19, 19, 30, 0, 0, time.UTC)
	got := Quick(KindJournee, evening)
	if got.Hour() != 18 {
		t.Errorf("heure = %d, attendu 18 sans condition", got.Hour())
	}
}
