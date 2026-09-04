import { useState } from 'react'
import { Icon } from '../components/Icon'

const categories = [
  { emoji: '🚀', name: 'Getting Started', count: 12 },
  { emoji: '🔄', name: 'File Conversion', count: 24 },
  { emoji: '⚡', name: 'Batch Processing', count: 8 },
  { emoji: '🤖', name: 'AI Processing', count: 10 },
  { emoji: '🔍', name: 'OCR', count: 6 },
  { emoji: '🔌', name: 'API & Developer', count: 18 },
  { emoji: '💳', name: 'Billing', count: 9 },
  { emoji: '⚙️', name: 'Account', count: 14 },
]

const faqs = [
  { q: 'What file formats does FileForge support?', a: 'Phase 1 currently supports account registration and login only. Image, PDF, and Office format conversion arrives in Phase 2.' },
  { q: 'How large can my files be?', a: 'File size limits will be configurable per plan once quotas ship in Phase 5 (Production Hardening).' },
  { q: 'How long are my files stored?', a: 'Temporary file retention policies are part of Phase 5. For now, only your account data is stored.' },
  { q: 'How does AI Processing work?', a: 'AI-assisted OCR, classification, and natural-language batch planning ship in Phases 7 and 8.' },
  { q: 'Can I use FileForge via API?', a: 'A public API with keys and webhooks is planned for Phase 9.' },
]

export function HelpPage() {
  const [query, setQuery] = useState('')
  const [openIndex, setOpenIndex] = useState<number | null>(null)

  return (
    <div>
      <div className="page-header">
        <div>
          <h1>Help Center</h1>
          <p>Find answers to your questions.</p>
        </div>
      </div>

      <div className="card" style={{ textAlign: 'center', marginBottom: 24 }}>
        <h2 style={{ fontSize: 17, fontWeight: 600, marginBottom: 14 }}>How can we help?</h2>
        <div className="search-input" style={{ margin: '0 auto', maxWidth: 480 }}>
          <Icon name="search" size={16} />
          <input
            type="text"
            placeholder="Search help articles..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      <div className="help-category-grid" style={{ marginBottom: 24 }}>
        {categories.map((c) => (
          <div className="help-category" key={c.name}>
            <div className="help-category-emoji">{c.emoji}</div>
            <div className="help-category-name">{c.name}</div>
            <div className="help-category-count">{c.count} articles</div>
          </div>
        ))}
      </div>

      <div className="card">
        <div className="card-title">Frequently Asked Questions</div>
        {faqs.map((faq, i) => (
          <div className="faq-item" key={faq.q}>
            <button
              type="button"
              className={`faq-question ${openIndex === i ? 'faq-question-open' : ''}`}
              onClick={() => setOpenIndex(openIndex === i ? null : i)}
            >
              {faq.q}
              <Icon name="chevron" size={16} />
            </button>
            {openIndex === i && <div className="faq-answer">{faq.a}</div>}
          </div>
        ))}
      </div>
    </div>
  )
}
