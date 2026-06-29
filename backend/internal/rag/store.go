package rag

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/pradella/voz-clinica/internal/models"
)

// Store provides access to clinical knowledge base chunks via pgvector similarity search.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Upsert inserts or replaces a clinical source chunk with its embedding.
func (s *Store) Upsert(ctx context.Context, src models.ClinicalSource) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO clinical_sources (title, origin, version, chunk_text, embedding, updated_at)
		 VALUES ($1, $2, $3, $4, $5, now())
		 ON CONFLICT (id) DO UPDATE
		   SET title = EXCLUDED.title,
		       chunk_text = EXCLUDED.chunk_text,
		       embedding = EXCLUDED.embedding,
		       updated_at = now()`,
		src.Title, src.Origin, src.Version, src.ChunkText, src.Embedding,
	)
	if err != nil {
		return fmt.Errorf("upsert clinical source: %w", err)
	}
	return nil
}

// SimilarChunks returns the top-k most similar chunks to the given query embedding
// using cosine distance (pgvector).
func (s *Store) SimilarChunks(ctx context.Context, queryEmbedding []float32, k int) ([]models.ClinicalSource, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, title, origin, version, chunk_text, updated_at
		 FROM clinical_sources
		 ORDER BY embedding <=> $1
		 LIMIT $2`,
		pgvector.NewVector(queryEmbedding), k,
	)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}
	defer rows.Close()

	var results []models.ClinicalSource
	for rows.Next() {
		var src models.ClinicalSource
		if err := rows.Scan(&src.ID, &src.Title, &src.Origin, &src.Version, &src.ChunkText, &src.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan clinical source: %w", err)
		}
		results = append(results, src)
	}
	return results, rows.Err()
}

// ChunkTexts returns only the text content from a list of sources (for LLM context injection).
func ChunkTexts(sources []models.ClinicalSource) []string {
	texts := make([]string, len(sources))
	for i, s := range sources {
		texts[i] = fmt.Sprintf("[%s v%s] %s", s.Origin, s.Version, s.ChunkText)
	}
	return texts
}
