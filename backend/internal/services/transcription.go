package services

import (
	"context"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

const groqBaseURL = "https://api.groq.com/openai/v1"

// TranscriptionService transcribes audio using Groq Whisper (free tier).
type TranscriptionService struct {
	client *openai.Client
}

func NewTranscriptionService(apiKey string) (*TranscriptionService, error) {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = groqBaseURL
	return &TranscriptionService{client: openai.NewClientWithConfig(cfg)}, nil
}

// TranscribeAudio sends the audio to Groq Whisper and returns the Portuguese transcription.
func (s *TranscriptionService) TranscribeAudio(ctx context.Context, audio io.Reader, filename string) (string, error) {
	resp, err := s.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    "whisper-large-v3-turbo",
		Reader:   audio,
		FilePath: filename,
		Language: "pt",
		Format:   openai.AudioResponseFormatText,
	})
	if err != nil {
		return "", fmt.Errorf("groq whisper transcription: %w", err)
	}
	return resp.Text, nil
}
