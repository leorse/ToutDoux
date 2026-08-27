package sqlite

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"toutdoux/domain"
)

// EmbeddingRepository implémente ports.EmbeddingRepository sur la table
// `embeddings` du §3.2.
type EmbeddingRepository struct{ db *DB }

// NewEmbeddingRepository câble le repository sur une base ouverte.
func NewEmbeddingRepository(db *DB) *EmbeddingRepository { return &EmbeddingRepository{db: db} }

// Put insère ou remplace le vecteur d'une entité.
func (r *EmbeddingRepository) Put(e domain.Embedding) error {
	blob, err := encodeVector(e.Vector)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO embeddings (entity_id, type, project_id, vector, dimensions, source_hash, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(entity_id) DO UPDATE SET
		     type        = excluded.type,
		     project_id  = excluded.project_id,
		     vector      = excluded.vector,
		     dimensions  = excluded.dimensions,
		     source_hash = excluded.source_hash,
		     updated_at  = excluded.updated_at`,
		e.EntityID, string(e.Type), e.ProjectID, blob, len(e.Vector), e.SourceHash, formatTime(nowUTC()),
	)
	if err != nil {
		return fmt.Errorf("écriture du vecteur : %w", err)
	}
	return nil
}

// Get rend le vecteur d'une entité, ou domain.ErrNotFound.
func (r *EmbeddingRepository) Get(entityID string) (domain.Embedding, error) {
	row := r.db.QueryRow(
		`SELECT entity_id, type, project_id, vector, dimensions, source_hash, updated_at
		 FROM embeddings WHERE entity_id = ?`, entityID,
	)
	e, err := scanEmbedding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Embedding{}, domain.ErrNotFound
	}
	return e, err
}

// Delete retire une entité de l'index sémantique.
func (r *EmbeddingRepository) Delete(entityID string) error {
	if _, err := r.db.Exec(`DELETE FROM embeddings WHERE entity_id = ?`, entityID); err != nil {
		return fmt.Errorf("suppression du vecteur : %w", err)
	}
	return nil
}

// DeleteByProject retire tous les vecteurs d'un projet (§2.1).
func (r *EmbeddingRepository) DeleteByProject(projectID string) error {
	if _, err := r.db.Exec(`DELETE FROM embeddings WHERE project_id = ?`, projectID); err != nil {
		return fmt.Errorf("purge des vecteurs du projet : %w", err)
	}
	return nil
}

// List rend tout l'index sémantique.
func (r *EmbeddingRepository) List() ([]domain.Embedding, error) {
	rows, err := r.db.Query(
		`SELECT entity_id, type, project_id, vector, dimensions, source_hash, updated_at
		 FROM embeddings`,
	)
	if err != nil {
		return nil, fmt.Errorf("lecture de l'index sémantique : %w", err)
	}
	defer rows.Close()

	out := []domain.Embedding{}
	for rows.Next() {
		e, err := scanEmbedding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Count rend le nombre d'entités indexées.
func (r *EmbeddingRepository) Count() (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM embeddings`).Scan(&n); err != nil {
		return 0, fmt.Errorf("comptage de l'index sémantique : %w", err)
	}
	return n, nil
}

// scanner couvre *sql.Row et *sql.Rows, dont les Scan ont la même signature.
type scanner interface{ Scan(dest ...any) error }

func scanEmbedding(s scanner) (domain.Embedding, error) {
	var (
		e       domain.Embedding
		typ     string
		blob    []byte
		updated sql.NullString
	)
	if err := s.Scan(&e.EntityID, &typ, &e.ProjectID, &blob, &e.Dimensions, &e.SourceHash, &updated); err != nil {
		return domain.Embedding{}, err
	}
	e.Type = domain.SearchType(typ)

	vec, err := decodeVector(blob)
	if err != nil {
		return domain.Embedding{}, fmt.Errorf("vecteur de %s : %w", e.EntityID, err)
	}
	e.Vector = vec

	if t, err := scanNullTime(updated); err != nil {
		return domain.Embedding{}, err
	} else if t != nil {
		e.UpdatedAt = *t
	}
	return e, nil
}

// encodeVector sérialise un vecteur en BLOB.
//
// Little-endian explicite plutôt que l'ordre natif de la machine : la base est
// un fichier que l'utilisateur peut copier d'un poste à l'autre, et un vecteur
// relu à l'envers ne produirait pas une erreur mais des scores absurdes — le
// pire des deux mondes.
func encodeVector(v []float32) ([]byte, error) {
	if len(v) == 0 {
		return nil, errors.New("vecteur vide : rien à enregistrer")
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(v)*4))
	for _, f := range v {
		if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(f)); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// decodeVector relit un vecteur écrit par encodeVector.
func decodeVector(blob []byte) ([]float32, error) {
	if len(blob)%4 != 0 {
		return nil, fmt.Errorf("taille de %d octets, non multiple de 4", len(blob))
	}
	out := make([]float32, len(blob)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4 : i*4+4]))
	}
	return out, nil
}
