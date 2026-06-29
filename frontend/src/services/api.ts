const BASE_URL = '/api'

export class APIError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly upgrade?: boolean
  ) {
    super(message)
    this.name = 'APIError'
  }
}

// ---- Types ----

export interface UserInfo {
  id: string
  email: string
  plan: 'free' | 'pro'
}

export interface AuthResponse {
  token: string
  user: UserInfo
}

export interface SOAP {
  s: string
  o: string
  a: string
  p: string
}

export interface CIDSuggestion {
  code: string
  description: string
}

export interface ConfidenceFlag {
  span: string
  reason: string
}

export interface SourceRef {
  origin: string
  version: string
}

export interface EvolutionResponse {
  id: string | null
  soap: SOAP
  cid_suggestions: CIDSuggestion[]
  confidence_flags: ConfidenceFlag[]
  source_refs: SourceRef[]
  status: 'draft' | 'finalized'
}

export interface EvolutionListItem {
  id: string
  label: string | null
  created_at: string
  status: 'draft' | 'finalized'
}

export interface EvolutionListResponse {
  items: EvolutionListItem[]
  page: number
  total: number
}

export interface SubscriptionResponse {
  plan: 'free' | 'pro'
  status: 'active' | 'canceled' | 'past_due'
  current_period_end?: string
  quota: {
    used: number
    limit: number | null
  }
}

// ---- Auth context ----

const TOKEN_KEY = 'pradella_token'
const USER_KEY = 'pradella_user'

export function getStoredToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function getStoredUser(): UserInfo | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as UserInfo
  } catch {
    return null
  }
}

export function storeAuth(token: string, user: UserInfo): void {
  localStorage.setItem(TOKEN_KEY, token)
  localStorage.setItem(USER_KEY, JSON.stringify(user))
}

export function clearAuth(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
}

// ---- Core fetch helper ----

async function request<T>(
  path: string,
  init: RequestInit = {},
  requiresAuth = true
): Promise<T> {
  const headers: Record<string, string> = {
    ...(init.headers as Record<string, string>),
  }

  if (requiresAuth) {
    const token = getStoredToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
  }

  if (!(init.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers })

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: { code: 'unknown', message: res.statusText } }))
    throw new APIError(
      res.status,
      body?.error?.code ?? 'unknown',
      body?.error?.message ?? 'Request failed',
      body?.upgrade
    )
  }

  return res.json() as Promise<T>
}

// ---- Auth API ----

export async function register(email: string, password: string): Promise<AuthResponse> {
  const data = await request<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }, false)
  storeAuth(data.token, data.user)
  return data
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  const data = await request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  }, false)
  storeAuth(data.token, data.user)
  return data
}

// ---- Evolutions API ----

export async function createEvolution(audioBlob: Blob, label?: string): Promise<EvolutionResponse> {
  const form = new FormData()
  form.append('audio', audioBlob, 'recording.webm')
  if (label) form.append('label', label)

  return request<EvolutionResponse>('/evolutions', {
    method: 'POST',
    body: form,
  })
}

export async function patchEvolution(
  id: string,
  patch: { label?: string; soap?: Partial<SOAP>; status?: 'finalized' }
): Promise<EvolutionResponse> {
  return request<EvolutionResponse>(`/evolutions/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(patch),
  })
}

export async function listEvolutions(page = 1, limit = 20): Promise<EvolutionListResponse> {
  return request<EvolutionListResponse>(`/evolutions?page=${page}&limit=${limit}`)
}

export async function getEvolution(id: string): Promise<EvolutionResponse> {
  return request<EvolutionResponse>(`/evolutions/${id}`)
}

// ---- Subscription API ----

export async function getSubscription(): Promise<SubscriptionResponse> {
  return request<SubscriptionResponse>('/subscription')
}

export async function startCheckout(): Promise<{ checkout_url: string }> {
  return request<{ checkout_url: string }>('/subscription/checkout', { method: 'POST' })
}
