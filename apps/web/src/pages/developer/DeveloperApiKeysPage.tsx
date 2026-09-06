import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon } from '../../components/Icon'
import { useToast } from '../../components/Toast'
import { api, ApiError } from '../../lib/api'
import { formatDate } from '../../lib/format'

interface ApiKeyResponse {
  id: string
  name: string
  key_prefix: string
  created_at: string
  last_used_at?: string
  revoked: boolean
  // Only ever populated on the create response - the raw key is not
  // recoverable afterward, so it has to be surfaced there and then.
  key?: string
}

export function DeveloperApiKeysPage() {
  const queryClient = useQueryClient()
  const toast = useToast()

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [name, setName] = useState('')
  const [revealedKey, setRevealedKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [confirmingId, setConfirmingId] = useState<string | null>(null)

  const keysQuery = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => api.get<ApiKeyResponse[]>('/api/v1/api-keys'),
  })

  const createKey = useMutation({
    mutationFn: (keyName: string) => api.post<ApiKeyResponse>('/api/v1/api-keys', { name: keyName }),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      setRevealedKey(created.key ?? null)
      setCopied(false)
      setName('')
      setShowCreateForm(false)
      toast.success('API key created')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to create API key'),
  })

  const revokeKey = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/api-keys/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      toast.success('API key revoked')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to revoke API key'),
    onSettled: () => setConfirmingId(null),
  })

  async function handleCopy() {
    if (!revealedKey) return
    await navigator.clipboard.writeText(revealedKey)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const keys = keysQuery.data ?? []

  return (
    <div>
      {revealedKey && (
        <div className="banner banner-warning">
          <div style={{ minWidth: 0 }}>
            <strong>Copy your API key now.</strong> This is the only time it will be shown.
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: 12, marginTop: 6, wordBreak: 'break-all' }}>
              {revealedKey}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
            <button className="btn-secondary" type="button" onClick={handleCopy}>
              <Icon name={copied ? 'check' : 'copy'} size={14} /> {copied ? 'Copied' : 'Copy'}
            </button>
            <button className="btn-secondary" type="button" onClick={() => setRevealedKey(null)}>
              Done
            </button>
          </div>
        </div>
      )}

      <div className="card">
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
          <button className="btn-primary" type="button" onClick={() => setShowCreateForm((prev) => !prev)}>
            <Icon name="plus" size={14} /> Create API Key
          </button>
        </div>

        {showCreateForm && (
          <div style={{ borderBottom: '1px solid var(--border)', paddingBottom: 16, marginBottom: 16 }}>
            <div className="field-row">
              <label htmlFor="api-key-name">Key name</label>
              <input
                id="api-key-name"
                type="text"
                placeholder="e.g. Production server"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div style={{ display: 'flex', gap: 10 }}>
              <button
                className="btn-primary"
                type="button"
                disabled={!name.trim() || createKey.isPending}
                onClick={() => createKey.mutate(name.trim())}
              >
                {createKey.isPending ? 'Creating…' : 'Create'}
              </button>
              <button
                className="btn-secondary"
                type="button"
                onClick={() => {
                  setShowCreateForm(false)
                  setName('')
                }}
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Key</th>
                <th>Created</th>
                <th>Last Used</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key) => (
                <tr key={key.id} style={key.revoked ? { opacity: 0.55 } : undefined}>
                  <td style={{ fontWeight: 500 }}>{key.name}</td>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{key.key_prefix}…</td>
                  <td>{formatDate(key.created_at)}</td>
                  <td>{key.last_used_at ? formatDate(key.last_used_at) : 'Never'}</td>
                  <td>
                    <span style={{ color: key.revoked ? 'var(--text-muted)' : 'var(--success)', fontWeight: 600 }}>
                      {key.revoked ? 'Revoked' : 'Active'}
                    </span>
                  </td>
                  <td>
                    {key.revoked ? (
                      <span style={{ color: 'var(--text-muted)' }}>—</span>
                    ) : confirmingId === key.id ? (
                      <div className="table-actions">
                        <button
                          className="btn-danger-ghost"
                          type="button"
                          disabled={revokeKey.isPending}
                          onClick={() => revokeKey.mutate(key.id)}
                        >
                          {revokeKey.isPending ? 'Revoking…' : 'Confirm revoke'}
                        </button>
                        <button type="button" onClick={() => setConfirmingId(null)}>
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <button className="btn-icon" type="button" title="Revoke" onClick={() => setConfirmingId(key.id)}>
                        <Icon name="trash" size={14} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {keysQuery.isLoading && <p className="empty-hint">Loading API keys…</p>}
          {keysQuery.isError && <p className="empty-hint">Could not load API keys.</p>}
          {!keysQuery.isLoading && !keysQuery.isError && keys.length === 0 && (
            <p className="empty-hint">No API keys yet — create one to get started.</p>
          )}
        </div>
      </div>
    </div>
  )
}
