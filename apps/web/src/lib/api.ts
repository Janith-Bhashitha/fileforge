const API_BASE_URL = import.meta.env.VITE_API_BASE_URL

function getToken(): string | null {
  return localStorage.getItem('token')
}

export function setToken(token: string) {
  localStorage.setItem('token', token)
}

export function clearToken() {
  localStorage.removeItem('token')
}

interface ApiErrorBody {
  error?: string
}

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) {
    super(message)
    this.status = status
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  }

  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers })

  if (!res.ok) {
    const body: ApiErrorBody = await res.json().catch(() => ({}))
    throw new ApiError(res.status, body.error ?? 'Request failed')
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json() as Promise<T>
}

async function upload<T>(path: string, formData: FormData): Promise<T> {
  const token = getToken()
  const res = await fetch(`${API_BASE_URL}${path}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: formData,
  })

  if (!res.ok) {
    const body: ApiErrorBody = await res.json().catch(() => ({}))
    throw new ApiError(res.status, body.error ?? 'Upload failed')
  }

  return res.json() as Promise<T>
}

async function downloadBlob(path: string): Promise<Blob> {
  const token = getToken()
  const res = await fetch(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  })

  if (!res.ok) {
    throw new ApiError(res.status, 'Download failed')
  }

  return res.blob()
}

// assetUrl resolves a server-relative path the API handed us (an avatar, say)
// against the same base the API itself is on.
//
// In production the base is empty and the page is same-origin with the API,
// so this returns the path untouched and nginx proxies it. In development
// the API is on a different port, and an <img src="/avatars/..."> would
// otherwise resolve against the Vite dev server and 404.
export function assetUrl(path: string): string {
  if (/^https?:\/\//.test(path)) return path
  return `${API_BASE_URL}${path}`
}

export const api = {
  get: <T>(path: string) => request<T>(path, { method: 'GET' }),
  post: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'POST', body: data ? JSON.stringify(data) : undefined }),
  patch: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: 'PATCH', body: data ? JSON.stringify(data) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  upload,
  downloadBlob,
}
