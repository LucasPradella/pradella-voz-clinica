import { useState, FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { login, register, APIError } from '../services/api'

type AuthMode = 'login' | 'register'

export default function Auth() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      if (mode === 'login') {
        await login(email, password)
      } else {
        await register(email, password)
      }
      navigate('/')
    } catch (err) {
      if (err instanceof APIError) {
        if (err.code === 'email_taken') {
          setError('Este e-mail já está cadastrado. Faça login.')
        } else if (err.code === 'invalid_credentials') {
          setError('E-mail ou senha incorretos.')
        } else {
          setError('Ocorreu um erro. Tente novamente.')
        }
      } else {
        setError('Falha na conexão. Verifique sua internet.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="min-h-screen bg-gray-100 flex items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-6">
        <header className="text-center">
          <h1 className="text-2xl font-bold text-navy-700">Pradella Voz Clínica</h1>
          <p className="text-sm text-gray-500 mt-1">
            Evolução clínica por voz para fisioterapeutas
          </p>
        </header>

        <div className="bg-white rounded-lg border border-gray-300 shadow-sm overflow-hidden">
          {/* Tab switcher */}
          <div className="flex border-b border-gray-200">
            <TabButton active={mode === 'login'} onClick={() => setMode('login')}>
              Entrar
            </TabButton>
            <TabButton active={mode === 'register'} onClick={() => setMode('register')}>
              Cadastrar
            </TabButton>
          </div>

          <form onSubmit={handleSubmit} className="p-6 space-y-4" noValidate>
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-grafite-700 mb-1">
                E-mail profissional
              </label>
              <input
                id="email"
                type="email"
                autoComplete="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                required
                disabled={loading}
                className="w-full px-3 py-2 text-sm rounded border border-gray-300
                           focus:outline-none focus:ring-2 focus:ring-navy-300 focus:border-navy-500
                           disabled:bg-gray-50"
                placeholder="seu@email.com"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-grafite-700 mb-1">
                Senha
              </label>
              <input
                id="password"
                type="password"
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                value={password}
                onChange={e => setPassword(e.target.value)}
                required
                minLength={8}
                disabled={loading}
                className="w-full px-3 py-2 text-sm rounded border border-gray-300
                           focus:outline-none focus:ring-2 focus:ring-navy-300 focus:border-navy-500
                           disabled:bg-gray-50"
                placeholder="Mínimo 8 caracteres"
              />
            </div>

            {error && (
              <p role="alert" className="text-sm text-red-600">
                {error}
              </p>
            )}

            <button
              type="submit"
              disabled={loading || !email || !password}
              className="w-full py-2.5 rounded bg-navy-700 text-white font-medium text-sm
                         hover:bg-navy-600 disabled:opacity-50 disabled:cursor-not-allowed
                         transition-colors focus:outline-none focus:ring-4 focus:ring-navy-300"
            >
              {loading
                ? 'Aguarde...'
                : mode === 'login'
                ? 'Entrar'
                : 'Criar conta grátis'}
            </button>

            {mode === 'register' && (
              <p className="text-xs text-gray-400 text-center">
                Plano Free: 10 evoluções/mês. Sem cartão de crédito.
              </p>
            )}
          </form>
        </div>
      </div>
    </main>
  )
}

interface TabButtonProps {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}

function TabButton({ active, onClick, children }: TabButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex-1 py-3 text-sm font-medium transition-colors
        ${active
          ? 'text-navy-700 border-b-2 border-navy-700 bg-white'
          : 'text-gray-500 hover:text-grafite-600 bg-gray-50'
        }`}
    >
      {children}
    </button>
  )
}
