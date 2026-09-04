import { useState, type FormEvent } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '../lib/api'
import { useAuth } from '../lib/auth'
import { AuthLayout } from '../components/AuthLayout'
import '../components/Form.css'

interface LoginResponse {
  token: string
}

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const navigate = useNavigate()
  const { login } = useAuth()

  const mutation = useMutation({
    mutationFn: () => api.post<LoginResponse>('/api/auth/login', { email, password }),
    onSuccess: (data) => {
      login(data.token)
      navigate('/', { replace: true })
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    mutation.mutate()
  }

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to continue to FileForge"
      footer={
        <span>
          Don&apos;t have an account? <Link to="/register">Create one</Link>
        </span>
      }
    >
      <form className="form" onSubmit={handleSubmit}>
        {mutation.isError && (
          <div className="form-error">
            {mutation.error instanceof ApiError ? mutation.error.message : 'Something went wrong'}
          </div>
        )}
        <div className="field">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="you@example.com"
            required
          />
        </div>
        <div className="field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            required
          />
        </div>
        <button className="btn-primary" type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </AuthLayout>
  )
}
