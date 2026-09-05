import './Switch.css'

interface SwitchProps {
  checked: boolean
  onChange: (checked: boolean) => void
  label?: string
}

export function Switch({ checked, onChange, label }: SwitchProps) {
  return (
    <label className="switch-row">
      <span className="switch" data-checked={checked}>
        <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />
        <span className="switch-track">
          <span className="switch-thumb" />
        </span>
      </span>
      {label && <span className="switch-label">{label}</span>}
    </label>
  )
}
