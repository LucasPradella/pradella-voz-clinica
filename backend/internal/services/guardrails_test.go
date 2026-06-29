package services_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pradella/voz-clinica/internal/models"
	"github.com/pradella/voz-clinica/internal/services"
)

func TestGuardrail_NothingFabricated(t *testing.T) {
	checker := services.NewGuardrailChecker()

	transcript := "Paciente relata dor lombar há três dias, piora ao sentar por longos períodos. Realizei alongamento e exercícios de estabilização lombar."
	soap := &services.SOAPOutput{
		S: "Dor lombar há três dias, piora ao sentar.",
		O: "Amplitude de movimento reduzida. Teste de Lasègue negativo.",
		A: "Lombalgia mecânica com disfunção de estabilização lombar.",
		P: "Exercícios de estabilização lombar. Orientação postural. Retorno em uma semana.",
		CIDSuggestions: []models.CIDSuggestion{
			{Code: "M54.5", Description: "Dor lombar baixa"},
		},
		ConfidenceFlags: nil,
	}

	flags := checker.Check(transcript, soap)
	// Should produce few or zero flags — content well-anchored in transcript.
	assert.LessOrEqual(t, len(flags), 3, "well-anchored SOAP should not produce many guardrail flags")
}

func TestGuardrail_FabricationDetected(t *testing.T) {
	checker := services.NewGuardrailChecker()

	// Transcript has no mention of surgery or fracture.
	transcript := "Paciente relata leve dor no joelho esquerdo."
	soap := &services.SOAPOutput{
		S: "Dor leve no joelho esquerdo.",
		O: "Edema grau III. Derrame articular volumoso.",
		A: "Ruptura completa do ligamento cruzado anterior com fratura condral associada e lesão meniscal.",
		P: "Indicação imediata de artroscopia de joelho com reconstrução do LCA e meniscectomia parcial.",
	}

	flags := checker.Check(transcript, soap)
	// Should detect unanchored content in A and P fields.
	assert.NotEmpty(t, flags, "fabricated content should produce guardrail flags")
	for _, f := range flags {
		assert.Equal(t, "not_mentioned", f.Reason)
	}
}

func TestGuardrail_CIDAlwaysSuggestion(t *testing.T) {
	checker := services.NewGuardrailChecker()

	// CID codes in cid_suggestions are always suggestions — this should never trigger a flag.
	transcript := "Paciente com dor lombar crônica, relatando melhora após fisioterapia."
	soap := &services.SOAPOutput{
		S: "Dor lombar crônica com melhora após fisioterapia.",
		O: "Força muscular preservada. Reflexos normais.",
		A: "Lombalgia crônica em fase de recuperação.",
		P: "Manutenção do protocolo de exercícios.",
		CIDSuggestions: []models.CIDSuggestion{
			{Code: "M54.5", Description: "Dor lombar baixa"},
			{Code: "M47.8", Description: "Outras espondiloartrose"},
		},
	}

	flags := checker.Check(transcript, soap)
	// CID in cid_suggestions does not trigger flags — they are always suggestions.
	for _, f := range flags {
		// None of the flags should reference CID codes directly.
		assert.NotContains(t, f.Span, "M54.5")
		assert.NotContains(t, f.Span, "M47.8")
	}
}

func TestGuardrail_LowConfidencePreserved(t *testing.T) {
	checker := services.NewGuardrailChecker()

	transcript := "O paciente... hmm... relata assim..."
	soap := &services.SOAPOutput{
		S: "Relato impreciso do paciente.",
		O: "Avaliação limitada.",
		A: "Dados insuficientes para avaliação completa.",
		P: "Reavaliação necessária.",
		ConfidenceFlags: []models.ConfidenceFlag{
			{Span: "Relato impreciso", Reason: "audio_unclear"},
		},
	}

	extraFlags := checker.Check(transcript, soap)

	// Guardrail may add more flags but should not remove existing ones.
	// Caller is responsible for merging (tested in handler), not checker.
	assert.NotNil(t, soap.ConfidenceFlags, "existing flags must be preserved by caller")
	_ = extraFlags
}

func TestGuardrail_EmptyTranscript(t *testing.T) {
	checker := services.NewGuardrailChecker()

	soap := &services.SOAPOutput{
		A: "Avaliação impossível.",
		P: "Sem plano.",
	}

	// Empty transcript: guardrail skips (not enough data to compare against).
	flags := checker.Check("", soap)
	assert.Nil(t, flags)
}

func TestGuardrail_ShortTranscriptLongSOAP(t *testing.T) {
	checker := services.NewGuardrailChecker()

	transcript := "dor."
	soap := &services.SOAPOutput{
		A: "Síndrome dolorosa complexa multifatorial com comprometimento neurológico periférico extenso e disfunção autonômica.",
		P: "Bloqueio neural diagnóstico, eletroneuromiografia, ressonância magnética contrastada, interconsulta com neurologia e reumatologia.",
	}

	flags := checker.Check(transcript, soap)
	// Very short transcript vs very long SOAP → should generate many flags.
	assert.Greater(t, len(flags), 2, "short transcript with elaborate SOAP should generate multiple flags")
}
