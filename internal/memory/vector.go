package memory

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// EmbeddingProvider generates vector embeddings for text. When nil (no provider
// configured on the store), the system gracefully degrades to pure lexical
// search — see SearchHybrid. This interface and the helpers below are
// infrastructure for future semantic search; nothing here is wired into the
// hot path yet, so an absent provider has zero cost.
type EmbeddingProvider interface {
	// Embed returns a float32 vector representation of text.
	// Implementations must be safe for concurrent use.
	Embed(ctx context.Context, text string) ([]float32, error)

	// Model identifies the embedding model version, used as the storage key
	// so vectors from different models are not silently mixed.
	Model() string
}

// ensureEmbeddingsTable creates the embeddings table if it doesn't exist.
// The table is keyed by file_id (one vector per file) and tagged with the
// model that produced it, so swapping models requires re-embedding rather
// than overwriting.
func ensureEmbeddingsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS embeddings (
			file_id    INTEGER NOT NULL,
			embedding  BLOB NOT NULL,
			model      TEXT NOT NULL DEFAULT 'default',
			created_at TEXT NOT NULL,
			FOREIGN KEY (file_id) REFERENCES file_index(id),
			UNIQUE(file_id)
		)
	`)
	return err
}

// storeEmbedding upserts an embedding for a file under the given model.
func storeEmbedding(db *sql.DB, fileID int64, vec []float32, model string) error {
	_, err := db.Exec(`
		INSERT INTO embeddings (file_id, embedding, model, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(file_id) DO UPDATE SET
			embedding=excluded.embedding,
			model=excluded.model,
			created_at=excluded.created_at
	`, fileID, serializeFloat32(vec), model, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("store embedding: %w", err)
	}
	return nil
}

// serializeFloat32 converts a float32 slice to little-endian bytes for BLOB
// storage. Each element occupies exactly 4 bytes.
func serializeFloat32(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// deserializeFloat32 converts BLOB bytes back to a float32 slice. Returns nil
// if buf length is not a multiple of 4 (corrupt/truncated blob).
func deserializeFloat32(buf []byte) []float32 {
	if len(buf)%4 != 0 {
		return nil
	}
	vec := make([]float32, len(buf)/4)
	for i := range vec {
		vec[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return vec
}

// cosineSimilarity computes cosine similarity between two float32 vectors,
// returning a value in [-1, 1]. Returns 0 for mismatched lengths, empty
// vectors, or zero-norm vectors (which would otherwise divide by zero).
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
