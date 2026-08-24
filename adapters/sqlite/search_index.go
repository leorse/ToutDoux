package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"toutdoux/domain"
)

// SearchIndexAdapter implémente ports.SearchIndex sur la table FTS5
// `search_index` du §3.2.
type SearchIndexAdapter struct{ db *DB }

// NewSearchIndexAdapter câble l'index sur une base ouverte.
func NewSearchIndexAdapter(db *DB) *SearchIndexAdapter { return &SearchIndexAdapter{db: db} }

// Put insère ou remplace l'entrée d'une entité.
//
// FTS5 n'a pas de clé primaire ni de UPSERT : on efface puis on réinsère, dans
// une transaction pour qu'une entité ne disparaisse jamais de l'index entre les
// deux.
func (s *SearchIndexAdapter) Put(e domain.IndexEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM search_index WHERE entity_id = ?`, e.EntityID); err != nil {
		return fmt.Errorf("nettoyage de l'entrée d'index : %w", err)
	}
	var created any
	if e.CreatedAt != nil {
		created = formatTime(*e.CreatedAt)
	}
	if _, err := tx.Exec(
		`INSERT INTO search_index (type, title, content, project_id, entity_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		string(e.Type), e.Title, e.Content, e.ProjectID, e.EntityID, created,
	); err != nil {
		return fmt.Errorf("indexation : %w", err)
	}
	return tx.Commit()
}

// Delete retire une entité de l'index, sans erreur si elle n'y figurait pas.
func (s *SearchIndexAdapter) Delete(entityID string) error {
	if _, err := s.db.Exec(`DELETE FROM search_index WHERE entity_id = ?`, entityID); err != nil {
		return fmt.Errorf("suppression de l'entrée d'index : %w", err)
	}
	return nil
}

// DeleteByProject retire toutes les entrées d'un projet (§2.1).
func (s *SearchIndexAdapter) DeleteByProject(projectID string) error {
	if _, err := s.db.Exec(`DELETE FROM search_index WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("purge de l'index du projet : %w", err)
	}
	return nil
}

// Search interroge l'index plein texte.
//
// Rend une tranche vide plutôt qu'une erreur sur une requête vide : c'est le
// cas normal quand l'utilisateur efface la barre de recherche.
func (s *SearchIndexAdapter) Search(query string) ([]domain.IndexEntry, error) {
	expr := toMatchExpression(query)
	if expr == "" {
		return []domain.IndexEntry{}, nil
	}

	rows, err := s.db.Query(
		`SELECT type, title, content, project_id, entity_id, created_at
		 FROM search_index WHERE search_index MATCH ?`, expr,
	)
	if err != nil {
		return nil, fmt.Errorf("recherche : %w", err)
	}
	defer rows.Close()

	entries := []domain.IndexEntry{}
	for rows.Next() {
		var (
			e       domain.IndexEntry
			typ     string
			content sql.NullString
			created sql.NullString
		)
		if err := rows.Scan(&typ, &e.Title, &content, &e.ProjectID, &e.EntityID, &created); err != nil {
			return nil, err
		}
		e.Type = domain.SearchType(typ)
		e.Content = content.String
		if e.CreatedAt, err = scanNullTime(created); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// toMatchExpression traduit une saisie utilisateur en expression FTS5.
//
// La syntaxe MATCH est un langage à part entière : une apostrophe, un tiret ou
// un guillemet dans « aujourd'hui » ou « mi-parcours » produisent une erreur de
// syntaxe, pas zéro résultat. Chaque mot est donc entouré de guillemets et
// suffixé de `*` pour la recherche par préfixe, ce qui est le comportement
// attendu pendant la frappe (§2.9).
func toMatchExpression(query string) string {
	champs := strings.FieldsFunc(strings.TrimSpace(query), func(r rune) bool {
		// On ne conserve que lettres et chiffres : tout le reste est de la
		// ponctuation, qui n'a pas de sens dans une requête plein texte.
		return !isAlphanumeric(r)
	})
	if len(champs) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(champs))
	for _, mot := range champs {
		quoted = append(quoted, `"`+mot+`"*`)
	}
	// Les termes sont conjonctifs : « client feedback » cherche les documents
	// contenant les deux, pas l'un ou l'autre (§4).
	return strings.Join(quoted, " AND ")
}

func isAlphanumeric(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r > 127:
		// Les lettres accentuées sont hors ASCII et doivent être conservées :
		// « réunion » et « échéance » sont partout dans ce corpus.
		return true
	}
	return false
}
