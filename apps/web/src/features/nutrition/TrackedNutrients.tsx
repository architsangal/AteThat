import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { getAuthenticatedJson, patchAuthenticatedJson } from '../../api'
import { label, nutrientFields, nutrientGroup } from './model'
import { CheckOption } from '../CheckOption'

export const defaultNutrients = ['ENERGY', 'PROTEIN', 'CARBOHYDRATE', 'FAT', 'FIBER']
export function normalizeTracked(values?: string[] | null) {
  return nutrientFields.map(([code]) => code).filter(code => code === 'ENERGY' || (values?.length ? values : defaultNutrients).includes(code))
}
type Selection = { codes: string[]; loading: boolean; error: string; save: (codes: string[]) => Promise<void> }
const Context = createContext<Selection>({ codes: defaultNutrients, loading: false, error: '', save: async () => {} })
export const useTrackedNutrients = () => useContext(Context)

export function TrackedNutrientProvider({ children }: { children: ReactNode }) {
  const [codes, setCodes] = useState(defaultNutrients), [loading, setLoading] = useState(true), [error, setError] = useState('')
  useEffect(() => {
    let active = true
    getAuthenticatedJson<{dashboard_nutrients: string[]}>('/preferences')
      .then(p => { if (active) setCodes(normalizeTracked(p.dashboard_nutrients)) })
      .catch(() => { if (active) setError('Your nutrient selection could not be loaded. Default nutrients are shown.') })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [])
  async function save(next: string[]) {
    const current = await getAuthenticatedJson<{version: number}>('/preferences')
    const saved = await patchAuthenticatedJson<{dashboard_nutrients: string[]}>('/preferences', {version: current.version, dashboard_nutrients: normalizeTracked(next)})
    setCodes(normalizeTracked(saved.dashboard_nutrients)); setError('')
    window.dispatchEvent(new Event('atethat:preferences-updated'))
  }
  return <Context.Provider value={{codes, loading, error, save}}>{children}</Context.Provider>
}

export function TrackedNutrientsSettings() {
  const selection = useTrackedNutrients()
  const [draft, setDraft] = useState(selection.codes), [busy, setBusy] = useState(false), [error, setError] = useState(''), [message, setMessage] = useState('')
  useEffect(() => { setDraft(selection.codes) }, [selection.codes])
  const changed = normalizeTracked(draft).join(',') !== selection.codes.join(',')
  function choose(next:string[]) {setDraft(next);setMessage('');setError('')}
  return <section className="panel section-panel nutrient-preferences" aria-labelledby="tracked-nutrients-title">
    <div className="nutrient-preferences-header"><h2 id="tracked-nutrients-title">Tracked nutrients</h2><span className="nutrient-preferences-count" role="status" aria-live="polite">{selection.loading?'Loading selection…':`${draft.length} selected${changed?' · Unsaved changes':''}`}</span></div>
    <p className="muted">Choose which nutrients you see when logging food and viewing your progress. You can still open All nutrition details to see the full record.</p>
    <form className="auth-form" onSubmit={async event => { event.preventDefault(); setBusy(true); setError(''); setMessage(''); try { await selection.save(draft); setMessage('Tracked nutrients saved across AteThat.') } catch(cause) {setError(cause instanceof Error ? cause.message : 'Could not save tracked nutrients.')} finally {setBusy(false)} }}>
      <fieldset disabled={busy || selection.loading} className="tracked-selection">
        <legend className="sr-only">Nutrients to track</legend>
        <div className="filter-row"><button className="button button-outline button-small" type="button" onClick={() => choose(nutrientFields.map(([code]) => code))}>Select all</button><button className="button button-outline button-small" type="button" onClick={() => choose(['ENERGY'])}>Energy only</button><button className="quiet-link" type="button" onClick={() => choose(defaultNutrients)}>Use defaults</button></div>
        {['Energy & other nutrients','Fats','Vitamins','Minerals','Amino acids'].map(group => {
          const fields = nutrientFields.filter(([code]) => nutrientGroup(code) === group)
          const selected = fields.filter(([code]) => draft.includes(code)).length
          return <details className="nutrient-section" key={group} open={group === 'Energy & other nutrients' || group === 'Fats'}><summary><span>{group === 'Energy & other nutrients' ? 'Energy & nutrition basics' : group === 'Fats' ? 'Fats & cholesterol' : group}</span><span className="nutrient-group-count">{selected} of {fields.length}</span></summary><div className="tracked-grid">{fields.map(([code]) => <CheckOption key={code} label={label(code)} badge={code === 'ENERGY' ? 'Always on' : undefined} checked={draft.includes(code)} disabled={code === 'ENERGY'} onChange={event => choose(event.target.checked ? [...draft, code] : draft.filter(value => value !== code))} />)}</div></details>
        })}
      </fieldset>
      <div className="nutrient-selection-footer"><small>Defaults: energy, protein, carbohydrates, fat and fibre. Energy stays on for your food-log totals.</small><button className="button button-primary" disabled={busy || selection.loading || (!changed && !selection.error)}>{busy ? 'Saving…' : 'Save tracked nutrients'}</button></div>
      {(error || selection.error) && <p role="alert" className="form-message form-error">{error || selection.error}</p>}{message && <p role="status" className="form-message form-success">{message}</p>}
    </form>
  </section>
}
