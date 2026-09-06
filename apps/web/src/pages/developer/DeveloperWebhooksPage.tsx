import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon } from '../../components/Icon'
import { useToast } from '../../components/Toast'
import { api, ApiError } from '../../lib/api'
import { formatDate } from '../../lib/format'

// Mirrors webhooks.AllEventTypes on the API - these three are the only
// events the dispatcher actually fires.
const WEBHOOK_EVENT_TYPES = ['job.completed', 'job.failed', 'batch.completed']

interface WebhookResponse {
  id: string
  url: string
  event_types: string[]
  active: boolean
  created_at: string
  // Only ever populated on the create response.
  secret?: string
}

interface DeliveryResponse {
  id: string
  webhook_id: string
  event_type: string
  status: 'pending' | 'success' | 'failed'
  attempts: number
  response_status?: number
  last_attempted_at?: string
  created_at: string
}

export function DeveloperWebhooksPage() {
  const queryClient = useQueryClient()
  const toast = useToast()

  const [showCreateForm, setShowCreateForm] = useState(false)
  const [url, setUrl] = useState('')
  const [selectedEvents, setSelectedEvents] = useState<string[]>([])
  const [revealedSecret, setRevealedSecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [confirmingId, setConfirmingId] = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)

  const hooksQuery = useQuery({
    queryKey: ['webhooks'],
    queryFn: () => api.get<WebhookResponse[]>('/api/v1/webhooks'),
  })

  const createHook = useMutation({
    mutationFn: (payload: { url: string; event_types: string[] }) =>
      api.post<WebhookResponse>('/api/v1/webhooks', payload),
    onSuccess: (created) => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      setRevealedSecret(created.secret ?? null)
      setCopied(false)
      setUrl('')
      setSelectedEvents([])
      setShowCreateForm(false)
      toast.success('Webhook endpoint added')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to add endpoint'),
  })

  const deleteHook = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/webhooks/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['webhooks'] })
      toast.success('Webhook endpoint deleted')
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to delete endpoint'),
    onSettled: () => setConfirmingId(null),
  })

  const sendTest = useMutation({
    mutationFn: (id: string) => api.post<{ status: string }>(`/api/v1/webhooks/${id}/test`),
    onSuccess: (res, id) => {
      queryClient.invalidateQueries({ queryKey: ['webhooks', id, 'deliveries'] })
      toast.success(res.status)
    },
    onError: (err) => toast.error(err instanceof ApiError ? err.message : 'Failed to send test delivery'),
    onSettled: () => setTestingId(null),
  })

  async function handleCopy() {
    if (!revealedSecret) return
    await navigator.clipboard.writeText(revealedSecret)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  function toggleEvent(event: string) {
    setSelectedEvents((prev) => (prev.includes(event) ? prev.filter((e) => e !== event) : [...prev, event]))
  }

  const hooks = hooksQuery.data ?? []
  const canSubmit = url.trim().length > 0 && selectedEvents.length > 0 && !createHook.isPending

  return (
    <div>
      {revealedSecret && (
        <div className="banner banner-warning">
          <div style={{ minWidth: 0 }}>
            <strong>Copy your signing secret now.</strong> This is the only time it will be shown. Use it to verify
            the signature on incoming deliveries.
            <div style={{ fontFamily: 'var(--font-mono)', fontSize: 12, marginTop: 6, wordBreak: 'break-all' }}>
              {revealedSecret}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8, flexShrink: 0 }}>
            <button className="btn-secondary" type="button" onClick={handleCopy}>
              <Icon name={copied ? 'check' : 'copy'} size={14} /> {copied ? 'Copied' : 'Copy'}
            </button>
            <button className="btn-secondary" type="button" onClick={() => setRevealedSecret(null)}>
              Done
            </button>
          </div>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <button className="btn-primary" type="button" onClick={() => setShowCreateForm((prev) => !prev)}>
          <Icon name="plus" size={14} /> Add Endpoint
        </button>
      </div>

      {showCreateForm && (
        <div className="card">
          <div className="field-row">
            <label htmlFor="webhook-url">Endpoint URL</label>
            <input
              id="webhook-url"
              type="text"
              placeholder="https://myapp.com/webhooks/fileforge"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>
          <div className="field-row">
            <label>Events</label>
            <div className="pill-group">
              {WEBHOOK_EVENT_TYPES.map((event) => (
                <button
                  key={event}
                  type="button"
                  className={`pill-option ${selectedEvents.includes(event) ? 'pill-option-active' : ''}`}
                  onClick={() => toggleEvent(event)}
                >
                  {event}
                </button>
              ))}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 10 }}>
            <button
              className="btn-primary"
              type="button"
              disabled={!canSubmit}
              onClick={() => createHook.mutate({ url: url.trim(), event_types: selectedEvents })}
            >
              {createHook.isPending ? 'Adding…' : 'Add Endpoint'}
            </button>
            <button
              className="btn-secondary"
              type="button"
              onClick={() => {
                setShowCreateForm(false)
                setUrl('')
                setSelectedEvents([])
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {hooksQuery.isLoading && <p className="empty-hint">Loading endpoints…</p>}
      {hooksQuery.isError && <p className="empty-hint">Could not load endpoints.</p>}
      {!hooksQuery.isLoading && !hooksQuery.isError && hooks.length === 0 && (
        <p className="empty-hint">No endpoints yet — add one to start receiving events.</p>
      )}

      {hooks.map((hook) => (
        <div className="card" key={hook.id}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{hook.url}</span>
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  background: hook.active ? 'var(--success-soft)' : 'var(--surface-2)',
                  color: hook.active ? 'var(--success)' : 'var(--text-muted)',
                  padding: '2px 10px',
                  borderRadius: 999,
                }}
              >
                {hook.active ? 'Active' : 'Inactive'}
              </span>
            </div>
            <div className="table-actions">
              <button
                type="button"
                onClick={() => setExpandedId((prev) => (prev === hook.id ? null : hook.id))}
              >
                {expandedId === hook.id ? 'Hide logs' : 'View logs'}
              </button>
              <button
                type="button"
                disabled={sendTest.isPending && testingId === hook.id}
                onClick={() => {
                  setTestingId(hook.id)
                  sendTest.mutate(hook.id)
                }}
              >
                {sendTest.isPending && testingId === hook.id ? 'Sending…' : 'Send test'}
              </button>
              {confirmingId === hook.id ? (
                <>
                  <button
                    className="btn-danger-ghost"
                    type="button"
                    disabled={deleteHook.isPending}
                    onClick={() => deleteHook.mutate(hook.id)}
                  >
                    {deleteHook.isPending ? 'Deleting…' : 'Confirm delete'}
                  </button>
                  <button type="button" onClick={() => setConfirmingId(null)}>
                    Cancel
                  </button>
                </>
              ) : (
                <button className="btn-icon" type="button" title="Delete" onClick={() => setConfirmingId(hook.id)}>
                  <Icon name="trash" size={14} />
                </button>
              )}
            </div>
          </div>
          <div style={{ display: 'flex', gap: 6, margin: '10px 0', flexWrap: 'wrap' }}>
            {hook.event_types.map((e) => (
              <span
                key={e}
                style={{
                  fontSize: 11,
                  background: 'var(--surface-2)',
                  border: '1px solid var(--border)',
                  borderRadius: 4,
                  padding: '2px 8px',
                  color: 'var(--text-muted)',
                }}
              >
                {e}
              </span>
            ))}
          </div>
          <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>Added {formatDate(hook.created_at)}</p>

          {expandedId === hook.id && <DeliveryLog webhookId={hook.id} />}
        </div>
      ))}

      <div className="card">
        <div className="card-title">Available Events</div>
        <div className="pill-group">
          {WEBHOOK_EVENT_TYPES.map((e) => (
            <span key={e} className="pill-option" style={{ cursor: 'default' }}>
              {e}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

const DELIVERY_STATUS_COLORS: Record<DeliveryResponse['status'], string> = {
  success: 'var(--success)',
  failed: 'var(--error)',
  pending: 'var(--warning)',
}

function DeliveryLog({ webhookId }: { webhookId: string }) {
  const deliveriesQuery = useQuery({
    queryKey: ['webhooks', webhookId, 'deliveries'],
    queryFn: () => api.get<DeliveryResponse[]>(`/api/v1/webhooks/${webhookId}/deliveries`),
  })

  const deliveries = deliveriesQuery.data ?? []

  return (
    <div style={{ borderTop: '1px solid var(--border)', marginTop: 14, paddingTop: 14 }}>
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Event</th>
              <th>Status</th>
              <th>Attempts</th>
              <th>Response</th>
              <th>Last Attempt</th>
            </tr>
          </thead>
          <tbody>
            {deliveries.map((d) => (
              <tr key={d.id}>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{d.event_type}</td>
                <td>
                  <span style={{ color: DELIVERY_STATUS_COLORS[d.status], fontWeight: 600 }}>{d.status}</span>
                </td>
                <td>{d.attempts}</td>
                <td>{d.response_status ?? '—'}</td>
                <td>{d.last_attempted_at ? formatDate(d.last_attempted_at) : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {deliveriesQuery.isLoading && <p className="empty-hint">Loading deliveries…</p>}
        {deliveriesQuery.isError && <p className="empty-hint">Could not load deliveries.</p>}
        {!deliveriesQuery.isLoading && !deliveriesQuery.isError && deliveries.length === 0 && (
          <p className="empty-hint">No deliveries yet — use “Send test” to fire one.</p>
        )}
      </div>
    </div>
  )
}
