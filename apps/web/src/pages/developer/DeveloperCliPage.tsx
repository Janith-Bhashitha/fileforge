const commands = [
  { cmd: 'fileforge convert input.docx --to pdf', desc: 'Convert a single file' },
  { cmd: 'fileforge batch ./invoices --to pdf --zip', desc: 'Batch convert a directory' },
  { cmd: 'fileforge ocr invoice.pdf --lang en', desc: 'Run OCR on a document' },
  { cmd: 'fileforge job status <job-id>', desc: 'Check a job\'s status' },
]

export function DeveloperCliPage() {
  return (
    <div>
      <div className="card">
        <div className="card-title">Installation</div>
        <div className="code-block-header">bash</div>
        <div className="code-block">npm install -g fileforge-cli</div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 10 }}>
          Or with pnpm: <code>pnpm add -g fileforge-cli</code>
        </p>
      </div>

      <div className="card">
        <div className="card-title">Authentication</div>
        <div className="code-block-header">bash</div>
        <div className="code-block">{`fileforge login
# Or with an API key:
fileforge config --api-key ff_live_...`}</div>
      </div>

      <div className="card">
        <div className="card-title">Commands</div>
        {commands.map((c) => (
          <div key={c.cmd} style={{ marginBottom: 14 }}>
            <div className="code-block">{c.cmd}</div>
            <p style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 4 }}>{c.desc}</p>
          </div>
        ))}
      </div>
    </div>
  )
}
