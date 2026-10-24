import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getAuthenticatedJson, patchAuthenticatedJson } from '../api'
import { Form, Loading } from './ui'
import { AutomaticTargetProposal } from './TargetSettings'
import { AppDatePicker } from './AppDatePicker'

type Profile = {version:number; display_name:string; timezone:string; date_of_birth?:string; height?:{amount:string;unit:string}; weight?:{amount:string;unit:string}; goal?:string; activity_level?:string; sex?:string}
export function Welcome() {
  const navigate = useNavigate()
  const heading = useRef<HTMLHeadingElement>(null)
  const [profile,setProfile] = useState<Profile|null>(null)
  const [dob,setDob]=useState('')
  const [step,setStep] = useState(0)
  useEffect(() => { window.scrollTo({top:0,behavior:'instant'}); heading.current?.focus({preventScroll:true}) },[step])
  const [busy,setBusy] = useState(false)
  const [error,setError] = useState('')
  const load = () => { setError(''); getAuthenticatedJson<Profile>('/profile').then(value=>{setProfile(value);setDob(value.date_of_birth||'')}).catch(() => setError('Could not load your profile. Please try again.')) }
  useEffect(load,[])
  async function save(event:React.FormEvent<HTMLFormElement>) {
    event.preventDefault(); if (!profile) return
    const values=Object.fromEntries(new FormData(event.currentTarget)) as Record<string,string>
    setBusy(true);setError('')
    try {
      const body:Record<string,unknown>={version:profile.version,display_name:values.display_name,date_of_birth:values.date_of_birth,timezone:values.timezone,goal:values.goal}
      if(values.activity_level) body.activity_level=values.activity_level
      if(values.sex) body.sex=values.sex
      if(values.height) body.height={amount:values.height,unit:'cm'}
      if(values.weight) body.weight={amount:values.weight,unit:'kg'}
      const updated=await patchAuthenticatedJson<Profile>('/profile',body)
      setProfile(updated);setStep(1)
    } catch(cause) {setError(cause instanceof Error?cause.message:'Could not save profile.')} finally {setBusy(false)}
  }
  async function finish(path='/app') {
    setBusy(true);setError('')
    try { const latest=await getAuthenticatedJson<Profile>('/profile');await patchAuthenticatedJson('/profile',{version:latest.version,onboarding_complete:true});navigate(path,{replace:true}) }
    catch(cause) {setError(cause instanceof Error?cause.message:'Could not finish setup.')} finally {setBusy(false)}
  }
  const daily=[
    ['Log food','Open Log food → search AteThat foods → Add → Review → Log food. Use Search USDA to find a new food; your remaining daily allowance is shown there.'],
    ['Make it yours','In Selected foods, edit the amount. Open More options to save a favourite, define a household portion, or edit nutrition for that entry.'],
    ['Find your favourites','In Log food, open Recent, favourites & saved portions. Reuse a food or portion without repeating a search.'],
    ['Check your history','Open History to browse a tracking day, see meal details, correct an amount with a review, or remove a meal.'],
  ]
  const more=[
    ['Food Library','Create a private custom food with nutrition values, or save several selected foods as a reusable meal.'],
    ['Recipes','Create a recipe from ingredients, set the batch yield, review its nutrition, then log a whole or fractional serving.'],
    ['Water, weight & activity','Record water with quick amounts, track weight, and record activity. Open history and trends to see earlier entries.'],
    ['Health items','Schedule medication, supplements and everyday tasks daily, weekly or monthly. Mark medication/supplements taken, tasks done, or occurrences skipped, and review completion history.'],
    ['Targets & analysis','Review automatically suggested starting targets after profile setup. Edit independent targets in Settings and compare intake in Analysis. Text Summary gives an editable factual report.'],
    ['Your account','In Settings, update your profile, tracking-day reset, units or theme. The help icon opens this guide again anytime.'],
  ]
  return <section className="welcome-panel"><div className="page-head"><div><span className="eyebrow">Welcome to AteThat · Step {step+1} of 3</span><h1 ref={heading} tabIndex={-1}>{step===0?'A little about you':step===1?'Your first food log':'Explore your daily tools'}</h1><p>{step===0?'Add what you are comfortable sharing. Every optional detail can be added later in Settings.':'A quick guide you can reopen from the help icon or Settings.'}</p></div></div>
    <button className="quiet-link" type="button" disabled={busy} onClick={()=>void finish()}>Skip setup and guide</button>
    <div className="welcome-progress" aria-hidden="true">{[0,1,2].map(n=><span key={n} className={n<=step?'active':''}/>)}</div>
    {error && <p className="form-message form-error" role="alert">{error}</p>}
    {!profile ? error ? <button className="button button-outline" onClick={load}>Try again</button> : <Loading>Loading your profile…</Loading> : <div className="panel">
      {step===0 ? <Form className="auth-form" onSubmit={save}><label>Display name<input name="display_name" defaultValue={profile.display_name} required maxLength={80}/></label><div><AppDatePicker label="Date of birth (optional)" name="date_of_birth" value={dob} onChange={setDob} max={new Date().toISOString().slice(0,10)} describedBy="welcome-dob-help" clearable/><small id="welcome-dob-help">Used to check eligibility for adult nutrition benchmarks.</small></div><div className="recipe-meta"><label>Height in cm (optional)<input name="height" type="number" min="1" step="any" defaultValue={profile.height?.unit==='cm'?profile.height.amount:''}/></label><label>Weight in kg (optional)<input name="weight" type="number" min="1" step="any" defaultValue={profile.weight?.unit==='kg'?profile.weight.amount:''}/></label></div><p className="muted">If missing, planning assumes 175 cm, 70.4375 kg and Moderate activity. These are not recorded measurements; no age or sex is guessed.</p><label>Activity (optional)<select name="activity_level" defaultValue={profile.activity_level || ''}><option value="">Moderate (assumed)</option><option value="SEDENTARY">Sedentary</option><option value="LIGHT">Light</option><option value="MODERATE">Moderate</option><option value="ACTIVE">Active</option><option value="VERY_ACTIVE">Very active</option></select></label><label>Equation-specific sex (optional)<select name="sex" defaultValue={profile.sex || ''}><option value="">Not supplied</option><option value="FEMALE">Female</option><option value="MALE">Male</option><option value="DECLINED">Prefer not to provide</option></select></label><label>Tracking goal (optional)<input name="goal" defaultValue={profile.goal} maxLength={120} placeholder="For example, understand my daily food habits"/></label><label>Account timezone<select name="timezone" defaultValue={profile.timezone}>{Array.from(new Set([profile.timezone,'UTC',...Intl.supportedValuesOf('timeZone')])).map(zone=><option key={zone}>{zone}</option>)}</select><small>Your food history uses this timezone and your tracking-day reset.</small></label><div className="dialog-actions"><button type="button" className="button button-outline" disabled={busy} onClick={()=>setStep(1)}>Skip profile</button><button className="button button-primary" disabled={busy}>{busy?'Saving…':'Save and continue'}</button></div></Form> : <>{step===1&&<AutomaticTargetProposal/>}<div className="guide-grid">{(step===1?daily:more).map(([title,description])=><article className="guide-card" key={title}><h3>{title}</h3><p>{description}</p></article>)}</div><div className="dialog-actions"><button className="button button-outline" disabled={busy} onClick={()=>setStep(step-1)}>Back</button>{step===1?<button className="button button-primary" onClick={()=>setStep(2)}>Next: your other tools</button>:<button className="button button-primary" disabled={busy} onClick={()=>void finish('/app/log-food')}>{busy?'Finishing…':'Start logging food'}</button>}</div></>}

    </div>}
  </section>
}
