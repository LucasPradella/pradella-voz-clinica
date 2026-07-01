import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Recorder } from '../components/Recorder'
import { SoapResult } from '../components/SoapResult'
import { SoapEditor } from '../components/SoapEditor'
import {
  EvolutionResponse,
  createEvolution,
  patchEvolution,
  getSubscription,
  startCheckout,
  getStoredUser,
  APIError,
} from '../services/api'

type HomeState = 'idle' | 'processing' | 'result' | 'editing' | 'error' | 'quota_exceeded'

export default function Home() {
  const [state, setState] = useState<HomeState>('idle')
  const [evolution, setEvolution] = useState<EvolutionResponse | null>(null)
  const [errorMsg, setErrorMsg] = useState('')
  const [isEditorSaving, setIsEditorSaving] = useState(false)

  const user = getStoredUser()
  const { data: subscription } = useQuery({
    queryKey: ['subscription'],
    queryFn: getSubscription,
    staleTime: 60_000,
  })

  async function handleAudioReady(blob: Blob) {
    setState('processing')
    setErrorMsg('')
    try {
      const result = await createEvolution(blob)
      setEvolution(result)
      setState('result')
    } catch (err) {
      if (err instanceof APIError && err.status === 402) {
        setState('quota_exceeded')
      } else {
        setErrorMsg(
          err instanceof APIError
            ? err.message
            : 'Ocorreu um erro ao processar o áudio. Tente novamente.'
        )
        setState('error')
      }
    }
  }

  async function handlePatch(patch: Parameters<typeof patchEvolution>[1]) {
    if (!evolution?.id) return
    setIsEditorSaving(true)
    try {
      const updated = await patchEvolution(evolution.id, patch)
      setEvolution(updated)
      setState('result')
    } finally {
      setIsEditorSaving(false)
    }
  }

  function reset() {
    setEvolution(null)
    setState('idle')
    setErrorMsg('')
  }

  const quota = subscription?.quota
  const quotaWarning =
    user?.plan === 'free' && quota && quota.limit !== null && quota.used >= quota.limit - 1

  return (
    <main className="min-h-screen bg-gray-100 px-4 py-8">
      <div className="max-w-xl mx-auto space-y-6">
        <header className="text-center">
          <div className="flex items-center justify-between mb-1">
            <h1 className="text-2xl font-bold text-navy-700">Pradella Voz Clínica</h1>
            {user?.plan === 'pro' && (
              <Link
                to="/history"
                className="text-sm text-navy-700 underline hover:text-navy-600"
              >
                Histórico
              </Link>
            )}
          </div>
          {user && (
            <p className="text-sm text-gray-500">
              {user.email} · {user.plan === 'pro' ? 'Pro' : 'Free'}
              {quota && quota.limit !== null && (
                <span className="ml-2 text-gray-400">
                  ({quota.used}/{quota.limit} evoluções este mês)
                </span>
              )}
            </p>
          )}
        </header>

        {quotaWarning && (
          <div className="rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-800">
            Você usou {quota?.used} de {quota?.limit} evoluções do plano Free este mês.{' '}
            <a href="/account" className="underline font-medium">
              Faça upgrade para Pro
            </a>{' '}
            para acesso ilimitado.
          </div>
        )}

        {/* Recording section */}
        {(state === 'idle' || state === 'error') && (
          <section className="flex flex-col items-center gap-6 py-8">
            <p className="text-center text-grafite-600 text-sm max-w-xs">
              Grave a evolução do atendimento. O áudio é descartado após a geração.
            </p>
            <Recorder onAudioReady={handleAudioReady} disabled={state === 'processing'} />
            {state === 'error' && (
              <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 text-center max-w-sm">
                {errorMsg}
                <button onClick={reset} className="block mt-2 underline text-xs">
                  Tentar novamente
                </button>
              </div>
            )}
          </section>
        )}

        {/* Processing */}
        {state === 'processing' && (
          <section className="flex flex-col items-center gap-4 py-8">
            <LoadingSpinner />
            <p className="text-grafite-600 text-sm">Processando áudio...</p>
            <p className="text-gray-400 text-xs">Transcrição → RAG → SOAP (pode levar até 30s)</p>
          </section>
        )}

        {/* Quota exceeded */}
        {state === 'quota_exceeded' && (
          <section className="rounded-lg border border-navy-300 bg-navy-50 px-6 py-8 text-center space-y-4">
            <h2 className="font-semibold text-navy-700">Cota mensal atingida</h2>
            <p className="text-sm text-grafite-600">
              Você usou todas as {quota?.limit} evoluções gratuitas deste mês.
              Faça upgrade para o plano Pro e tenha acesso ilimitado.
            </p>
            <UpgradeButton />
            <button onClick={reset} className="block mx-auto text-xs text-gray-400 underline">
              Voltar
            </button>
          </section>
        )}

        {/* Result */}
        {state === 'result' && evolution && (
          <section className="space-y-4">
            <SoapResult
              evolution={evolution}
              onEdit={evolution.id ? () => setState('editing') : undefined}
            />
            <button
              onClick={reset}
              className="w-full py-3 rounded border border-gray-300 text-sm text-grafite-600
                         hover:bg-gray-50 transition-colors"
            >
              Nova gravação
            </button>
          </section>
        )}

        {/* Editor */}
        {state === 'editing' && evolution && (
          <SoapEditor
            evolution={evolution}
            onSave={handlePatch}
            onCancel={() => setState('result')}
            isSaving={isEditorSaving}
          />
        )}
      </div>
    </main>
  )
}

function UpgradeButton() {
  async function handleUpgrade() {
    try {
      const { checkout_url } = await startCheckout()
      window.location.href = checkout_url
    } catch {
      alert('Erro ao iniciar o checkout. Tente novamente.')
    }
  }

  return (
    <button
      onClick={handleUpgrade}
      className="inline-block px-6 py-3 rounded bg-navy-700 text-white font-medium
                 hover:bg-navy-600 transition-colors"
    >
      Assinar Pro — R$ 49,90/mês
    </button>
  )
}

function LoadingSpinner() {
  return (
    <div
      className="w-12 h-12 rounded-full border-4 border-gray-200 border-t-navy-700 animate-spin"
      role="status"
      aria-label="Carregando"
    />
  )
}
