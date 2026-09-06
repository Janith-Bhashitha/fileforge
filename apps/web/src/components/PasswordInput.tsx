import { useState } from 'react'
import { Icon } from './Icon'
import './PasswordInput.css'

interface PasswordInputProps {
  id: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  autoComplete?: string
  required?: boolean
  minLength?: number
}

// A password field with a reveal toggle. Renders only the input and button,
// never a label or wrapper: the app has two field wrappers (.field on the
// auth pages, .field-row in settings) and both style their input by
// descendant selector, so this inherits either one.
export function PasswordInput({
  id,
  value,
  onChange,
  placeholder,
  autoComplete,
  required,
  minLength,
}: PasswordInputProps) {
  const [revealed, setRevealed] = useState(false)

  return (
    <div className="password-field">
      <input
        id={id}
        type={revealed ? 'text' : 'password'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        autoComplete={autoComplete}
        required={required}
        minLength={minLength}
      />
      <button
        type="button"
        className="password-reveal"
        onClick={() => setRevealed((prev) => !prev)}
        aria-label={revealed ? 'Hide password' : 'Show password'}
        aria-pressed={revealed}
        tabIndex={-1}
      >
        <Icon name={revealed ? 'eye-off' : 'eye'} size={16} />
      </button>
    </div>
  )
}
