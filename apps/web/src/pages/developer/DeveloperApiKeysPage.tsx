import { Icon } from '../../components/Icon'
import { mockApiKeys } from '../../lib/mockData'

export function DeveloperApiKeysPage() {
  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 16 }}>
        <button className="btn-primary" type="button" disabled>
          <Icon name="plus" size={14} /> Create API Key
        </button>
      </div>

      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Key</th>
              <th>Permissions</th>
              <th>Created</th>
              <th>Last Used</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {mockApiKeys.map((key) => (
              <tr key={key.id}>
                <td style={{ fontWeight: 500 }}>{key.name}</td>
                <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{key.key}</td>
                <td>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {key.permissions.map((p) => (
                      <span
                        key={p}
                        style={{
                          fontSize: 11,
                          background: 'var(--surface-2)',
                          border: '1px solid var(--border)',
                          borderRadius: 4,
                          padding: '2px 8px',
                          color: 'var(--text-muted)',
                        }}
                      >
                        {p}
                      </span>
                    ))}
                  </div>
                </td>
                <td>{key.created}</td>
                <td>{key.lastUsed}</td>
                <td>
                  <button className="btn-icon" type="button" disabled title="Revoke">
                    <Icon name="trash" size={14} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
