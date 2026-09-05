import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { StatusBadge } from '../components/StatusBadge'
import { api } from '../lib/api'
import { formatBytes, formatDate, mimeTypeToLabel } from '../lib/format'
import type { FileRecord } from '../lib/types'

export function HistoryPage() {
  const [query, setQuery] = useState('')

  const filesQuery = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get<FileRecord[]>('/api/v1/files'),
  })

  // A file only shows up here if it was produced by a conversion
  // (derived_from_file_id set) — plain uploads belong to the Files page.
  const history = (filesQuery.data ?? []).filter((f) => f.derived_from_file_id)
  const filtered = history.filter((f) => f.filename.toLowerCase().includes(query.toLowerCase()))

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Processing History</h1>
          <p>Every file produced by a conversion.</p>
        </div>
      </div>

      <div className="card">
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
          <div className="search-input">
            <Icon name="search" size={16} />
            <input
              type="text"
              placeholder="Search output files..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        </div>

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>File</th>
                <th>Operation</th>
                <th>Status</th>
                <th>Created</th>
                <th>Size</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((file) => {
                const type = mimeTypeToLabel(file.mime_type)
                return (
                  <tr key={file.id}>
                    <td>
                      <div className="table-file-cell">
                        <FileTypeIcon type={type} />
                        <span className="table-file-name">{file.filename}</span>
                      </div>
                    </td>
                    <td>{file.operation ?? '—'}</td>
                    <td>
                      <StatusBadge status="completed" />
                    </td>
                    <td>{formatDate(file.created_at)}</td>
                    <td>{formatBytes(file.size)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          {filesQuery.isLoading && <p className="empty-hint">Loading history…</p>}
          {!filesQuery.isLoading && filtered.length === 0 && (
            <p className="empty-hint">No conversions yet — try converting a file first.</p>
          )}
        </div>
      </div>
    </div>
  )
}
