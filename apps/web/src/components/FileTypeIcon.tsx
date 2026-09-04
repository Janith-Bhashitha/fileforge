import './FileTypeIcon.css'

const colorMap: Record<string, string> = {
  PDF: 'red',
  DOC: 'blue',
  DOCX: 'blue',
  PNG: 'purple',
  JPG: 'purple',
  JPEG: 'purple',
  XLS: 'green',
  XLSX: 'green',
  PPT: 'orange',
  PPTX: 'orange',
  CSV: 'teal',
  TXT: 'gray',
  ZIP: 'yellow',
}

export function FileTypeIcon({ type }: { type: string }) {
  const normalized = type.toUpperCase()
  const color = colorMap[normalized] ?? 'gray'
  return <span className={`file-type-icon file-type-icon-${color}`}>{normalized.slice(0, 3)}</span>
}
