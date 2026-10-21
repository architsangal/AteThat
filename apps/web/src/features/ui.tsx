import { createPortal } from 'react-dom'
import { createContext, useContext, useEffect, useId, useRef, useState, type FormHTMLAttributes, type ReactNode } from 'react'
import { display, label, type Nutrition } from './nutrition/model'
import { isTextTooShort } from './form-validation'

export function Loading({ children = 'Loading…' }: { children?: ReactNode }) {
  return <p className="loading-state" role="status">{children}</p>
}

// Native dialogs provide focus containment, Escape, inert background content,
// and return focus to the opener. Keep the primary action in a visible footer.
export function Modal({ title, children, onClose, busy = false, initialFocus = 'field' }: { title: string; children: ReactNode; onClose: () => void; busy?: boolean; initialFocus?: 'field' | 'heading' }) {
  const ref = useRef<HTMLDialogElement>(null)
  const titleId = useId()
  useEffect(() => {
    const dialog = ref.current!
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    dialog.showModal()
    if (initialFocus === 'heading') {
      dialog.querySelector<HTMLElement>('h2')?.focus({preventScroll:true})
      dialog.scrollTop = 0
    } else dialog.querySelector<HTMLInputElement>('input:not([disabled]), select:not([disabled]), textarea:not([disabled])')?.focus()
    return () => { dialog.close(); document.body.style.overflow = previousOverflow }
  }, [])
  return createPortal(<dialog ref={ref} className="flow-dialog" aria-labelledby={titleId} onCancel={event => { event.preventDefault(); if (!busy) onClose() }}>
    <div className="dialog-heading"><h2 id={titleId} tabIndex={initialFocus === 'heading' ? -1 : undefined}>{title}</h2><button className="icon-button" type="button" disabled={busy} onClick={onClose} aria-label={`Close ${title}`}>×</button></div>
    {children}
  </dialog>, document.body)
}

export function NutritionSummary({ values }: { values: Nutrition }) {
  const primary = ['ENERGY', 'PROTEIN', 'CARBOHYDRATE', 'FAT', 'FIBER']
  const other = Object.entries(values).filter(([code, value]) => !primary.includes(code) && value.status === 'KNOWN')
  const missing = Object.values(values).filter(value => value.status !== 'KNOWN').length
  return <><dl className="nutrition-summary">{primary.map(code => <div key={code}><dt>{label(code)}</dt><dd>{display(values[code])}</dd></div>)}</dl>
    <details className="secondary-details"><summary>More nutrition details</summary><dl className="nutrition-summary">{other.map(([code, value]) => <div key={code}><dt>{label(code)}</dt><dd>{display(value)}</dd></div>)}</dl>
      {missing > 0 && <p className="muted">{missing} nutrients have no source value. Unknown values are not counted as zero.</p>}
    </details></>
}

export function ScheduleTimes({ value, onChange }: { value: string[]; onChange: (times: string[]) => void }) {
  return <div className="schedule-times"><span>Scheduled times</span>{value.map((time, index) => <div className="time-row" key={index}>
    <label><span className="sr-only">Time {index + 1}</span><input type="time" name="schedule_time" required value={time} onInput={event => { const next = event.currentTarget.value; onChange(value.map((entry, i) => i === index ? next : entry)) }} onChange={event => { const next = event.currentTarget.value; onChange(value.map((entry, i) => i === index ? next : entry)) }} /></label>
    {value.length > 1 && <button className="quiet-link" type="button" aria-label={`Remove time ${index + 1}`} onClick={() => onChange(value.filter((_, i) => i !== index))}>Remove</button>}
  </div>)}<button className="quiet-link" type="button" disabled={value.length >= 24} onClick={() => onChange([...value, ''])}>Add another time</button></div>
}

export function MonthDays({ value, onChange }: { value: number[]; onChange: (days: number[]) => void }) {
  return <fieldset className="month-days"><legend>Dates each month</legend>{Array.from({ length: 31 }, (_, i) => i + 1).map(day => <label key={day}><input type="checkbox" checked={value.includes(day)} onChange={() => onChange(value.includes(day) ? value.filter(n => n !== day) : [...value, day].sort((a,b) => a-b))} /><span>{day}</span></label>)}<small>Dates absent from a month are skipped.</small></fieldset>
}

// One validation surface for mouse, keyboard and touch; never invoke browser bubbles.
export function Form({ children, onSubmit, ...props }: FormHTMLAttributes<HTMLFormElement>) {
  const [errors, setErrors] = useState<{field: HTMLElement; message: string}[]>([])
  return <form {...props} noValidate onInput={event => { setErrors([]); (event.target as HTMLElement).removeAttribute('aria-invalid') }} onSubmit={event => {
    const fields = Array.from(event.currentTarget.elements).filter((field): field is HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement => field instanceof HTMLInputElement || field instanceof HTMLSelectElement || field instanceof HTMLTextAreaElement)
    const tooShort = (field: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement) => isTextTooShort(field.value, Number(field.getAttribute('minlength') || 0))
    const invalid = fields.filter(field => field.willValidate && (!field.validity.valid || tooShort(field) || (field.required && !field.value.trim())))
    fields.forEach(field => field.removeAttribute('aria-invalid'))
    if (invalid.length) {
      event.preventDefault()
      const next = invalid.map(field => {
        field.setAttribute('aria-invalid', 'true')
        const label = field.getAttribute('aria-label') || Array.from(field.labels?.[0]?.childNodes || []).filter(node => node.nodeType === Node.TEXT_NODE).map(node => node.textContent).join(' ').trim() || 'This field'
        const detail = field.validity.valueMissing ? 'Please enter a value.' : (field.validity.tooShort || tooShort(field)) ? `needs at least ${field.getAttribute('minlength')} characters.` : field.validity.rangeUnderflow ? `must be at least ${field.getAttribute('min')}.` : field.validity.rangeOverflow ? `must be at most ${field.getAttribute('max')}.` : field.validity.typeMismatch ? 'needs a valid value.' : 'needs a valid value in the allowed range.'
        return { field, message: field.getAttribute('data-validation-message') || `${label}: ${detail}` }
      })
      setErrors(next);
      let ancestor: HTMLElement | null = invalid[0].parentElement
      while (ancestor) { if (ancestor instanceof HTMLDetailsElement) ancestor.open = true; ancestor = ancestor.parentElement }
      invalid[0].focus(); return
    }
    setErrors([]); onSubmit?.(event)
  }}>
    {children}
    {errors.length > 0 && <div className="validation-message" role="alert">{errors.map((error, index) => <p key={index}><button type="button" onClick={() => error.field.focus()}>{error.message}</button></p>)}</div>}
  </form>
}

type DialogField = { name: string; label: string; value?: string; type?: string; min?: string; max?: string; step?: string; optional?: boolean; minLength?: number; validationMessage?: string }
type DialogSpec = { title: string; description?: string; fields?: DialogField[]; action?: string; resolve: (value: Record<string,string> | null) => void }
const DialogContext = createContext<{
  prompt: (title: string, value?: string) => Promise<string | null>;
  confirm: (description: string) => Promise<boolean>;
  fields: (title: string, fields: DialogField[], description?: string, action?: string) => Promise<Record<string,string> | null>;
}>(null!)
export const useDialogs = () => useContext(DialogContext)
export function DialogProvider({children}: {children: ReactNode}) {
  const [spec, setSpec] = useState<DialogSpec | null>(null)
  const open = (title: string, fields?: DialogField[], description?: string, action?: string) => new Promise<Record<string,string> | null>(resolve => setSpec({title, fields, description, action, resolve}))
  const close = (value: Record<string,string> | null) => { spec?.resolve(value); setSpec(null) }
  return <DialogContext.Provider value={{
    fields: open,
    prompt: async (title, value) => { const result = await open(title, [{name:'value', label:title, value}], undefined, 'Save'); return result?.value ?? null },
    confirm: async description => { const action = /^(Remove|Delete|Disable|Replace)/.exec(description)?.[1] || 'Continue'; return !!await open(action === 'Continue' ? 'Review this change' : `${action} this item?`, undefined, description, action) },
  }}>{children}{spec && <Modal title={spec.title} onClose={() => close(null)}>
    {spec.description && <p className="muted">{spec.description}</p>}
    <Form className="auth-form" onSubmit={event => { event.preventDefault(); close(Object.fromEntries(new FormData(event.currentTarget).entries()) as Record<string,string>) }}>
      {spec.fields?.map((field,index) => <label key={field.name}>{field.label}<input name={field.name} type={field.type || 'text'} defaultValue={field.value || ''} required={!field.optional} min={field.min} max={field.max} step={field.step} minLength={field.minLength} data-validation-message={field.validationMessage} autoFocus={index === 0} /></label>)}
      <div className="dialog-actions"><button type="button" className="button button-outline" onClick={() => close(null)}>Cancel</button><button className="button button-primary" type="submit">{spec.action || 'Continue'}</button></div>
    </Form>
  </Modal>}</DialogContext.Provider>
}
