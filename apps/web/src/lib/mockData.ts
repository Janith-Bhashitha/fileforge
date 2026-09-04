export type JobStatus = 'completed' | 'processing' | 'queued' | 'failed'

export interface MockFile {
  id: string
  name: string
  type: string
  size: string
  modified: string
  jobs: number
}

export const mockFiles: MockFile[] = [
  { id: '1', name: 'invoice_2026_001.pdf', type: 'PDF', size: '1.2 MB', modified: '2 min ago', jobs: 3 },
  { id: '2', name: 'project-proposal.docx', type: 'DOCX', size: '842 KB', modified: '5 min ago', jobs: 1 },
  { id: '3', name: 'company-logo.png', type: 'PNG', size: '540 KB', modified: '12 min ago', jobs: 2 },
  { id: '4', name: 'financial-report.xlsx', type: 'XLSX', size: '2.1 MB', modified: '1 hour ago', jobs: 1 },
  { id: '5', name: 'meeting-notes.docx', type: 'DOCX', size: '156 KB', modified: '2 hours ago', jobs: 0 },
  { id: '6', name: 'presentation.pptx', type: 'PPTX', size: '8.4 MB', modified: '3 hours ago', jobs: 1 },
  { id: '7', name: 'product-photo.jpg', type: 'JPG', size: '2.8 MB', modified: '2 days ago', jobs: 1 },
]

export interface MockJob {
  id: string
  file: string
  type: string
  operation: string
  status: JobStatus
  created: string
  duration: string
  output: string
}

export const mockHistory: MockJob[] = [
  {
    id: 'job_01J9A3kx',
    file: 'invoice_2026_001.pdf',
    type: 'PDF',
    operation: 'PDF → Searchable PDF',
    status: 'completed',
    created: '2026-09-04 09:14',
    duration: '4.2s',
    output: '1.4 MB',
  },
  {
    id: 'job_01J9A2mx',
    file: 'project-proposal.docx',
    type: 'DOC',
    operation: 'DOCX → PDF',
    status: 'processing',
    created: '2026-09-04 09:09',
    duration: '—',
    output: '—',
  },
  {
    id: 'job_01J9A1nx',
    file: 'company-logo.png',
    type: 'PNG',
    operation: 'PNG → PDF',
    status: 'queued',
    created: '2026-09-04 09:08',
    duration: '—',
    output: '—',
  },
  {
    id: 'job_01J9A0px',
    file: 'financial-report.xlsx',
    type: 'XLS',
    operation: 'XLSX → PDF',
    status: 'completed',
    created: '2026-09-04 08:15',
    duration: '6.8s',
    output: '2.3 MB',
  },
  {
    id: 'job_01J98qxr',
    file: 'meeting-notes.docx',
    type: 'DOC',
    operation: 'DOCX → PDF',
    status: 'failed',
    created: '2026-09-04 07:22',
    duration: '—',
    output: '—',
  },
]

export interface BatchItem {
  id: string
  file: string
  type: string
  operation: string
  status: JobStatus
  duration: string
  output: string
  error: string
}

export const mockBatchItems: BatchItem[] = [
  {
    id: '1',
    file: 'invoice_2026_001.pdf',
    type: 'PDF',
    operation: 'PDF → Searchable PDF',
    status: 'completed',
    duration: '4.2s',
    output: 'invoice_2026_001_ocr.pdf',
    error: '—',
  },
  {
    id: '2',
    file: 'invoice_2026_002.pdf',
    type: 'PDF',
    operation: 'PDF → Searchable PDF',
    status: 'completed',
    duration: '3.8s',
    output: 'invoice_2026_002_ocr.pdf',
    error: '—',
  },
  {
    id: '3',
    file: 'invoice_2026_003.pdf',
    type: 'PDF',
    operation: 'PDF → Searchable PDF',
    status: 'processing',
    duration: '—',
    output: '—',
    error: '—',
  },
  {
    id: '4',
    file: 'receipt_001.jpg',
    type: 'JPG',
    operation: 'JPG → PDF',
    status: 'failed',
    duration: '—',
    output: '—',
    error: 'Corrupted file',
  },
  {
    id: '5',
    file: 'receipt_002.jpg',
    type: 'JPG',
    operation: 'JPG → PDF',
    status: 'failed',
    duration: '—',
    output: '—',
    error: 'Unsupported encoding',
  },
  {
    id: '6',
    file: 'invoice_2026_004.pdf',
    type: 'PDF',
    operation: 'PDF → Searchable PDF',
    status: 'queued',
    duration: '—',
    output: '—',
    error: '—',
  },
]

export interface ApiKeyRow {
  id: string
  name: string
  key: string
  permissions: string[]
  created: string
  lastUsed: string
}

export const mockApiKeys: ApiKeyRow[] = [
  {
    id: '1',
    name: 'Production API Key',
    key: 'ff_live_01J9A3kx...',
    permissions: ['files', 'jobs', 'batches'],
    created: '2026-08-01',
    lastUsed: '2 min ago',
  },
  {
    id: '2',
    name: 'Development Key',
    key: 'ff_test_01J8F4mx...',
    permissions: ['files', 'jobs'],
    created: '2026-07-15',
    lastUsed: '1 hour ago',
  },
]

export interface WebhookRow {
  id: string
  url: string
  events: string[]
  active: boolean
  lastDelivery: string
  successRate: string
}

export const mockWebhooks: WebhookRow[] = [
  {
    id: '1',
    url: 'https://myapp.com/webhooks/fileforge',
    events: ['job.completed', 'job.failed'],
    active: true,
    lastDelivery: '2 min ago',
    successRate: '99.2%',
  },
]

export const availableWebhookEvents = [
  'job.completed',
  'job.failed',
  'batch.completed',
  'batch.failed',
  'file.processed',
  'file.deleted',
]
