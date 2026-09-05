export interface FileRecord {
  id: string
  filename: string
  mime_type: string
  size: number
  derived_from_file_id?: string
  operation?: string
  derived_count: number
  created_at: string
}
