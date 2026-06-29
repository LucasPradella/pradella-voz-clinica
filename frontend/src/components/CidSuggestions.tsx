import { CIDSuggestion } from '../services/api'

interface CidSuggestionsProps {
  suggestions: CIDSuggestion[]
  onRemove?: (index: number) => void
}

export function CidSuggestions({ suggestions, onRemove }: CidSuggestionsProps) {
  if (suggestions.length === 0) return null

  return (
    <section aria-label="Sugestões de CID">
      <h3 className="text-sm font-semibold text-grafite-600 mb-2">
        Sugestões de CID
        <span className="ml-2 text-xs font-normal text-gray-500">(validar antes de usar)</span>
      </h3>
      <ul className="space-y-1" role="list">
        {suggestions.map((s, i) => (
          <li
            key={i}
            className="flex items-center justify-between gap-2 px-3 py-2 rounded bg-gray-100 border border-gray-300"
          >
            <div className="flex items-center gap-2 min-w-0">
              <span className="font-mono text-sm font-semibold text-navy-700 shrink-0">{s.code}</span>
              <span className="text-sm text-grafite-600 truncate">{s.description}</span>
            </div>
            {onRemove && (
              <button
                onClick={() => onRemove(i)}
                className="shrink-0 text-gray-400 hover:text-grafite-600 transition-colors"
                aria-label={`Remover sugestão ${s.code}`}
              >
                <RemoveIcon />
              </button>
            )}
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs text-gray-500">
        Sugestões geradas por IA — confirme com o prontuário antes de registrar.
      </p>
    </section>
  )
}

function RemoveIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
      <path d="M3 3l8 8M11 3l-8 8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  )
}
