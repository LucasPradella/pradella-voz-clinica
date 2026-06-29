package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	openai "github.com/sashabaranov/go-openai"

	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/rag"
)

const embeddingModel = "text-embedding-ada-002" // 1536 dims — matches VECTOR(1536) schema

// ProcessResult is the output of the full audio→SOAP pipeline.
type ProcessResult struct {
	Transcription string
	SOAP          *SOAPOutput
	SourceRefs    []models.SourceRef
}

// SOAPService orchestrates: Whisper transcription → RAG retrieval → Claude SOAP generation.
type SOAPService struct {
	transcription *TranscriptionService
	claude        *ClaudeSOAPClient
	ragStore      *rag.Store
	embedFn       func(ctx context.Context, text string) ([]float32, error)
}

// NewSOAPService wires the full pipeline.
func NewSOAPService(
	transcription *TranscriptionService,
	claude *ClaudeSOAPClient,
	ragStore *rag.Store,
	openaiClient *openai.Client,
) *SOAPService {
	return &SOAPService{
		transcription: transcription,
		claude:        claude,
		ragStore:      ragStore,
		embedFn: func(ctx context.Context, text string) ([]float32, error) {
			return embedText(ctx, openaiClient, text)
		},
	}
}

// Process runs the full pipeline: audio → transcription → RAG → SOAP.
// The audio is never persisted; it is read and discarded after this call.
func (s *SOAPService) Process(ctx context.Context, audio io.Reader, filename string) (*ProcessResult, error) {
	transcript, err := s.transcription.TranscribeAudio(ctx, audio, filename)
	if err != nil {
		return nil, fmt.Errorf("transcribe: %w", err)
	}

	ragContext, sourceRefs := s.retrieveRAG(ctx, transcript)

	soapOut, err := s.claude.GenerateSOAP(ctx, transcript, ragContext)
	if err != nil {
		return nil, fmt.Errorf("soap generation: %w", err)
	}

	// Overwrite source_refs with the actual RAG sources used (authoritative).
	soapOut.SourceRefs = sourceRefs

	return &ProcessResult{
		Transcription: transcript,
		SOAP:          soapOut,
		SourceRefs:    sourceRefs,
	}, nil
}

// retrieveRAG embeds the transcript and fetches the top-k similar clinical chunks.
// RAG failure is non-fatal: the pipeline continues without clinical context.
func (s *SOAPService) retrieveRAG(ctx context.Context, transcript string) ([]string, []models.SourceRef) {
	embedding, err := s.embedFn(ctx, transcript)
	if err != nil {
		slog.Warn("embed for RAG query failed; continuing without context", "err", err)
		return nil, nil
	}

	sources, err := s.ragStore.SimilarChunks(ctx, embedding, 5)
	if err != nil {
		slog.Warn("RAG similarity search failed; continuing without context", "err", err)
		return nil, nil
	}

	refs := make([]models.SourceRef, 0, len(sources))
	for _, src := range sources {
		refs = append(refs, models.SourceRef{Origin: src.Origin, Version: src.Version})
	}
	return rag.ChunkTexts(sources), refs
}

// embedText calls the OpenAI Embeddings API and returns a float32 vector.
func embedText(ctx context.Context, client *openai.Client, text string) ([]float32, error) {
	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(embeddingModel),
	})
	if err != nil {
		return nil, fmt.Errorf("openai embeddings: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	raw := resp.Data[0].Embedding
	vec := make([]float32, len(raw))
	for i, v := range raw {
		vec[i] = float32(v)
	}
	return vec, nil
}
