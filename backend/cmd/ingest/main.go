// Command ingest indexes clinical knowledge base chunks (CREFITO guidelines, protocols)
// into the pgvector store for RAG retrieval. Run once after initial setup or when sources change.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	openai "github.com/sashabaranov/go-openai"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/pradella/voz-clinica/internal/core"
	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/rag"
)

func main() {
	ctx := context.Background()

	cfg, err := core.Load()
	if err != nil {
		slog.Error("load config", "err", err)
		os.Exit(1)
	}

	pool, err := core.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	ragStore := rag.NewStore(pool)
	openaiClient := openai.NewClient(cfg.OpenAIKey)

	sources := defaultSources()
	slog.Info("ingesting clinical sources", "count", len(sources))

	for _, src := range sources {
		embedding, err := embed(ctx, openaiClient, src.ChunkText)
		if err != nil {
			slog.Error("embed chunk", "title", src.Title, "err", err)
			continue
		}
		src.Embedding = pgvector.NewVector(embedding)

		if err := ragStore.Upsert(ctx, src); err != nil {
			slog.Error("upsert source", "title", src.Title, "err", err)
			continue
		}
		slog.Info("indexed chunk", "origin", src.Origin, "title", src.Title)
	}

	slog.Info("ingestion complete")
}

// embed generates a text embedding using OpenAI's text-embedding-3-large model.
func embed(ctx context.Context, client *openai.Client, text string) (pgv, error) {
	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel("text-embedding-3-large"),
	})
	if err != nil {
		return nil, fmt.Errorf("create embedding: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	// Convert []float64 to []float32
	raw := resp.Data[0].Embedding
	result := make([]float32, len(raw))
	for i, v := range raw {
		result[i] = float32(v)
	}
	return result, nil
}

type pgv = []float32

// defaultSources returns the initial set of CREFITO/protocol chunks to ingest.
// In production these would be loaded from files in a knowledge base directory.
func defaultSources() []models.ClinicalSource {
	return []models.ClinicalSource{
		{
			Title:     "Diretrizes CREFITO: Fisioterapia Ortopédica",
			Origin:    "CREFITO",
			Version:   "2024",
			ChunkText: "A avaliação fisioterapêutica ortopédica inclui inspeção postural, amplitude de movimento articular (goniometria), testes de força muscular (escala MRC), testes especiais ortopédicos e avaliação funcional. O diagnóstico fisioterapêutico deve ser baseado nos achados clínicos e deve diferenciar a disfunção do diagnóstico médico.",
		},
		{
			Title:     "Diretrizes CREFITO: Documentação Clínica",
			Origin:    "CREFITO",
			Version:   "2024",
			ChunkText: "A evolução clínica em fisioterapia deve seguir o formato SOAP: S (Subjetivo) — relato do paciente sobre sintomas e queixas; O (Objetivo) — achados mensuráveis do exame físico; A (Avaliação) — interpretação clínica do fisioterapeuta; P (Plano) — condutas terapêuticas programadas. Não inclua PII do paciente sem consentimento.",
		},
		{
			Title:     "Protocolo de Reabilitação Lombar",
			Origin:    "protocolo-lombar",
			Version:   "2023",
			ChunkText: "Lombalgia inespecífica: tratamento conservador com exercícios de estabilização lombar (Pilates clínico, McKenzie), terapia manual, eletroterapia analgésica (TENS, ultrassom) e orientação postural. CID-10 M54.5. Educação em dor é componente essencial do plano terapêutico.",
		},
		{
			Title:     "Protocolo de Reabilitação do Joelho",
			Origin:    "protocolo-joelho",
			Version:   "2023",
			ChunkText: "Lesão de LCA: fases de reabilitação pós-cirúrgica — fase 1 (0-6 semanas): controle de edema, amplitude de movimento, ativação do quadríceps; fase 2 (6-12 semanas): fortalecimento progressivo, propriocepção; fase 3 (3-6 meses): retorno funcional esportivo. CID-10 M23.6.",
		},
		{
			Title:     "Protocolo de Fisioterapia Respiratória",
			Origin:    "protocolo-respiratorio",
			Version:   "2024",
			ChunkText: "Fisioterapia respiratória: técnicas de higiene brônquica (drenagem postural, PEP, Flutter), reexpansão pulmonar (IPPB, incentivadores), treinamento muscular respiratório (threshold). Indicações: DPOC, bronquiectasias, fibrose cística, pós-operatório torácico/abdominal.",
		},
	}
}
