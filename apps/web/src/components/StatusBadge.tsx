import './StatusBadge.css'

export type Status =
  | 'completed'
  | 'processing'
  | 'queued'
  | 'failed'
  | 'created'
  | 'retry_pending'
  | 'cancel_requested'
  | 'cancelled'
  | 'partially_completed'

const labels: Record<Status, string> = {
  completed: 'Completed',
  processing: 'Processing',
  queued: 'Queued',
  failed: 'Failed',
  created: 'Queued',
  retry_pending: 'Retrying',
  cancel_requested: 'Cancelling',
  cancelled: 'Cancelled',
  partially_completed: 'Partially Completed',
}

export function StatusBadge({ status }: { status: Status }) {
  return <span className={`status-badge status-badge-${status}`}>{labels[status]}</span>
}
