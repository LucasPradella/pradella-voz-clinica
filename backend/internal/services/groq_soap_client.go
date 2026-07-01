package services

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

const groqSOAPModel = "llama-3.3-70b-versatile"

// GroqSOAPClient generates SOAP via Groq LLM (free tier) using the OpenAI-compatible endpoint.
type GroqSOAPClient struct {
	client *openai.Client
}

// NewGroqSOAPClient creates a client backed by the Groq API.
func NewGroqSOAPClient(apiKey string) *GroqSOAPClient {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = groqBaseURL
	return &GroqSOAPClient{client: openai.NewClientWithConfig(cfg)}
}

// GenerateSOAP sends the transcription + RAG context to Groq Llama and returns structured SOAP.
func (g *GroqSOAPClient) GenerateSOAP(ctx context.Context, transcription string, ragContext []string) (*SOAPOutput, error) {
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
Retorne um JSON com exatamente este formato (sem markdown, apenas JSON puro):
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
		Model:     groqSOAPModel,
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
		return nil, fmt.Errorf("groq LLM: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("groq returned empty response")
	}

	var output SOAPOutput
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &output); err != nil {
		return nil, fmt.Errorf("parse SOAP JSON from Groq: %w", err)
	}

	return &output, nil
}
