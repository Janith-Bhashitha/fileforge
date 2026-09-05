import { useRef, useState, type ChangeEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Icon } from '../components/Icon'
import { FileTypeIcon } from '../components/FileTypeIcon'
import { api } from '../lib/api'
import { formatBytes, formatDate, mimeTypeToLabel } from '../lib/format'
import type { FileRecord } from '../lib/types'

export function FilesPage() {
  const [query, setQuery] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const filesQuery = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get<FileRecord[]>('/api/v1/files'),
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData()
      formData.append('file', file)
      return api.upload('/api/v1/files', formData)
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['files'] }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/api/v1/files/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['files'] }),
  })

  function handleFileSelected(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (file) uploadMutation.mutate(file)
    e.target.value = ''
  }

  async function handleDownload(file: FileRecord) {
    const blob = await api.downloadBlob(`/api/v1/files/${file.id}/download`)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = file.filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  const files = filesQuery.data ?? []
  const filtered = files.filter((f) => f.filename.toLowerCase().includes(query.toLowerCase()))

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Files</h1>
          <p>Manage your uploaded and processed files.</p>
        </div>
        <div className="page-header-actions">
          <input ref={fileInputRef} type="file" style={{ display: 'none' }} onChange={handleFileSelected} />
          <button
            className="btn-primary"
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadMutation.isPending}
          >
            <Icon name="upload" size={14} /> {uploadMutation.isPending ? 'Uploading…' : 'Upload'}
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
                <th>Created</th>
                <th>Jobs</th>
                <th>Actions</th>
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
                    <td>{type}</td>
                    <td>{formatBytes(file.size)}</td>
                    <td>{formatDate(file.created_at)}</td>
                    <td>{file.derived_count > 0 ? `${file.derived_count} job${file.derived_count > 1 ? 's' : ''}` : '—'}</td>
                    <td>
                      <div className="table-actions">
                        <button type="button" onClick={() => handleDownload(file)}>
                          Download
                        </button>
                        <button type="button" onClick={() => deleteMutation.mutate(file.id)}>
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          {filesQuery.isLoading && <p className="empty-hint">Loading files…</p>}
          {!filesQuery.isLoading && filtered.length === 0 && (
            <p className="empty-hint">No files yet — upload one to get started.</p>
          )}
        </div>
      </div>
    </div>
  )
}
