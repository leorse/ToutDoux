// Package adapters porte les implémentations concrètes des ports (§3.9).
package adapters

import (
	"sync"
	"time"
)

// SystemClock est l'horloge de production : elle lit l'heure du système.
type SystemClock struct{}

// Now rend l'heure courante.
func (SystemClock) Now() time.Time { return time.Now() }

// FixedClock est une horloge figée, pour les tests.
//
// Elle vit ici plutôt que dans un fichier _test.go parce que plusieurs packages
// en ont besoin, et qu'un helper de test n'est pas importable d'un package à
// l'autre en Go.
type FixedClock struct {
	mu sync.RWMutex
	at time.Time
}

// NewFixedClock rend une horloge arrêtée sur l'instant donné.
func NewFixedClock(at time.Time) *FixedClock { return &FixedClock{at: at} }

// Now rend l'instant figé.
func (c *FixedClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.at
}

// Set déplace l'horloge à un instant précis.
func (c *FixedClock) Set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = at
}

// Advance fait avancer l'horloge, pour observer le passage d'un seuil sans
// attendre réellement — le franchissement des 5 minutes d'urgence, typiquement.
func (c *FixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}
