import { useId, type InputHTMLAttributes, type ReactNode } from 'react'

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'type' | 'children'> & {
  label: string
  description?: ReactNode
  badge?: string
  className?: string
}

// Keep native checkbox semantics, label activation and keyboard handling.
export function CheckOption({ label, description, badge, className = '', id, ...input }: Props) {
  const generatedId = useId(), controlId = id || generatedId
  const descriptionId = description ? `${controlId}-description` : undefined
  return <label className={`check-option ${className}`} htmlFor={controlId}>
    <input {...input} id={controlId} type="checkbox" aria-labelledby={`${controlId}-label`} aria-describedby={[input['aria-describedby'], descriptionId].filter(Boolean).join(' ') || undefined} />
    <span className="check-option-copy"><span id={`${controlId}-label`}>{label}</span>{description && <small id={descriptionId}>{description}</small>}</span>
    {badge && <span className="check-option-badge">{badge}</span>}
  </label>
}
