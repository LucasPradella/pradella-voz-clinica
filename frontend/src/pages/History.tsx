import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { listEvolutions, getEvolution, EvolutionResponse } from '../services/api'
import { SoapResult } from '../components/SoapResult'

export default function History() {
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<EvolutionResponse | null>(null)
  const [loadingId, setLoadingId] = useState<string | null>(null)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['evolutions', page],
    queryFn: () => listEvolutions(page, 20),
  })

  async function openEvolution(id: string) {
    setLoadingId(id)
    try {
      const evo = await getEvolution(id)
      setSelected(evo)
    } finally {
      setLoadingId(null)
    }
  }

  if (selected) {
    return (
      <main className="min-h-screen bg-gray-100 px-4 py-8">
        <div className="max-w-xl mx-auto space-y-4">
          <button
            onClick={() => setSelected(null)}
            className="flex items-center gap-2 text-sm text-navy-700 hover:text-navy-600"
          >
            <BackIcon /> Voltar ao histórico
          </button>
          <SoapResult evolution={selected} />
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-gray-100 px-4 py-8">
      <div className="max-w-xl mx-auto space-y-4">
        <header className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-navy-700">Histórico</h1>
          <Link to="/" className="text-sm text-navy-700 hover:text-navy-600 underline">
            Nova gravação
          </Link>
        </header>

        {isLoading && (
          <div className="flex justify-center py-12">
            <div className="w-8 h-8 rounded-full border-4 border-gray-200 border-t-navy-700 animate-spin" />
          </div>
        )}

        {isError && (
          <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            Erro ao carregar histórico. Tente novamente.
          </div>
        )}

        {data && data.items.length === 0 && (
          <div className="text-center py-12 text-gray-400 text-sm">
            Nenhuma evolução registrada.
          </div>
        )}

        {data && data.items.length > 0 && (
          <>
            <ul className="space-y-2" role="list">
              {data.items.map(item => (
                <li key={item.id}>
                  <button
                    onClick={() => openEvolution(item.id)}
                    disabled={loadingId === item.id}
                    className="w-full text-left rounded-lg border border-gray-200 bg-white px-4 py-3
                               hover:border-navy-300 hover:bg-navy-50 transition-colors
                               disabled:opacity-70 focus:outline-none focus:ring-2 focus:ring-navy-300"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-grafite-700 truncate">
                          {item.label || 'Evolução sem rótulo'}
                        </p>
                        <p className="text-xs text-gray-400 mt-0.5">
                          {formatDate(item.created_at)}
                        </p>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        {item.status === 'finalized' ? (
                          <span className="text-xs px-2 py-0.5 rounded bg-navy-100 text-navy-700">
                            Finalizado
                          </span>
                        ) : (
                          <span className="text-xs px-2 py-0.5 rounded bg-gray-100 text-gray-500">
                            Rascunho
                          </span>
                        )}
                        {loadingId === item.id ? (
                          <span className="w-4 h-4 rounded-full border-2 border-gray-300 border-t-navy-700 animate-spin" />
                        ) : (
                          <ChevronIcon />
                        )}
                      </div>
                    </div>
                  </button>
                </li>
              ))}
            </ul>

            {data.total > 20 && (
              <div className="flex items-center justify-center gap-4 pt-2">
                <button
                  onClick={() => setPage(p => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="px-3 py-1 text-sm rounded border border-gray-300 disabled:opacity-50
                             hover:bg-gray-50 transition-colors"
                >
                  Anterior
                </button>
                <span className="text-sm text-gray-500">
                  {page} / {Math.ceil(data.total / 20)}
                </span>
                <button
                  onClick={() => setPage(p => p + 1)}
                  disabled={page >= Math.ceil(data.total / 20)}
                  className="px-3 py-1 text-sm rounded border border-gray-300 disabled:opacity-50
                             hover:bg-gray-50 transition-colors"
                >
                  Próxima
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  )
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function BackIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M10 3L5 8l5 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}

function ChevronIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
      <path d="M6 4l4 4-4 4" stroke="#9e9e9e" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  )
}
