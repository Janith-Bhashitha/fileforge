import { useRef, useState } from 'react'
import { useOutletContext } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import type { MeResponse, ShellContext } from '../components/AppShell'
import { useDeveloperMode } from '../lib/developerMode'
import { Switch } from '../components/Switch'
import { api, ApiError, assetUrl } from '../lib/api'
import { useToast } from '../components/Toast'

export function SettingsPage() {
  const { me } = useOutletContext<ShellContext>()
  const { developerMode, toggleDeveloperMode } = useDeveloperMode()
  const toast = useToast()
  const queryClient = useQueryClient()
  const avatarInputRef = useRef<HTMLInputElement>(null)

  const [name, setName] = useState(me?.display_name ?? '')

  const updateProfile = useMutation({
    mutationFn: (displayName: string) => api.patch<MeResponse>('/api/v1/profile', { display_name: displayName }),
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated)
      toast.success('Profile updated')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to update profile'),
  })

  const uploadAvatar = useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData()
      formData.append('avatar', file)
      return api.upload<MeResponse>('/api/v1/profile/avatar', formData)
    },
    onSuccess: (updated) => {
      queryClient.setQueryData(['me'], updated)
      toast.success('Avatar updated')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to upload avatar'),
  })

  function handleAvatarSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) uploadAvatar.mutate(file)
    e.target.value = ''
  }

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const changePassword = useMutation({
    mutationFn: () => api.post('/api/v1/profile/change-password', { current_password: currentPassword, new_password: newPassword }),
    onSuccess: () => {
      toast.success('Password changed')
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to change password'),
  })

  const passwordsMismatch = newPassword.length > 0 && confirmPassword.length > 0 && newPassword !== confirmPassword
  const passwordTooShort = newPassword.length > 0 && newPassword.length < 8
  const canSubmitPassword =
    currentPassword.length > 0 && newPassword.length >= 8 && newPassword === confirmPassword && !changePassword.isPending

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Settings</h1>
          <p>Manage your profile, password and preferences.</p>
        </div>
      </div>

      <div className="card">
        <div className="card-title">Profile</div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 24 }}>
          {me?.avatar_url ? (
            <img
              src={assetUrl(me.avatar_url)}
              alt=""
              style={{ width: 56, height: 56, borderRadius: '50%', objectFit: 'cover' }}
            />
          ) : (
            <div
              style={{
                width: 56,
                height: 56,
                borderRadius: '50%',
                background: 'var(--accent)',
                color: 'var(--accent-contrast)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontWeight: 700,
                fontSize: 20,
              }}
            >
              {(me?.display_name || me?.email || '?')[0].toUpperCase()}
            </div>
          )}
          <input
            ref={avatarInputRef}
            type="file"
            accept="image/jpeg,image/png"
            style={{ display: 'none' }}
            onChange={handleAvatarSelected}
          />
          <button
            className="btn-secondary"
            type="button"
            disabled={uploadAvatar.isPending}
            onClick={() => avatarInputRef.current?.click()}
          >
            {uploadAvatar.isPending ? 'Uploading…' : 'Change avatar'}
          </button>
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16, marginBottom: 20 }}>
          <div className="field-row" style={{ marginBottom: 0 }}>
            <label htmlFor="name">Full Name</label>
            <input id="name" type="text" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="field-row" style={{ marginBottom: 0 }}>
            <label htmlFor="settings-email">Email</label>
            <input id="settings-email" type="text" value={me?.email ?? ''} disabled />
          </div>
        </div>

        <button
          className="btn-primary"
          type="button"
          disabled={!name.trim() || name === me?.display_name || updateProfile.isPending}
          onClick={() => updateProfile.mutate(name.trim())}
        >
          {updateProfile.isPending ? 'Saving…' : 'Save Changes'}
        </button>
      </div>

      <div className="card">
        <div className="card-title">Change Password</div>
        <div className="field-row">
          <label htmlFor="current-password">Current Password</label>
          <input
            id="current-password"
            type="password"
            value={currentPassword}
            onChange={(e) => setCurrentPassword(e.target.value)}
          />
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          <div className="field-row" style={{ marginBottom: 0 }}>
            <label htmlFor="new-password">New Password</label>
            <input id="new-password" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
            {passwordTooShort && <span style={{ fontSize: 12, color: 'var(--error)' }}>At least 8 characters</span>}
          </div>
          <div className="field-row" style={{ marginBottom: 0 }}>
            <label htmlFor="confirm-password">Confirm New Password</label>
            <input
              id="confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
            {passwordsMismatch && <span style={{ fontSize: 12, color: 'var(--error)' }}>Passwords don't match</span>}
          </div>
        </div>
        <button
          className="btn-primary"
          type="button"
          style={{ marginTop: 16 }}
          disabled={!canSubmitPassword}
          onClick={() => changePassword.mutate()}
        >
          {changePassword.isPending ? 'Changing…' : 'Change Password'}
        </button>
      </div>

      <div className="card">
        <div className="card-title">Developer Mode</div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginBottom: 16 }}>
          Shows the Developer section in the sidebar (API, API Keys, Webhooks, Usage, CLI) for integrating
          FileForge into your own applications.
        </p>
        <Switch checked={developerMode} onChange={toggleDeveloperMode} label={developerMode ? 'On' : 'Off'} />
      </div>
    </div>
  )
}
