import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { getSubscription, startCheckout, clearAuth } from '../services/api'

export default function Account() {
  const navigate = useNavigate()
  const { data: sub, isLoading } = useQuery({
    queryKey: ['subscription'],
    queryFn: getSubscription,
  })

  async function handleUpgrade() {
    try {
      const { checkout_url } = await startCheckout()
      window.location.href = checkout_url
    } catch {
      alert('Erro ao iniciar o checkout. Tente novamente.')
    }
  }

  function handleLogout() {
    clearAuth()
    navigate('/auth')
  }

  return (
    <main className="min-h-screen bg-gray-100 px-4 py-8">
      <div className="max-w-sm mx-auto space-y-6">
        <header className="flex items-center justify-between">
          <h1 className="text-xl font-bold text-navy-700">Minha conta</h1>
          <Link to="/" className="text-sm text-navy-700 hover:text-navy-600 underline">
            Início
          </Link>
        </header>

        {isLoading ? (
          <div className="flex justify-center py-8">
            <div className="w-8 h-8 rounded-full border-4 border-gray-200 border-t-navy-700 animate-spin" />
          </div>
        ) : sub ? (
          <>
            <div className="bg-white rounded-lg border border-gray-300 shadow-sm p-5 space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-grafite-600">Plano atual</span>
                <span className={`text-sm font-semibold ${sub.plan === 'pro' ? 'text-navy-700' : 'text-grafite-600'}`}>
                  {sub.plan === 'pro' ? 'Pro' : 'Free'}
                </span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm text-grafite-600">Status</span>
                <StatusBadge status={sub.status} />
              </div>
              {sub.plan === 'free' && sub.quota && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-grafite-600">Evoluções este mês</span>
                  <span className="text-sm font-medium text-grafite-700">
                    {sub.quota.used} / {sub.quota.limit ?? '∞'}
                  </span>
                </div>
              )}
              {sub.current_period_end && (
                <div className="flex items-center justify-between">
                  <span className="text-sm text-grafite-600">Próxima renovação</span>
                  <span className="text-sm text-grafite-600">
                    {new Date(sub.current_period_end).toLocaleDateString('pt-BR')}
                  </span>
                </div>
              )}
            </div>

            {sub.plan === 'free' && (
              <div className="bg-white rounded-lg border border-navy-300 p-5 space-y-3">
                <h2 className="font-semibold text-navy-700">Upgrade para Pro</h2>
                <ul className="space-y-1 text-sm text-grafite-600">
                  <li>✓ Evoluções ilimitadas</li>
                  <li>✓ Histórico na nuvem</li>
                  <li>✓ Edição e finalização</li>
                </ul>
                <button
                  onClick={handleUpgrade}
                  className="w-full py-2.5 rounded bg-navy-700 text-white font-medium text-sm
                             hover:bg-navy-600 transition-colors"
                >
                  Assinar Pro — R$ 49,90/mês
                </button>
              </div>
            )}
          </>
        ) : (
          <p className="text-sm text-red-600">Erro ao carregar dados da conta.</p>
        )}

        <button
          onClick={handleLogout}
          className="w-full py-2.5 rounded border border-gray-300 text-sm text-grafite-600
                     hover:bg-gray-50 transition-colors"
        >
          Sair
        </button>
      </div>
    </main>
  )
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    active: { label: 'Ativo', cls: 'bg-navy-100 text-navy-700' },
    canceled: { label: 'Cancelado', cls: 'bg-gray-100 text-gray-600' },
    past_due: { label: 'Pagamento pendente', cls: 'bg-amber-100 text-amber-700' },
  }
  const { label, cls } = map[status] ?? { label: status, cls: 'bg-gray-100 text-gray-600' }
  return <span className={`text-xs px-2 py-0.5 rounded font-medium ${cls}`}>{label}</span>
}
