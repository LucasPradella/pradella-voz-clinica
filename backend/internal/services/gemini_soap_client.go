package services

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const (
	geminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	geminiModel   = "gemini-2.0-flash"
)

// GeminiSOAPClient generates SOAP via Google Gemini using its OpenAI-compatible endpoint.
type GeminiSOAPClient struct {
	client *openai.Client
}

// NewGeminiSOAPClient creates a client backed by the Gemini API.
func NewGeminiSOAPClient(apiKey string) *GeminiSOAPClient {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = geminiBaseURL
	return &GeminiSOAPClient{client: openai.NewClientWithConfig(cfg)}
}

// GenerateSOAP sends the transcription + RAG context to Gemini and returns structured SOAP.
func (g *GeminiSOAPClient) GenerateSOAP(ctx context.Context, transcription string, ragContext []string) (*SOAPOutput, error) {
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

	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     geminiModel,
		MaxTokens: 2048,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: soapSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gemini API: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("gemini returned empty response")
	}

	var output SOAPOutput
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &output); err != nil {
		return nil, fmt.Errorf("parse SOAP JSON from Gemini: %w", err)
	}

	return &output, nil
}
