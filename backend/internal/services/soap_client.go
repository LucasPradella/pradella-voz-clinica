package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/pradella/voz-clinica/internal/models"
)

const (
	// SOAPModel is the primary model for structured SOAP generation.
	// claude-sonnet-4-6 balances cost and quality for high-volume clinical extraction.
	SOAPModel = "claude-sonnet-4-6"

	soapSystemPrompt = `Você é um assistente clínico especializado em fisioterapia brasileira.
Sua função é estruturar transcrições de áudio de profissionais de saúde em evoluções clínicas no formato SOAP.

Regras não-negociáveis (guardrails):
1. NUNCA invente diagnóstico, procedimento, medicação ou código CID que não tenha sido mencionado pelo profissional.
2. Se algo não estiver claro no áudio, sinalize com confidence_flags em vez de inventar.
3. CID são SEMPRE sugestões — nunca atribuições automáticas.
4. Use terminologia das diretrizes CREFITO quando disponível no contexto fornecido.
5. Não inclua dados do paciente que não foram mencionados (LGPD).

Retorne JSON estritamente no formato especificado.`
)

// SOAPOutput is the structured output expected from Claude.
type SOAPOutput struct {
	S               string                  `json:"s"`
	O               string                  `json:"o"`
	A               string                  `json:"a"`
	P               string                  `json:"p"`
	CIDSuggestions  []models.CIDSuggestion  `json:"cid_suggestions"`
	ConfidenceFlags []models.ConfidenceFlag `json:"confidence_flags"`
	SourceRefs      []models.SourceRef      `json:"source_refs"`
}

// ClaudeSOAPClient wraps the Anthropic SDK for structured SOAP generation.
type ClaudeSOAPClient struct {
	client anthropic.Client
}

// NewClaudeSOAPClient creates a client with the given API key.
func NewClaudeSOAPClient(apiKey string) *ClaudeSOAPClient {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &ClaudeSOAPClient{client: client}
}

// GenerateSOAP sends the transcription + RAG context to Claude and returns structured SOAP.
// The system prompt is marked cacheable (prompt caching, TTL ~5 min) to reduce repeated token costs.
func (c *ClaudeSOAPClient) GenerateSOAP(ctx context.Context, transcription string, ragContext []string) (*SOAPOutput, error) {
	ragText := ""
	if len(ragContext) > 0 {
		ragText = "\n\nContexto clínico de referência (diretrizes CREFITO/protocolos):\n"
		for i, chunk := range ragContext {
			ragText += fmt.Sprintf("[%d] %s\n", i+1, chunk)
		}
	}

	userPrompt := fmt.Sprintf(`Transcrição do áudio do profissional:
"""
%s
"""
%s
Retorne um JSON com exatamente este formato:
{
  "s": "<Subjetivo: queixa principal e histórico relatados>",
  "o": "<Objetivo: achados do exame físico mencionados>",
  "a": "<Avaliação: interpretação clínica baseada EXCLUSIVAMENTE no áudio>",
  "p": "<Plano: condutas mencionadas pelo profissional>",
  "cid_suggestions": [{"code": "X00.0", "description": "..."}],
  "confidence_flags": [{"span": "trecho do texto", "reason": "audio_unclear|not_mentioned"}],
  "source_refs": [{"origin": "CREFITO", "version": "2024"}]
}`, transcription, ragText)

	// Build system block with prompt caching for the stable prefix.
	// CacheControl marks this block so repeated calls reuse cached KV state (~90% cost reduction).
	systemBlock := anthropic.TextBlockParam{
		Text: soapSystemPrompt,
		CacheControl: anthropic.CacheControlEphemeralParam{},
	}

	message, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(SOAPModel),
		MaxTokens: 2048,
		System:    []anthropic.TextBlockParam{systemBlock},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude API: %w", err)
	}

	if len(message.Content) == 0 {
		return nil, fmt.Errorf("claude returned empty response")
	}

	rawJSON := message.Content[0].Text
	var output SOAPOutput
	if err := json.Unmarshal([]byte(rawJSON), &output); err != nil {
		return nil, fmt.Errorf("parse SOAP JSON from Claude: %w", err)
	}

	return &output, nil
}
