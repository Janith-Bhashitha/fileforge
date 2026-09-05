export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function formatDate(iso: string): string {
  const date = new Date(iso)
  return date.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

const mimeLabels: Record<string, string> = {
  'application/pdf': 'PDF',
  'image/jpeg': 'JPG',
  'image/png': 'PNG',
  'text/plain': 'TXT',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'DOCX',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation': 'PPTX',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'XLSX',
  'application/zip': 'ZIP',
}

export function mimeTypeToLabel(mime: string): string {
  return mimeLabels[mime] ?? 'FILE'
}
