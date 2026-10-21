import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAuthenticatedJson, postAuthenticatedJson } from '../api'
import { Form, Modal, useDialogs } from './ui'
import { calorieDifference, exceedsCalorieTolerance } from './target-math'
import { useDraftGuard } from './useDraftGuard'
import { missingPlanningDetails, focusProfileField } from './profile-completion'

type ReferenceRange = {minimum:string; maximum:string; energy_basis:string; source_url:string; reference_revision:string}
type RangeWarning = {nutrient:string; selected_value:string; unit:string; reference_range:ReferenceRange}
type SaveBody = {expected_active_id:string; expected_revision:number; profile_version:number; changes:Record<string,{active_value:string;override_reason:string}>; energy_provenance:string; acknowledgement_token?:string}
type CustomReview = {body:SaveBody; warnings:RangeWarning[]; acknowledgement_token:string}
type Inputs = { height_cm: string; weight_kg: string; activity: string; assumed_height?: boolean; assumed_weight?: boolean; assumed_activity?: boolean; age?: number; sex?: string; profile_saved_at?: string }
type Plan = { inputs: Inputs; profile_version: number; policy_version: string; personalized_enabled: boolean; pending_decisions: string[]; missing_inputs: string[]; macro_calories: string; rounding_tolerance_kcal: string; explanation: string; calculated_defaults?: Record<string,string>; energy_equation?: string; protein_rate?: string; activity_category?: string; reset_preview_token?: string; calculation_unavailable_reason?: string; calculated_warnings?:RangeWarning[] }
type Version = { status?:string; id: string; revision: number; profile_version: number; effective_from_tracking_day: string; input_snapshot?: Inputs; targets: Record<string, { active_value: string; unit: string; provenance?: string; is_manual_override?: boolean; reference_value?: string; reference_range?:ReferenceRange; outside_reference?:boolean; acknowledged_at?:string }> }
const fields = [['ENERGY', 'Daily calorie target', 'kcal'], ['PROTEIN', 'Protein', 'g'], ['CARBOHYDRATE', 'Carbohydrate', 'g'], ['FAT', 'Fat', 'g'], ['FIBER', 'Fibre', 'g']]
const message = (e: unknown) => e instanceof Error ? e.message : 'Could not complete this request.'

export function TargetSettings() {
  const dialogs = useDialogs()
  const [customReview,setCustomReview] = useState<CustomReview|null>(null)
  const [preview, setPreview] = useState<Plan | null>(null)
  const [scopeConfirmed, setScopeConfirmed] = useState(false)
  const [resetError, setResetError] = useState('')
  const [plan, setPlan] = useState<Plan | null>(null)
  const [active, setActive] = useState<Version | null>(null)
  const [draft, setDraft] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [derived, setDerived] = useState(false)
  const [notice, setNotice] = useState(false)
  const [macroEnergy, setMacroEnergy] = useState('')
  async function load() {
    try {
      const [p, selection] = await Promise.all([getAuthenticatedJson<Plan>('/nutrition/targets/planning'), getAuthenticatedJson<{ active: Version | null }>('/nutrition/targets')])
      setPlan(p); setActive(selection.active); setDraft(Object.fromEntries(fields.map(([key]) => [key, selection.active?.targets[key]?.active_value || '']))); setDerived(false); setNotice(false)
    } catch (e) { setError(message(e)) }
  }
  useEffect(() => { void load(); const listener = () => { getAuthenticatedJson<Plan>('/nutrition/targets/planning').then(setPlan).catch(e => setError(message(e))) }; window.addEventListener('atethat:profile-updated', listener); return () => window.removeEventListener('atethat:profile-updated', listener) }, [])
  useDraftGuard(fields.some(([key]) => (draft[key] || '') !== (active?.targets[key]?.active_value || '')), 'Leave this page and discard unsaved nutrition target edits?')
  useEffect(() => {
    if (!notice) return
    let current = true
    postAuthenticatedJson<Plan>('/nutrition/targets/planning', { macros: { PROTEIN: draft.PROTEIN, CARBOHYDRATE: draft.CARBOHYDRATE, FAT: draft.FAT } }).then(p => { if (current) setMacroEnergy(p.macro_calories) }).catch(() => { if (current) setMacroEnergy('') })
    return () => { current = false }
  }, [draft.PROTEIN, draft.CARBOHYDRATE, draft.FAT, notice])
  async function refresh() {
    setBusy(true); setError('')
    try {
      const p = await postAuthenticatedJson<Plan>('/nutrition/targets/planning', { macros: { PROTEIN: draft.PROTEIN, CARBOHYDRATE: draft.CARBOHYDRATE, FAT: draft.FAT } })
      setDraft(d => ({ ...d, ENERGY: p.macro_calories })); setMacroEnergy(p.macro_calories); setDerived(true); setNotice(false)
    } catch (e) { setError(message(e)) } finally { setBusy(false) }
  }
  async function persist(body:SaveBody) {
    setBusy(true); setError(''); setSuccess('')
    try {
      const result = await postAuthenticatedJson<Version>('/nutrition/targets/custom',body)
      setActive(result);setDraft(Object.fromEntries(fields.map(([key])=>[key,result.targets[key].active_value])));setCustomReview(null)
      setSuccess('Targets saved. Previous revisions remain in history.')
    } catch(e) {setError(message(e))} finally {setBusy(false)}
  }
  async function save() {
    if (!plan) return
    const changes = Object.fromEntries(fields.filter(([key]) => !active || draft[key] !== active.targets[key]?.active_value).map(([key]) => [key, { active_value: draft[key], override_reason: '' }]))
    if (!Object.keys(changes).length) { setSuccess('No target values changed.'); return }
    const body:SaveBody={expected_active_id:active?.id || '',expected_revision:active?.revision || 0,profile_version:plan.profile_version,changes,energy_provenance:changes.ENERGY && derived ? 'MACRO_DERIVED' : 'CUSTOM'}
    setBusy(true);setError('');setSuccess('')
    try {
      const review=await postAuthenticatedJson<{warnings:RangeWarning[]|null;acknowledgement_token:string}>('/nutrition/targets/custom/review',body)
      if(review.warnings?.length) {setCustomReview({body,warnings:review.warnings,acknowledgement_token:review.acknowledgement_token});return}
      if(await dialogs.confirm('Save these selected targets? Only edited values change. Existing food logs, water goals and profile measurements remain unchanged.')) await persist(body)
    } catch(e) {setError(message(e))} finally {setBusy(false)}
  }
  async function previewReset() {
    setBusy(true); setError(''); setResetError(''); setScopeConfirmed(false)
    try {
      const fresh = await getAuthenticatedJson<Plan>('/nutrition/targets/planning')
      setPlan(fresh)
      if (!fresh.personalized_enabled) { setError(fresh.calculation_unavailable_reason || 'Review your profile before calculating defaults.'); return }
      setPreview(fresh)
    } catch (e) { setError(message(e)) } finally { setBusy(false) }
  }
  async function confirmReset() {
    if (!preview || !scopeConfirmed) return
    setBusy(true); setResetError('')
    try {
      const result = await postAuthenticatedJson<Version>('/nutrition/targets/reset', { expected_active_id:active?.id || '', expected_revision:active?.revision || 0, profile_version:preview.profile_version, preview_token:preview.reset_preview_token, adult_scope_confirmed:scopeConfirmed })
      setActive(result); setDraft(Object.fromEntries(fields.map(([key]) => [key,result.targets[key].active_value]))); setNotice(false); setDerived(false); setPreview(null); setError(''); setSuccess('Calculated defaults saved. Your previous targets remain in revision history.')
    } catch (e) { setResetError(message(e)) } finally { setBusy(false) }
  }
  const missing = plan ? missingPlanningDetails(plan) : []
  return <section className="panel" id="nutrition-targets"><h2>Nutrition targets</h2><p>Choose independent tracking values. Changing a macro never changes your calories automatically.</p><Link className="quiet-link" to="/app/target-calculations">How your targets are calculated →</Link>
    {plan && <><p className="muted">Your profile details: {plan.inputs.height_cm} cm{plan.inputs.assumed_height ? ' (assumed)' : ''}, {plan.inputs.weight_kg} kg{plan.inputs.assumed_weight ? ' (assumed)' : ''}, {activityNames[plan.inputs.activity] || plan.inputs.activity}{plan.inputs.assumed_activity ? ' (assumed)' : ''}. Weight logs do not change these inputs.</p>{missing.length > 0 && <div className="form-message target-notice">Optional details not provided: {missing.map((field,index) => <span key={field.key}>{index > 0 && ', '}<button type="button" className="quiet-link" onClick={() => focusProfileField(field.fieldId)}>Review {field.label.toLowerCase()}</button></span>)}. Each link opens and highlights its field. Logging remains available.</div>}{active && active.profile_version !== plan.profile_version && <p role="status" className="form-message target-notice">Profile changed since revision {active.revision}. Review your targets; no accepted value was automatically replaced.</p>}</>}
    {!active && plan?.personalized_enabled && <AutomaticTargetProposal onAccepted={result=>{setActive(result);setDraft(Object.fromEntries(fields.map(([key])=>[key,result.targets[key].active_value])));setSuccess('Your calculated starting targets are now active.')}} />}
    <Form className="auth-form" onSubmit={e => { e.preventDefault(); void save() }}>
      {!active && <p className="muted">No accepted set yet. Choose calculated defaults below, or enter all five values manually.</p>}
      {fields.map(([key, label, unit]) => <div className="target-field" key={key}><label htmlFor={`target-${key}`}>{label} ({unit})</label><input id={`target-${key}`} aria-describedby={`target-${key}-source`} disabled={busy} required inputMode="decimal" pattern="(0|[1-9][0-9]*)(\.[0-9]+)?" maxLength={32} value={draft[key] || ''} onChange={e => { setDraft(d => ({ ...d, [key]: e.target.value })); setSuccess(''); if (['PROTEIN','CARBOHYDRATE','FAT'].includes(key)) { setNotice(true); setDerived(false) } if (key === 'ENERGY') setDerived(false) }} /><small id={`target-${key}-source`}>{active ? targetSourceLabel(active.targets[key]) : 'Enter a selected value'}{key === 'ENERGY' && derived ? ' · unsaved macro-derived value' : ''}</small>{active?.targets[key].reference_range && <small>{referenceLabel(active.targets[key].reference_range!, unit)}. Guidance, not a safety limit.</small>}{key === 'ENERGY' && <button type="button" className="button button-outline" disabled={busy || !draft.PROTEIN || !draft.CARBOHYDRATE || !draft.FAT} onClick={() => void refresh()}>Calculate calories from selected macros</button>}</div>)}
      {notice && exceedsCalorieTolerance(macroEnergy, draft.ENERGY || '') && <div className="form-message target-notice" role="status">Selected calories: {draft.ENERGY} kcal. Macro-derived calories: {macroEnergy} kcal. Difference: {calorieDifference(macroEnergy, draft.ENERGY)} kcal. These fields are independent. <button type="button" className="quiet-link" onClick={() => setNotice(false)} aria-label="Dismiss calorie mismatch notice">Dismiss ×</button></div>}
      <button type="submit" className="button button-primary" disabled={busy || !plan}>{busy ? 'Working…' : 'Save selected targets'}</button>
      <button type="button" className="button button-outline" disabled={busy || !plan?.personalized_enabled} onClick={() => void previewReset()}>Reset to calculated defaults</button><small>{plan?.personalized_enabled ? 'Review a calculated set before replacing all five targets. Water, profile measurements and food logs stay unchanged.' : plan?.calculation_unavailable_reason || 'Loading calculated defaults…'}</small>
    </Form><Link className="quiet-link" to="/app/targets">View target revision history</Link>{error && <p className="form-message form-error" role="alert">{error}</p>}{success && <p className="form-message form-success" role="status">{success}</p>}
    {customReview && <Modal title="Review outside-reference targets" initialFocus="heading" busy={busy} onClose={()=>setCustomReview(null)}>
      <div className="dialog-content"><p>These targets are outside the adult reference ranges shown. Save these custom targets? This confirmation is not medical approval. Only edited fields change; a new calorie target can change the comparison ranges without changing your macros.</p><ReferenceWarnings warnings={customReview.warnings}/>{error&&<p className="form-message form-error" role="alert">{error} Cancel and review again if your profile or saved targets changed.</p>}</div>
      <div className="dialog-actions"><button type="button" className="button button-outline" disabled={busy} onClick={()=>setCustomReview(null)}>Cancel</button><button type="button" className="button button-primary" disabled={busy} onClick={()=>void persist({...customReview.body,acknowledgement_token:customReview.acknowledgement_token})}>{busy?'Saving…':'Save custom targets'}</button></div>
    </Modal>}
    {preview && <Modal title="Review calculated defaults" initialFocus="heading" busy={busy} onClose={() => setPreview(null)}>
      <div className="dialog-content"><p>All five saved nutrition targets will be replaced only when you confirm. Cancelling keeps your saved targets and unsaved edits.</p>
      <ResetComparison active={active} preview={preview} /><ReferenceWarnings warnings={preview.calculated_warnings || []}/>
      {fields.some(([key]) => (draft[key] || '') !== (active?.targets[key]?.active_value || '')) && <p className="form-message target-notice">You have unsaved edits. Confirming reset replaces those drafts as well.</p>}
      <p>Using {preview.inputs.height_cm} cm{preview.inputs.assumed_height ? ' (assumed)' : ' (profile)'}, {preview.inputs.weight_kg} kg{preview.inputs.assumed_weight ? ' (assumed)' : ' (profile)'}, {activityNames[preview.inputs.activity]}{preview.inputs.assumed_activity ? ' (assumed)' : ''}. Protein rate: {preview.protein_rate} g/kg. <Link to="/app/target-calculations">See formulas and assumptions</Link>.</p>
      <label className="checkbox-label"><input type="checkbox" checked={scopeConfirmed} onChange={e => setScopeConfirmed(e.target.checked)} />I have reviewed the targets and any reference differences shown. I am 19 or older, not pregnant or breastfeeding, and do not need a prescribed diet or fluid restriction. I understand these are approximate tracking defaults, not medical advice.</label>
      {resetError && <p role="alert" className="form-message form-error">{resetError} Cancel and review a fresh preview if your profile or targets changed.</p>}
      </div><div className="dialog-actions"><button type="button" className="button button-outline" disabled={busy} onClick={() => setPreview(null)}>Cancel</button><button type="button" className="button button-primary" disabled={busy || !scopeConfirmed} onClick={() => void confirmReset()}>{busy ? 'Saving…' : 'Confirm calculated defaults'}</button></div>
    </Modal>}
  </section>
}


const activityNames: Record<string, string> = { SEDENTARY: 'Sedentary', LIGHT: 'Light', MODERATE: 'Moderate', ACTIVE: 'Active', VERY_ACTIVE: 'Very active' }
const targetSourceLabel = (value?: Version['targets'][string]) => value?.outside_reference ? 'Custom / selected target · outside reference'+(value.acknowledged_at ? ' · confirmed' : ' · review on next save') : value?.provenance === 'CALCULATED_DEFAULT' ? 'Calculated default' : value?.provenance === 'MACRO_DERIVED' ? 'Calculated from selected macros' : value?.provenance === 'CUSTOM' || value?.is_manual_override ? 'Custom target' : 'General benchmark'

const displayCalculated = (value: string | undefined) => value ? Number(value).toLocaleString(undefined,{maximumFractionDigits:2}) : 'Not set'
export function ResetComparison({active,preview}: {active:Version | null; preview:Plan}) {
 return <div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><caption>Current and calculated targets</caption><thead><tr><th>Target</th><th>Current</th><th>Calculated</th></tr></thead><tbody>{fields.map(([key,label,unit]) => <tr key={key}><th>{label}</th><td data-label="Current">{active?.targets[key] ? displayCalculated(active.targets[key].active_value)+' '+unit : 'Not set'}</td><td data-label="Calculated">{displayCalculated(preview.calculated_defaults?.[key])} {unit}</td></tr>)}</tbody></table></div>
}

const referenceLabel=(range:ReferenceRange,unit:string)=>'Adult reference: approximately '+displayCalculated(range.minimum)+'–'+displayCalculated(range.maximum)+' '+unit+' at '+displayCalculated(range.energy_basis)+' kcal'
export function ReferenceWarnings({warnings}:{warnings:RangeWarning[]}) {
 if(!warnings.length)return null
 return <div className="target-reference-warnings">{warnings.map(warning=><div className="form-message target-notice" key={warning.nutrient}><strong>{fields.find(([key])=>key===warning.nutrient)?.[1] || warning.nutrient}: {warning.selected_value} {warning.unit}</strong><p>{referenceLabel(warning.reference_range,warning.unit)}. This is guidance, not a clinical limit.</p><a className="quiet-link" href={warning.reference_range.source_url} target="_blank" rel="noreferrer">View reference</a></div>)}</div>
}

export function AutomaticTargetProposal({onAccepted}:{onAccepted?:(value:Version)=>void}) {
 const [proposal,setProposal]=useState<Version|null>(null),[plan,setPlan]=useState<Plan|null>(null)
 const [busy,setBusy]=useState(false),[scope,setScope]=useState(false),[error,setError]=useState(''),[success,setSuccess]=useState('')
 const request=useRef(0)
 useEffect(()=>{
  const load=async()=>{
   const id=++request.current;setError('');setScope(false)
   try{
    const [p,current]=await Promise.all([getAuthenticatedJson<Plan>('/nutrition/targets/planning'),getAuthenticatedJson<{active:Version|null}>('/nutrition/targets')])
    if(id!==request.current)return
    if(current.active||!p.personalized_enabled){setProposal(null);return}
    const next=await postAuthenticatedJson<Version>('/nutrition/targets/proposal',{})
    if(id!==request.current)return
    if(next.profile_version!==p.profile_version)throw new Error('Profile changed. Reload to review your latest starting targets.')
    setPlan({...p,inputs:next.input_snapshot || p.inputs,calculated_defaults:Object.fromEntries(Object.entries(next.targets).map(([key,value])=>[key,value.active_value]))})
    setProposal(next.status==='PROPOSED'?next:null)
   }catch(e){if(id===request.current)setError(message(e))}
  }
  void load();window.addEventListener('atethat:profile-updated',load)
  return()=>{request.current++;window.removeEventListener('atethat:profile-updated',load)}
 },[])
 async function decide(accept:boolean) {
  if(!proposal || (accept&&!scope))return
  setBusy(true);setError('')
  try{
   if(accept) {
    const day=await getAuthenticatedJson<{tracking_day:string}>('/tracking-day/current')
    const saved=await postAuthenticatedJson<Version>('/nutrition/targets/'+proposal.id+'/accept',{effective_from_tracking_day:day.tracking_day,overrides:{},adult_scope_confirmed:scope})
    onAccepted?.(saved);setSuccess('Your starting targets are active. You can edit each value independently in Settings.')
   }else{await postAuthenticatedJson('/nutrition/targets/'+proposal.id+'/reject',{});setSuccess('Starting targets skipped. You can set manual targets or use Reset in Settings later.')}
   setProposal(null)
  }catch(e){setError(message(e))}finally{setBusy(false)}
 }
 if(!proposal||!plan)return <>{error&&<p className="form-message form-error" role="alert">{error}</p>}{success&&<p className="form-message form-success" role="status">{success}</p>}</>
 return <section className="target-initial-proposal" aria-label="Suggested starting targets"><h3>Your suggested starting targets</h3><p>Calculated automatically from your profile, using labelled defaults where information is missing. Nothing is active until you accept.</p><ResetComparison active={null} preview={plan}/><p className="muted">Inputs: {plan.inputs.height_cm} cm{plan.inputs.assumed_height?' (assumed)':''}, {plan.inputs.weight_kg} kg{plan.inputs.assumed_weight?' (assumed)':''}, {activityNames[plan.inputs.activity]}{plan.inputs.assumed_activity?' (assumed)':''}. Logged weight never changes profile weight.</p><ReferenceWarnings warnings={plan.calculated_warnings || []}/><label className="target-scope-choice"><input type="checkbox" checked={scope} onChange={e=>setScope(e.target.checked)}/>I have reviewed these values and any reference differences. I am 19 or older, not pregnant or breastfeeding, and do not need a prescribed diet or fluid restriction. These are approximate tracking defaults, not medical advice.</label><div className="secondary-actions"><button type="button" className="button button-primary" disabled={busy||!scope} onClick={()=>void decide(true)}>Accept calculated targets</button><button type="button" className="button button-outline" disabled={busy} onClick={()=>void decide(false)}>Not now</button><Link to="/app/target-calculations" className="quiet-link">See formulas and assumptions</Link></div>{error&&<p role="alert" className="form-message form-error">{error}</p>}</section>
}

// User-facing guide describes only calculations the app actually performs.
// Internal policy/version metadata stays in the API and engineering documents.
export function TargetCalculationGuide({ plan, active, macroEnergy, macroError = '' }: { plan: Plan; active: Version | null; macroEnergy: string; macroError?: string }) {
  const inputs = plan.inputs
  const targets = active?.targets
  return <div className="target-guide">
    <header><h1>How your nutrition targets are calculated</h1><p>Understand the numbers you see, the information behind them, and what happens when you make a change.</p></header>
    <section className="panel"><h2>Calories from protein, carbohydrate and fat</h2>
      <p>When you choose <strong>Calculate calories from selected macros</strong>, AteThat uses this formula. Amounts are in grams; the result is in kilocalories (kcal).</p>
      <div className="formula-block"><p className="formula">Calories = (protein × 4) + (carbohydrate × 4) + (fat × 9)</p><p>Each gram of protein or carbohydrate contributes 4 kcal in this calculation; each gram of fat contributes 9 kcal.</p></div>
      {targets && macroEnergy && <div className="formula-block"><h3>Using your saved macros</h3><p className="formula">({targets.PROTEIN.active_value} × 4) + ({targets.CARBOHYDRATE.active_value} × 4) + ({targets.FAT.active_value} × 9) = {macroEnergy} kcal</p><p>Your saved calorie target is <strong>{targets.ENERGY.active_value} kcal</strong>. Changing a macro does not automatically replace that target. The refresh button uses the values currently in your Settings form, including unsaved edits.</p>{exceedsCalorieTolerance(macroEnergy, targets.ENERGY.active_value) && <p>The difference is {calorieDifference(macroEnergy, targets.ENERGY.active_value)} kcal. You can keep independently chosen values or deliberately refresh calories from your macros.</p>}</div>}
      {macroError && <p className="form-message target-notice" role="status">Your saved macro calculation could not be loaded. The formula and example below are still available.</p>}
      <details className="secondary-details"><summary>See a worked example</summary><p>For example, 160 g protein, 400 g carbohydrate and 100 g fat give:</p><p className="formula">(160 × 4) + (400 × 4) + (100 × 9) = 640 + 1,600 + 900 = 3,140 kcal</p><p>This is an arithmetic example, not a suggested diet.</p></details>
      <p>This formula is for planning targets only. It does not recalculate calories in your food history. Food labels and provider values can differ because of fibre, alcohol, food-specific energy factors and rounding.</p>
    </section>
    <section className="panel"><h2>Calculated defaults from your profile</h2>
      <p>For adults aged 19 or older, reset estimates maintenance calories from age, equation-specific sex, height, profile weight and activity. Missing height, weight or activity uses the fallbacks below. Missing age or sex is never guessed. These estimates are not for pregnancy, breastfeeding or prescribed diets.</p>
      {plan.energy_equation ? <div className="formula-block"><p>Energy equation category: {plan.activity_category}. Age: {inputs.age} years; equation: {inputs.sex === 'MALE' ? 'male' : 'female'}; height: {inputs.height_cm} cm; weight: {inputs.weight_kg} kg.</p><p className="formula">Daily calories = {plan.energy_equation}</p><p>Result: {displayCalculated(plan.calculated_defaults?.ENERGY)} kcal/day. Activity is already included; no extra activity multiplier is applied.</p></div> : <p className="form-message target-notice">{plan.calculation_unavailable_reason || 'Add date of birth and equation-specific sex in Settings to see your energy calculation.'}</p>}
      <h3>Reference ranges for custom targets</h3><p>For adults aged 19+, protein uses 10–35%, carbohydrate 45–65% and fat 20–35% of the selected calorie target. The comparison is calculated as calories × the percentage ÷ 4 for protein/carbohydrate, or ÷ 9 for fat. Saving outside these ranges requires review of the exact values and calorie basis. Changing calories can require a new confirmation without changing your macros. Saved outside-reference values remain labelled when you return. Calories and fibre are not assigned invented upper limits; missing/minor age has no adult range applied.</p><h3>Activity and protein rates</h3><p>These are AteThat’s approximate mappings, not measured activity levels or individual prescriptions. Moderate is used when no activity level is supplied. Moderate and Active use the same energy branch but different protein rates.</p>
      <div className="nutrition-table-wrap"><table className="nutrition-table"><caption>Activity mapping used for calculated defaults</caption><thead><tr><th>Your activity</th><th>Energy category</th><th>Protein per kg</th></tr></thead><tbody>{[['Sedentary','Inactive','0.8'],['Light','Low active','0.8'],['Moderate (default)','Active','1.2'],['Active','Active','1.6'],['Very active','Very active','2.0']].map(([activity,category,rate]) => <tr key={activity}><th>{activity}</th><td>{category}</td><td>{rate} g/kg</td></tr>)}</tbody></table></div>
      <p>Protein = profile weight (kg) × activity protein rate{plan.protein_rate ? ' ('+plan.protein_rate+' g/kg for your activity)' : ''}.</p>
      <p>Fat (g) = daily calories × 0.275 ÷ 9. This chooses 27.5% of calories, the midpoint of the adult 20–35% reference range.</p>
      <p>Carbohydrate (g) = [daily calories − (protein × 4) − (fat × 9)] ÷ 4. If this produces a carbohydrate share outside 45–65%, reset asks you to review the inputs instead of silently changing the formula.</p>
      <p>Fibre (g) = daily calories × 14 ÷ 1,000.</p>
      {plan.calculated_defaults && <ResetComparison active={active} preview={plan} />}
      <p>Reset shows a preview and requires your confirmation before saving a new revision. It never changes profile measurements, water goals or food records. Ordinary edits remain independent.</p>
      <p>Sources: <a href="https://www.nationalacademies.org/read/26818/chapter/7" target="_blank" rel="noreferrer">National Academies adult energy equations, Table 5-16</a>; <a href="https://medlineplus.gov/ency/article/002458.htm" target="_blank" rel="noreferrer">MedlinePlus sports nutrition</a>; <a href="https://www.canada.ca/en/health-canada/services/food-nutrition/healthy-eating/dietary-reference-intakes/tables/reference-values-macronutrients.html" target="_blank" rel="noreferrer">Health Canada macronutrient reference tables</a>; <a href="https://www.nationalacademies.org/read/11767/chapter/9" target="_blank" rel="noreferrer">National Academies fibre-density reference</a>. The activity-to-protein mapping is an app approximation: being moderately active does not by itself establish athletic protein requirements.</p>
    </section>
    <section className="panel"><h2>Where your other target values come from</h2>
      <p>Targets can be calculated defaults, custom values or general benchmarks. The label beside each saved value tells you its origin. If you use the general benchmarks, they come from the FDA’s food-label reference values for a 2,000-kcal diet. They are not a personalized estimate of your daily needs, a minimum/maximum range, or an upper limit.</p>
      {targets ? <div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><caption>Your saved targets and their reference values</caption><thead><tr><th>Target</th><th>Your saved value</th><th>Reference at selection</th></tr></thead><tbody>{fields.map(([key, label, unit]) => targets[key] && <tr key={key}><th>{label}</th><td data-label="Your saved value">{targets[key].active_value} {unit}<small className="muted"> · {targetSourceLabel(targets[key])}</small></td><td data-label="Reference at selection">{targets[key].reference_value ? targets[key].reference_value + ' ' + unit : 'Not available'}</td></tr>)}</tbody></table></div> : <p>You have not saved any nutrition targets yet. Choose your tracking values in Settings.</p>}
      <p>Editing one field changes only that field. Your other targets, water goal, profile measurements and existing food records stay unchanged.</p>
      <p>Calculated defaults are estimates, not a guaranteed energy requirement. General FDA benchmarks are not personalized. A reference alongside a calculated target is its original calculated value, not an FDA value.</p>
      <p><a className="quiet-link" href="https://www.fda.gov/media/99059/download" target="_blank" rel="noreferrer">Read the FDA’s Daily Values reference</a></p>
    </section>
    <section className="panel"><h2>Your profile details and fallback values</h2>
      <p>These are the profile details available to AteThat. A fallback is used only when a detail has not been provided. Reset uses these effective inputs; supplied profile values take priority over fallbacks.</p>
      <dl className="input-facts">
        <div><dt>Height</dt><dd>{inputs.height_cm} cm<small>{inputs.assumed_height ? 'Default used — not a measurement' : 'From your saved profile'}</small></dd></div>
        <div><dt>Weight</dt><dd>{inputs.weight_kg} kg<small>{inputs.assumed_weight ? 'Default used — not a measurement' : 'From your saved profile'}</small></dd></div>
        <div><dt>Activity level</dt><dd>{activityNames[inputs.activity] || inputs.activity}<small>{inputs.assumed_activity ? 'Default used' : 'From your saved profile'}</small></dd></div>
      </dl>
      <p>When missing, the fixed fallbacks are <strong>175 cm</strong>, <strong>70.4375 kg</strong> and <strong>Moderate activity</strong>. Your supplied details always take priority. Missing age or sex is not guessed.</p>
      <details className="secondary-details"><summary>Why is the fallback weight 70.4375 kg?</summary><p>The fixed fallback was chosen once using a reference BMI of 23 and a height of 1.75 metres:</p><p className="formula">Weight (kg) = reference BMI × height (m)²</p><p className="formula">23 × 1.75 × 1.75 = 70.4375 kg</p><p>This is a fixed fallback, not a measurement, BMI assessment or recommended weight. It is not recalculated when you change your height.</p></details>
      <p>Logging a weight measurement never changes the weight in your profile. If you update your profile in Settings, AteThat invites you to review your targets; it does not silently replace them.</p>
    </section>
    <section className="panel"><h2>Rounding, notices and water</h2><p>Saved decimal values are preserved. A calorie mismatch notice appears after you edit macros when their calculated calories differ from your selected calorie target by more than {plan.rounding_tolerance_kcal} kcal. You can dismiss it; reloading clears the notice, not your saved targets.</p><p>Your water target is a separate drinking-water goal that you choose. It is not calculated from body weight, and it does not represent all water from food and drinks.</p><p>These tools help you track your own choices. They are not personalized medical advice.</p><Link className="button button-outline" to="/app/settings#nutrition-targets">Back to nutrition targets</Link></section>
  </div>
}

export function TargetCalculations() {
  const [plan, setPlan] = useState<Plan | null>(null)
  const [active, setActive] = useState<Version | null>(null)
  const [macroEnergy, setMacroEnergy] = useState('')
  const [macroError, setMacroError] = useState('')
  const [error, setError] = useState('')
  useEffect(() => {
    let alive = true
    async function load() {
      try {
        const [p, selection] = await Promise.all([getAuthenticatedJson<Plan>('/nutrition/targets/planning'), getAuthenticatedJson<{ active: Version | null }>('/nutrition/targets')])
        if (!alive) return
        setPlan(p); setActive(selection.active)
        if (selection.active) {
          const values = selection.active.targets
          try {
            const result = await postAuthenticatedJson<Plan>('/nutrition/targets/planning', { macros: { PROTEIN: values.PROTEIN.active_value, CARBOHYDRATE: values.CARBOHYDRATE.active_value, FAT: values.FAT.active_value } })
            if (alive) setMacroEnergy(result.macro_calories)
          } catch (e) { if (alive) setMacroError(message(e)) }
        }
      } catch (e) { if (alive) setError(message(e)) }
    }
    void load()
    return () => { alive = false }
  }, [])
  if (error) return <section className="panel"><h1>How your nutrition targets are calculated</h1><p role="alert">{error}</p><Link to="/app/settings">Back to Settings</Link></section>
  if (!plan) return <p role="status">Loading your target explanation…</p>
  return <TargetCalculationGuide plan={plan} active={active} macroEnergy={macroEnergy} macroError={macroError} />
}
