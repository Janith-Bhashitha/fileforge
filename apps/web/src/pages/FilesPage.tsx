import { useState } from 'react'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { mockFiles } from '../lib/mockData'

export function FilesPage() {
  const [query, setQuery] = useState('')

  const filtered = mockFiles.filter((f) => f.name.toLowerCase().includes(query.toLowerCase()))

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Files</h1>
          <p>Manage your uploaded and processed files.</p>
        </div>
        <div className="page-header-actions">
          <button className="btn-secondary" type="button" disabled>
            <Icon name="plus" size={14} /> New folder
          </button>
          <button className="btn-primary" type="button" disabled>
            <Icon name="upload" size={14} /> Upload
          </button>
        </div>
      </div>

      <div className="card">
        <div style={{ display: 'flex', gap: 12, marginBottom: 16, flexWrap: 'wrap' }}>
          <div className="search-input">
            <Icon name="search" size={16} />
            <input
              type="text"
              placeholder="Search files..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
        </div>

        <div className="table-wrap">
          <table className="data-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Size</th>
                <th>Modified</th>
                <th>Jobs</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((file) => (
                <tr key={file.id}>
                  <td>
                    <div className="table-file-cell">
                      <FileTypeIcon type={file.type} />
                      <span className="table-file-name">{file.name}</span>
                    </div>
                  </td>
                  <td>{file.type}</td>
                  <td>{file.size}</td>
                  <td>{file.modified}</td>
                  <td>{file.jobs > 0 ? `${file.jobs} job${file.jobs > 1 ? 's' : ''}` : '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {filtered.length === 0 && <p className="empty-hint">No files match your search.</p>}
        </div>
      </div>
    </div>
  )
}
