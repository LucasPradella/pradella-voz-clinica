package services

import (
	"context"
	"fmt"
	"io"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// WhisperModel is OpenAI's Whisper model for speech-to-text.
	WhisperModel = "whisper-1"

	// AudioLanguage instructs Whisper to transcribe in Brazilian Portuguese.
	AudioLanguage = "pt"

	// MaxAudioDurationSeconds is the hard limit enforced before sending to Whisper.
	MaxAudioDurationSeconds = 120
)

// TranscriptionService wraps the OpenAI Whisper API for audio-to-text conversion.
type TranscriptionService struct {
	client *openai.Client
}

func NewTranscriptionService(apiKey string) *TranscriptionService {
	return &TranscriptionService{
		client: openai.NewClient(apiKey),
	}
}

// TranscribeAudio sends the audio stream to Whisper and returns the Portuguese transcription.
// The audio is processed in memory and never persisted (FR-017b).
func (s *TranscriptionService) TranscribeAudio(ctx context.Context, audio io.Reader, filename string) (string, error) {
	resp, err := s.client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    WhisperModel,
		Reader:   audio,
		FilePath: filename,
		Language: AudioLanguage,
		Format:   openai.AudioResponseFormatText,
	})
	if err != nil {
		return "", fmt.Errorf("whisper transcription: %w", err)
	}

	return resp.Text, nil
}
