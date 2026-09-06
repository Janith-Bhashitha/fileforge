import { Navigate, Link } from 'react-router-dom'
import { useAuth } from '../lib/auth'
import { Icon } from '../components/Icon'
import './LandingPage.css'

// The tools listed here are deliberately only ones that actually work -
// no roadmap items, no "coming soon" - a first impression is the wrong
// place to oversell.
const featureCards = [
  { icon: 'convert' as const, title: 'Convert Anything', body: 'Images, Office documents and PDFs, in either direction, in seconds.' },
  { icon: 'batch' as const, title: 'Batch Processing', body: 'Apply one operation to dozens of files at once, with per-file progress.' },
  { icon: 'ai' as const, title: 'AI Analysis', body: 'Summarize, classify and tag documents automatically.' },
  { icon: 'ocr' as const, title: 'OCR', body: 'Pull real text out of scanned documents and photos.' },
  { icon: 'signature' as const, title: 'Edit & Sign PDFs', body: 'Add text, stamps and signatures directly onto any page.' },
  { icon: 'lock' as const, title: 'Password Protection', body: 'Lock sensitive PDFs, or unlock ones you already have the password for.' },
]

export function LandingPage() {
  const { isAuthenticated } = useAuth()
  if (isAuthenticated) return <Navigate to="/dashboard" replace />

  return (
    <div className="landing">
      <header className="landing-nav">
        <div className="landing-brand">
          <div className="landing-brand-mark">FF</div>
          <span>FileForge</span>
        </div>
        <nav className="landing-nav-links">
          <a href="#features">Features</a>
          <a href="#tools">Tools</a>
        </nav>
        <div className="landing-nav-actions">
          <Link to="/login" className="landing-link-btn">
            Sign in
          </Link>
          <Link to="/register" className="btn-primary">
            Get Started
          </Link>
        </div>
      </header>

      <section className="landing-hero">
        <div className="landing-hero-text">
          <span className="landing-eyebrow">Transform · Process · Understand</span>
          <h1>
            Everything you need
            <br />
            for your files. <span className="landing-accent-text">In one place.</span>
          </h1>
          <p>
            Convert, compress, merge, extract, and intelligently process your files — with AI-powered
            analysis built in.
          </p>
          <div className="landing-hero-actions">
            <Link to="/register" className="btn-primary landing-cta">
              Get Started Free <Icon name="arrow-right" size={16} />
            </Link>
            <a href="#features" className="landing-cta-secondary">
              Explore Features
            </a>
          </div>
          <div className="landing-badges">
            <span>
              <Icon name="check" size={14} /> Free to use
            </span>
            <span>
              <Icon name="lock" size={14} /> Private by default
            </span>
            <span>
              <Icon name="sparkles" size={14} /> AI enhanced
            </span>
          </div>
        </div>
        <div className="landing-hero-visual" aria-hidden="true">
          <div className="landing-hero-glow" />
          <div className="landing-file-stack">
            <div className="landing-file landing-file-pdf">PDF</div>
            <div className="landing-file landing-file-doc">W</div>
            <div className="landing-file landing-file-xls">X</div>
            <div className="landing-file landing-file-main">
              <div className="landing-brand-mark landing-brand-mark-lg">FF</div>
            </div>
            <div className="landing-file landing-file-img">IMG</div>
            <div className="landing-file landing-file-ppt">P</div>
          </div>
        </div>
      </section>

      <section className="landing-features" id="features">
        <h2>One tool, every format</h2>
        <p className="landing-section-sub">No roadmap items here — everything below works today.</p>
        <div className="landing-feature-grid" id="tools">
          {featureCards.map((f) => (
            <div className="landing-feature-card" key={f.title}>
              <div className="landing-feature-icon">
                <Icon name={f.icon} size={20} />
              </div>
              <h3>{f.title}</h3>
              <p>{f.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="landing-cta-section">
        <h2>Ready to get started?</h2>
        <p>Create a free account — no credit card required.</p>
        <Link to="/register" className="btn-primary landing-cta">
          Get Started Free <Icon name="arrow-right" size={16} />
        </Link>
      </section>

      <footer className="landing-footer">
        <div className="landing-brand">
          <div className="landing-brand-mark">FF</div>
          <span>FileForge</span>
        </div>
        <span className="landing-footer-copy">Built for people who deal with too many file formats.</span>
      </footer>
    </div>
  )
}
