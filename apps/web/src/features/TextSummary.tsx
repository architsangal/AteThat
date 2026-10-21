import { useEffect, useId, useRef, useState, type RefObject } from 'react'
import { Check, ChevronDown, Copy, FileText, LoaderCircle, RefreshCw } from 'lucide-react'
import { getAuthenticatedJson } from '../api'
import { useDraftGuard } from './useDraftGuard'
import { useDialogs } from './ui'
import { label as nutrientLabel } from './nutrition/model'
import { copySummaryText } from './summary-clipboard'

type Summary = { text: string; context: string; generated_at: string; partial: boolean }
type SummaryCardProps = {
  from: string; to: string; mode: 'DAY' | 'RANGE'; nutrient: string
  source: Summary | null; draft: string; busy: boolean; error: string; status: string
  expanded: boolean; dirty: boolean; stale: boolean
  editorRef?: RefObject<HTMLTextAreaElement | null>
  onToggle: () => void; onChange: (value: string) => void; onCopy: () => void; onRegenerate: () => void
}
function displayDay(day: string) {
  const date = new Date(day + 'T00:00:00Z')
  return Number.isNaN(date.getTime()) ? day : new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' }).format(date)
}
function generatedLabel(instant: string) {
  const date = new Date(instant)
  return Number.isNaN(date.getTime()) ? 'Report ready' : 'Updated ' + new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit', timeZone: 'UTC' }).format(date) + ' UTC'
}

// Shared presentation keeps Dashboard and Analysis identical, including collapsed actions.
export function TextSummaryCard({ from, to, mode, nutrient, source, draft, busy, error, status, expanded, dirty, stale, editorRef, onToggle, onChange, onCopy, onRegenerate }: SummaryCardProps) {
  const id = useId(), titleId = id + '-title', editorId = id + '-editor', panelId = id + '-panel', hintId = id + '-hint'
  return <section className={'panel text-summary' + (expanded ? ' text-summary-expanded' : '')} aria-labelledby={titleId}>
    <div className="text-summary-header">
      <div className="text-summary-intro"><span className="text-summary-icon" aria-hidden="true"><FileText size={21} /></span><div><h2 id={titleId}>Text Summary</h2><p>Your records, ready to copy or personalize.</p></div></div>
      <div className="text-summary-actions" role="group" aria-label="Summary actions">
        <button type="button" className="button button-primary" disabled={!draft} onClick={onCopy}>{status === 'Full edited summary copied.' ? <Check size={16} aria-hidden="true" /> : <Copy size={16} aria-hidden="true" />}Copy full summary</button>
        <button type="button" className="button button-outline" disabled={busy || !from || !to} onClick={onRegenerate}>{busy ? <LoaderCircle size={16} className="text-summary-spinner" aria-hidden="true" /> : <RefreshCw size={16} aria-hidden="true" />}{busy ? 'Generating…' : 'Regenerate summary'}</button>
      </div>
    </div>
    <div className="text-summary-meta"><span>{mode === 'DAY' ? displayDay(from) : displayDay(from) + ' – ' + displayDay(to)}</span>{mode === 'RANGE' && <span>Analysis · {nutrient === 'ALL' ? 'All nutrients' : nutrientLabel(nutrient)}</span>}<span className={'text-summary-state' + (dirty ? ' text-summary-state-edited' : '')}>{busy ? 'Preparing report' : dirty ? 'Edited draft' : source ? 'Report ready' : 'No report yet'}</span></div>
    {stale && <p className="form-message target-notice" role="status">Your edited report is from the previous selection. Copy it to keep your changes, or regenerate for these dates.</p>}
    {source?.partial && <p className="form-message target-notice" role="status">Some sections are incomplete. Expand the report to see its coverage notes.</p>}
    {error && <p className="form-message form-error" role="alert">{error}</p>}
    <div className="text-summary-disclosure">
      <button type="button" className="text-summary-toggle" aria-expanded={expanded} aria-controls={panelId} onClick={onToggle}><span><strong>{expanded ? 'Hide summary' : 'View & edit summary'}</strong><small>{expanded ? 'Edits stay on this page and never change your records.' : 'Expand the full report. Copy and regenerate work while it’s closed.'}</small></span><ChevronDown size={20} aria-hidden="true" /></button>
      <div id={panelId} className="text-summary-editor" hidden={!expanded}>
        <div className="text-summary-editor-heading"><label htmlFor={editorId}>Editable factual summary</label><span>Not saved to your records</span></div>
        <textarea id={editorId} ref={editorRef} readOnly={busy} value={draft} onChange={e => onChange(e.target.value)} spellCheck={false} aria-busy={busy} aria-describedby={hintId} placeholder={busy ? 'Preparing your factual report…' : 'Your report will appear here. Try regenerating if it is unavailable.'} />
        <p id={hintId}>Nutrition follows your saved Tracking Days; health schedule boundaries are listed in the report. Missing data is not counted as zero.</p>
      </div>
    </div>
    <div className="text-summary-footer"><p>For your privacy, share only with people you trust. This report may include sensitive health information.</p><p role="status" aria-live="polite">{status || (dirty ? 'Unsaved edits · copy before leaving' : source ? <time dateTime={source.generated_at}>{generatedLabel(source.generated_at)}</time> : busy ? 'Preparing your report…' : '')}</p></div>
  </section>
}
export function TextSummary({ from, to, mode, nutrient = 'ALL' }: { from: string; to: string; mode: 'DAY' | 'RANGE'; nutrient?: string }) {
  const dialogs = useDialogs()
  const [source, setSource] = useState<Summary | null>(null), [draft, setDraft] = useState('')
  const [busy, setBusy] = useState(false), [error, setError] = useState(''), [status, setStatus] = useState('')
  const [expanded, setExpanded] = useState(false), [selectForCopy, setSelectForCopy] = useState(false)
  const area = useRef<HTMLTextAreaElement>(null), request = useRef(0), currentContext = `${mode}|${from}|${to}|${nutrient}`
  const dirty = source !== null && draft !== source.text
  useEffect(() => {
    if (selectForCopy && expanded) { area.current?.focus(); area.current?.select(); setSelectForCopy(false) }
  }, [selectForCopy, expanded])
  async function generate() {
    const id = ++request.current
    setBusy(true); setError(''); setStatus('')
    try {
      const next = await getAuthenticatedJson<Summary>(`/text-summary?${new URLSearchParams({ from, to, mode, nutrient })}`)
      if (id !== request.current) return
      setSource(next); setDraft(next.text)
    } catch (e) { if (id === request.current) setError(e instanceof Error ? e.message : 'Summary unavailable. Existing text has been kept.') }
    finally { if (id === request.current) setBusy(false) }
  }
  useEffect(() => {
    // A context change must not destroy a user's edited report.
    request.current++; setBusy(false)
    if (from && to && !dirty) void generate()
    return () => { request.current++ }
  }, [currentContext])
  useEffect(() => {
    const expire = () => { request.current++; setDraft(''); setSource(null) }
    window.addEventListener('atethat:session-expired', expire)
    return () => window.removeEventListener('atethat:session-expired', expire)
  }, [])
  useDraftGuard(dirty, 'Leave this page and discard your edited Text Summary? Copy it first if you want to keep it.')
  async function regenerate() { if (dirty && !await dialogs.confirm('Replace your edited Text Summary with a fresh factual summary? Copy your changes first to keep them.')) return; void generate() }
  async function copy() { setStatus(await copySummaryText(draft, text => navigator.clipboard.writeText(text), () => { setExpanded(true); setSelectForCopy(true) })) }
  return <TextSummaryCard from={from} to={to} mode={mode} nutrient={nutrient} source={source} draft={draft} busy={busy} error={error} status={status} expanded={expanded} dirty={dirty} stale={!!source && source.context !== currentContext} editorRef={area} onToggle={() => setExpanded(value => !value)} onChange={value => { setDraft(value); setStatus('') }} onCopy={() => void copy()} onRegenerate={() => void regenerate()} />
}
