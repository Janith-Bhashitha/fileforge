import { useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '../lib/api'
import { AuthLayout } from '../components/AuthLayout'
import { PasswordInput } from '../components/PasswordInput'
import '../components/Form.css'

export function ResetPasswordPage() {
  const [searchParams] = useSearchParams()
  const token = searchParams.get('token') ?? ''
  const navigate = useNavigate()

  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const mutation = useMutation({
    mutationFn: () => api.post('/api/auth/reset-password', { token, new_password: newPassword }),
    onSuccess: () => {
      // Land back on login rather than auto-signing in - the reset token
      // proved control of the email, not identity strong enough to skip
      // the normal login step.
      navigate('/login', { replace: true })
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    mutation.mutate()
  }

  if (!token) {
    return (
      <AuthLayout
        title="Invalid reset link"
        subtitle="This link is missing its token"
        footer={
          <span>
            <Link to="/forgot-password">Request a new link</Link>
          </span>
        }
      >
        <p style={{ fontSize: 14, color: 'var(--text-muted)' }}>
          Make sure you opened the full link from the email, not a shortened or partial copy of it.
        </p>
      </AuthLayout>
    )
  }

  const mismatch = confirmPassword.length > 0 && newPassword !== confirmPassword
  const tooShort = newPassword.length > 0 && newPassword.length < 8
  const canSubmit = newPassword.length >= 8 && newPassword === confirmPassword && !mutation.isPending

  return (
    <AuthLayout
      title="Choose a new password"
      subtitle="Make it at least 8 characters"
      footer={
        <span>
          <Link to="/login">Back to sign in</Link>
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
          <label htmlFor="new-password">New Password</label>
          <PasswordInput
            id="new-password"
            value={newPassword}
            onChange={setNewPassword}
            placeholder="Password"
            autoComplete="new-password"
            required
          />
          {tooShort && <span style={{ fontSize: 12, color: 'var(--error)' }}>At least 8 characters</span>}
        </div>
        <div className="field">
          <label htmlFor="confirm-password">Confirm Password</label>
          <PasswordInput
            id="confirm-password"
            value={confirmPassword}
            onChange={setConfirmPassword}
            placeholder="Confirm password"
            autoComplete="new-password"
            required
          />
          {mismatch && <span style={{ fontSize: 12, color: 'var(--error)' }}>Passwords don't match</span>}
        </div>
        <button className="btn-primary" type="submit" disabled={!canSubmit}>
          {mutation.isPending ? 'Resetting…' : 'Reset password'}
        </button>
      </form>
    </AuthLayout>
  )
}
