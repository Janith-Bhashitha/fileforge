import './StatusBadge.css'

export type Status = 'completed' | 'processing' | 'queued' | 'failed'

const labels: Record<Status, string> = {
  completed: 'Completed',
  processing: 'Processing',
  queued: 'Queued',
  failed: 'Failed',
}

export function StatusBadge({ status }: { status: Status }) {
  return <span className={`status-badge status-badge-${status}`}>{labels[status]}</span>
}
