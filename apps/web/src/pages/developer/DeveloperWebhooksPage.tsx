import { Icon } from '../../components/Icon'
import { mockWebhooks, availableWebhookEvents } from '../../lib/mockData'

export function DeveloperWebhooksPage() {
  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <button className="btn-primary" type="button" disabled>
          <Icon name="plus" size={14} /> Add Endpoint
        </button>
      </div>

      {mockWebhooks.map((hook) => (
        <div className="card" key={hook.id}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 12 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
              <span style={{ fontFamily: 'var(--font-mono)', fontSize: 13 }}>{hook.url}</span>
              <span
                style={{
                  fontSize: 11,
                  fontWeight: 600,
                  background: 'var(--success-soft)',
                  color: 'var(--success)',
                  padding: '2px 10px',
                  borderRadius: 999,
                }}
              >
                {hook.active ? 'Active' : 'Inactive'}
              </span>
            </div>
            <div className="table-actions">
              <button type="button" disabled>
                View logs
              </button>
              <button type="button" disabled>
                Send test
              </button>
              <button className="btn-icon" type="button" disabled title="Delete">
                <Icon name="trash" size={14} />
              </button>
            </div>
          </div>
          <div style={{ display: 'flex', gap: 6, margin: '10px 0' }}>
            {hook.events.map((e) => (
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
          <p style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            Last delivery: {hook.lastDelivery} · Success rate: {hook.successRate}
          </p>
        </div>
      ))}

      <div className="card">
        <div className="card-title">Available Events</div>
        <div className="pill-group">
          {availableWebhookEvents.map((e) => (
            <span key={e} className="pill-option" style={{ cursor: 'default' }}>
              {e}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}
