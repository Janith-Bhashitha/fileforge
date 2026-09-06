// There is no FileForge CLI to document. This shows the same workflow with
// curl against the real endpoints instead of advertising a binary nobody
// can install.
export function DeveloperCliPage() {
  return (
    <div>
      <div className="card">
        <div className="card-title">1. Upload a file</div>
        <div className="code-block-header">bash</div>
        <div className="code-block">{`curl -X POST https://YOUR-HOST/api/v1/files \\
  -H "X-API-Key: ffk_..." \\
  -F "file=@input.docx"`}</div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 10 }}>
          Returns the file record. Keep its <code>id</code> for the next step.
        </p>
      </div>

      <div className="card">
        <div className="card-title">2. Convert it</div>
        <div className="code-block-header">bash</div>
        <div className="code-block">{`curl -X POST https://YOUR-HOST/api/v1/convert \\
  -H "X-API-Key: ffk_..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "file_id": "PASTE-THE-ID",
    "operation": "docx-to-pdf",
    "version": "v1"
  }'`}</div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', marginTop: 10 }}>
          Runs synchronously and returns the output file record. For many files at once, post to{' '}
          <code>/api/v1/batches</code> instead and poll the batch.
        </p>
      </div>

      <div className="card">
        <div className="card-title">3. Download the result</div>
        <div className="code-block-header">bash</div>
        <div className="code-block">{`curl https://YOUR-HOST/api/v1/files/OUTPUT-ID/download \\
  -H "X-API-Key: ffk_..." \\
  -o output.pdf`}</div>
      </div>

      <div className="card">
        <div className="card-title">Operation names</div>
        <p style={{ fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.7 }}>
          <code>image-to-pdf</code>, <code>pdf-to-image</code>, <code>docx-to-pdf</code>,{' '}
          <code>pptx-to-pdf</code>, <code>xlsx-to-pdf</code>, <code>txt-to-pdf</code>, <code>pdf-merge</code>,{' '}
          <code>pdf-split</code>, <code>pdf-compress</code>, <code>pdf-rotate</code>,{' '}
          <code>pdf-remove-pages</code>, <code>pdf-extract-pages</code>, <code>pdf-watermark</code>,{' '}
          <code>pdf-protect</code>, <code>pdf-unlock</code>, <code>pdf-overlay</code>, <code>ocr</code>,{' '}
          <code>document-insights</code>, <code>ai-analyze</code>, <code>image-convert</code>,{' '}
          <code>image-resize</code>
        </p>
      </div>
    </div>
  )
}
