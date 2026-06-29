package services

import (
	"strings"
	"unicode"

	"github.com/pradella/voz-clinica/internal/models"
)

// GuardrailChecker verifies post-generation that SOAP content is anchored in the transcription.
// It supplements (not replaces) the system-prompt-level guardrail already enforced by Claude.
type GuardrailChecker struct {
	// overlapThreshold is the minimum fraction of a SOAP sentence's words that must
	// appear in the transcript. Sentences below this are flagged as potentially unanchored.
	overlapThreshold float64
}

func NewGuardrailChecker() *GuardrailChecker {
	return &GuardrailChecker{overlapThreshold: 0.20}
}

// Check scans the Assessment (A) and Plan (P) fields for content not sufficiently anchored
// in the transcription. New flags are returned to be merged into the SOAP output;
// existing Claude-generated flags are preserved by the caller.
func (g *GuardrailChecker) Check(transcription string, soap *SOAPOutput) []models.ConfidenceFlag {
	if transcription == "" {
		return nil
	}
	transcriptWords := tokenizeWords(transcription)

	var flags []models.ConfidenceFlag
	for _, field := range []string{soap.A, soap.P} {
		for _, sentence := range splitSentences(field) {
			sentence = strings.TrimSpace(sentence)
			if sentence == "" {
				continue
			}
			if g.overlapScore(transcriptWords, sentence) < g.overlapThreshold {
				flags = append(flags, models.ConfidenceFlag{
					Span:   sentence,
					Reason: "not_mentioned",
				})
			}
		}
	}
	return flags
}

// overlapScore returns the fraction of meaningful words in sentence that appear in transcriptWords.
func (g *GuardrailChecker) overlapScore(transcriptWords map[string]bool, sentence string) float64 {
	words := tokenizeWords(sentence)
	if len(words) == 0 {
		return 1.0
	}
	matches := 0
	for w := range words {
		if transcriptWords[w] {
			matches++
		}
	}
	return float64(matches) / float64(len(words))
}

// tokenizeWords splits text into a set of normalized, non-stopword tokens.
func tokenizeWords(text string) map[string]bool {
	set := map[string]bool{}
	for _, word := range strings.Fields(text) {
		w := normalize(word)
		if w != "" && !ptStopword[w] && len(w) > 2 {
			set[w] = true
		}
	}
	return set
}

// normalize lowercases and strips punctuation from a word.
func normalize(word string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, word)
}

// splitSentences splits text on '.', '!', '?', ';' and ',' so that long
// comma-delimited lists count as separate spans for overlap scoring.
func splitSentences(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == ';' || r == ','
	})
}

// ptStopword is a set of common Portuguese words that carry little diagnostic meaning.
var ptStopword = map[string]bool{
	"a": true, "ao": true, "aos": true, "à": true, "às": true,
	"o": true, "os": true, "as": true, "e": true, "em": true,
	"de": true, "do": true, "da": true, "dos": true, "das": true,
	"para": true, "por": true, "com": true, "um": true, "uma": true,
	"que": true, "se": true, "no": true, "na": true, "nos": true,
	"nas": true, "seu": true, "sua": true, "seus": true, "suas": true,
	"mais": true, "mas": true, "ou": true, "este": true, "esta": true,
	"esse": true, "essa": true, "isso": true, "isto": true, "ele": true,
	"ela": true, "eles": true, "elas": true, "foi": true, "ser": true,
	"ter": true, "como": true, "não": true, "nao": true, "pelo": true,
	"pela": true, "pelos": true, "pelas": true, "também": true, "tambem": true,
}
