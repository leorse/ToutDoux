package main

import (
	"strings"

	"github.com/google/uuid"

	"toutdoux/domain"
)

/* ---------------- Réunions (§2.7) ---------------- */

// GetMeetings rend les réunions d'un projet.
func (a *App) GetMeetings(projectID string) ([]domain.Meeting, error) {
	return a.meetings.ListByProject(projectID)
}

// CreateMeeting crée une réunion dans un projet.
func (a *App) CreateMeeting(projectID, title string) (domain.Meeting, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return domain.Meeting{}, domain.ErrEmptyName
	}
	if _, err := a.projects.Get(projectID); err != nil {
		return domain.Meeting{}, err
	}
	now := a.clock.Now()
	m := domain.Meeting{ID: uuid.NewString(), ProjectID: projectID, Title: title, CreatedAt: now, UpdatedAt: now}
	if err := a.meetings.Create(m); err != nil {
		return domain.Meeting{}, err
	}
	return m, a.indexMeeting(m)
}

// RenameMeeting renomme une réunion.
func (a *App) RenameMeeting(meetingID, newTitle string) (domain.Meeting, error) {
	newTitle = strings.TrimSpace(newTitle)
	if newTitle == "" {
		return domain.Meeting{}, domain.ErrEmptyName
	}
	if err := a.meetings.Rename(meetingID, newTitle); err != nil {
		return domain.Meeting{}, err
	}
	m, err := a.meetings.Get(meetingID)
	if err != nil {
		return domain.Meeting{}, err
	}
	return m, a.indexMeeting(m)
}

// DeleteMeeting supprime une réunion et ses instances (§2.7).
func (a *App) DeleteMeeting(meetingID string) error {
	instances, err := a.meetings.ListInstances(meetingID)
	if err != nil {
		return err
	}
	if err := a.meetings.Delete(meetingID); err != nil {
		return err
	}
	// L'index se nettoie après coup : une entrée orpheline ferait remonter un
	// résultat qu'on ne peut plus ouvrir.
	for _, i := range instances {
		if err := a.index.Delete(i.ID); err != nil {
			return err
		}
		if err := a.embeddings.Delete(i.ID); err != nil {
			return err
		}
	}
	if err := a.embeddings.Delete(meetingID); err != nil {
		return err
	}
	return a.index.Delete(meetingID)
}

// GetInstances rend les instances d'une réunion, la plus récente d'abord (§2.7).
func (a *App) GetInstances(meetingID string) ([]domain.MeetingInstance, error) {
	return a.meetings.ListInstances(meetingID)
}

// AddMeetingInstance crée une instance datée de maintenant (§2.7).
//
// Le timestamp vient de l'horloge injectée : c'est ce qui rend le tri des
// instances testable sans dépendre de l'heure réelle.
func (a *App) AddMeetingInstance(meetingID string) (domain.MeetingInstance, error) {
	if _, err := a.meetings.Get(meetingID); err != nil {
		return domain.MeetingInstance{}, err
	}
	now := a.clock.Now()
	i := domain.MeetingInstance{
		ID: uuid.NewString(), MeetingID: meetingID,
		Timestamp: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := a.meetings.CreateInstance(i); err != nil {
		return domain.MeetingInstance{}, err
	}
	return i, a.indexInstance(i)
}

// UpdateInstanceNotes écrit le compte rendu d'une instance.
//
// C'est la cible de la sauvegarde automatique des réunions, avec la même
// fiabilité qu'en §2.6 : délai de 2 s côté frontend, plus vidage immédiat au
// changement d'instance, de réunion ou de projet.
func (a *App) UpdateInstanceNotes(instanceID, notes string) (domain.MeetingInstance, error) {
	if err := a.meetings.UpdateInstanceNotes(instanceID, notes); err != nil {
		return domain.MeetingInstance{}, err
	}
	i, err := a.meetings.GetInstance(instanceID)
	if err != nil {
		return domain.MeetingInstance{}, err
	}
	// Revectorisation en arrière-plan si l'instance est dans l'index sémantique
	// (§2.12). Sans effet sinon, et sans effet si le texte n'a pas bougé.
	a.rafraichirEnArrierePlan(instanceID)
	return i, a.indexInstance(i)
}

// DeleteInstance supprime une instance.
func (a *App) DeleteInstance(instanceID string) error {
	if err := a.meetings.DeleteInstance(instanceID); err != nil {
		return err
	}
	if err := a.embeddings.Delete(instanceID); err != nil {
		return err
	}
	return a.index.Delete(instanceID)
}
