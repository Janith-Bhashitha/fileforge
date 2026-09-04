import { useState } from 'react'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { StatusBadge } from '../components/StatusBadge'
import { mockHistory, type JobStatus } from '../lib/mockData'

const statuses: (JobStatus | 'all')[] = ['all', 'completed', 'processing', 'queued', 'failed']

export function HistoryPage() {
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState<JobStatus | 'all'>('all')

  const filtered = mockHistory.filter((job) => {
    const matchesQuery = job.file.toLowerCase().includes(query.toLowerCase()) || job.id.includes(query)
    const matchesStatus = status === 'all' || job.status === status
    return matchesQuery && matchesStatus
  })

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Processing History</h1>
          <p>All completed, failed, and active processing jobs.</p>
        </div>
      </div>

      <div className="card">
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
          <div className="search-input">
            <Icon name="search" size={16} />
            <input
              type="text"
              placeholder="Search files or job IDs..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <select
            value={status}
            onChange={(e) => setStatus(e.target.value as JobStatus | 'all')}
            style={{
              background: 'var(--surface-2)',
              border: '1px solid var(--border)',
              borderRadius: 'var(--radius-sm)',
              padding: '9px 12px',
              fontSize: 13,
              color: 'var(--text)',
            }}
          >
            {statuses.map((s) => (
              <option key={s} value={s}>
                {s === 'all' ? 'All statuses' : s[0].toUpperCase() + s.slice(1)}
              </option>
            ))}
          </select>
        </div>

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Job ID</th>
                <th>File</th>
                <th>Operation</th>
                <th>Status</th>
                <th>Created</th>
                <th>Duration</th>
                <th>Output</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((job) => (
                <tr key={job.id}>
                  <td style={{ fontFamily: 'var(--font-mono)', fontSize: 12 }}>{job.id}</td>
                  <td>
                    <div className="table-file-cell">
                      <FileTypeIcon type={job.type} />
                      <span className="table-file-name">{job.file}</span>
                    </div>
                  </td>
                  <td>{job.operation}</td>
                  <td>
                    <StatusBadge status={job.status} />
                  </td>
                  <td>{job.created}</td>
                  <td>{job.duration}</td>
                  <td>{job.output}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {filtered.length === 0 && <p className="empty-hint">No jobs match your filters.</p>}
        </div>
      </div>
    </div>
  )
}
