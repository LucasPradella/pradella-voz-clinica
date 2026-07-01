import { EvolutionResponse, ConfidenceFlag } from '../services/api'
import { CopyButton } from './CopyButton'
import { CidSuggestions } from './CidSuggestions'

interface SoapResultProps {
  evolution: EvolutionResponse
  onEdit?: () => void
}

export function SoapResult({ evolution, onEdit }: SoapResultProps) {
  const { soap, status } = evolution
  const cid_suggestions = evolution.cid_suggestions ?? []
  const confidence_flags = evolution.confidence_flags ?? []
  const source_refs = evolution.source_refs ?? []

  const fullText = formatSoapText(soap)

  return (
    <article className="rounded-lg border border-gray-300 bg-white shadow-sm overflow-hidden">
      <header className="flex items-center justify-between px-4 py-3 border-b border-gray-200 bg-gray-50">
        <h2 className="font-semibold text-grafite-700">Evolução SOAP</h2>
        <div className="flex items-center gap-2">
          {status === 'draft' && (
            <span className="text-xs px-2 py-0.5 rounded bg-gray-200 text-gray-600">Rascunho</span>
          )}
          {status === 'finalized' && (
            <span className="text-xs px-2 py-0.5 rounded bg-navy-100 text-navy-700">Finalizado</span>
          )}
          {onEdit && (
            <button
              onClick={onEdit}
              className="text-sm text-navy-700 underline hover:text-navy-600"
            >
              Editar
            </button>
          )}
          <CopyButton text={fullText} />
        </div>
      </header>

      <div className="divide-y divide-gray-100">
        <SoapSection label="S — Subjetivo" text={soap.s} flags={confidence_flags} />
        <SoapSection label="O — Objetivo" text={soap.o} flags={confidence_flags} />
        <SoapSection label="A — Avaliação" text={soap.a} flags={confidence_flags} />
        <SoapSection label="P — Plano" text={soap.p} flags={confidence_flags} />
      </div>

      {cid_suggestions.length > 0 && (
        <div className="px-4 py-3 border-t border-gray-200">
          <CidSuggestions suggestions={cid_suggestions} />
        </div>
      )}

      {source_refs.length > 0 && (
        <footer className="px-4 py-2 border-t border-gray-200 bg-gray-50">
          <p className="text-xs text-gray-400">
            Referências:{' '}
            {source_refs.map((r, i) => (
              <span key={i}>
                {r.origin} v{r.version}{i < source_refs.length - 1 ? ', ' : ''}
              </span>
            ))}
          </p>
        </footer>
      )}
    </article>
  )
}

interface SoapSectionProps {
  label: string
  text: string
  flags: ConfidenceFlag[]
}

function SoapSection({ label, text, flags }: SoapSectionProps) {
  const highlighted = highlightFlags(text, flags)

  return (
    <section className="px-4 py-3">
      <h3 className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1">{label}</h3>
      {highlighted.length > 0 ? (
        <p className="text-sm text-grafite-600 leading-relaxed whitespace-pre-wrap">
          {highlighted.map((part, i) =>
            part.flagged ? (
              <mark
                key={i}
                className="bg-amber-100 text-amber-900 rounded px-0.5"
                title={`Baixa confiança: ${part.reason}`}
              >
                {part.text}
              </mark>
            ) : (
              <span key={i}>{part.text}</span>
            )
          )}
        </p>
      ) : (
        <p className="text-sm text-grafite-600 leading-relaxed whitespace-pre-wrap">
          {text || <span className="text-gray-400 italic">Não informado</span>}
        </p>
      )}
    </section>
  )
}

interface TextPart {
  text: string
  flagged: boolean
  reason?: string
}

function highlightFlags(text: string, flags: ConfidenceFlag[]): TextPart[] {
  if (!flags.length) return []

  const parts: TextPart[] = []
  let remaining = text

  for (const flag of flags) {
    const idx = remaining.indexOf(flag.span)
    if (idx === -1) continue
    if (idx > 0) parts.push({ text: remaining.slice(0, idx), flagged: false })
    parts.push({ text: flag.span, flagged: true, reason: flag.reason })
    remaining = remaining.slice(idx + flag.span.length)
  }

  if (remaining) parts.push({ text: remaining, flagged: false })
  return parts.length > 0 ? parts : []
}

function formatSoapText(soap: EvolutionResponse['soap']): string {
  return [
    `S — Subjetivo:\n${soap.s}`,
    `O — Objetivo:\n${soap.o}`,
    `A — Avaliação:\n${soap.a}`,
    `P — Plano:\n${soap.p}`,
  ].join('\n\n')
}
