import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'
import { AuthLayout } from '../components/AuthLayout'
import '../components/Form.css'

export function ForgotPasswordPage() {
  const [email, setEmail] = useState('')

  const mutation = useMutation({
    mutationFn: () => api.post('/api/auth/forgot-password', { email }),
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    mutation.mutate()
  }

  // The backend deliberately gives the same response whether or not the
  // email has an account - showing anything different here would turn this
  // form into a way to check who's registered.
  if (mutation.isSuccess) {
    return (
      <AuthLayout
        title="Check your email"
        subtitle="If an account exists for that address, a reset link is on its way."
        footer={
          <span>
            <Link to="/login">Back to sign in</Link>
          </span>
        }
      >
        <p style={{ fontSize: 14, color: 'var(--text-muted)', lineHeight: 1.6 }}>
          The link expires in 1 hour. Didn't get it? Check spam, or try again with the same address.
        </p>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout
      title="Forgot your password?"
      subtitle="Enter your email and we'll send you a reset link"
      footer={
        <span>
          Remembered it? <Link to="/login">Sign in</Link>
        </span>
      }
    >
      <form className="form" onSubmit={handleSubmit}>
        <div className="field">
          <label htmlFor="email">Email</label>
          <input
            id="email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email address"
            required
          />
        </div>
        <button className="btn-primary" type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? 'Sending…' : 'Send reset link'}
        </button>
      </form>
    </AuthLayout>
  )
}
