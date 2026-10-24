import { useState, type FormEvent } from 'react'
import { Form } from '../ui'
import { NutrientInputs } from './NutrientInputs'
import { nutrientFields, type Nutrition } from './model'
import { AppDatePicker } from '../AppDatePicker'
import { AppTimeInput } from '../AppTimeInput'
import { CheckOption } from '../CheckOption'

export function LogTimestampInput({value,onChange}:{value:string;onChange:(value:string)=>void}) {
  const now=new Date(), today=`${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}-${String(now.getDate()).padStart(2,'0')}`
  const [date,time]=value?value.split('T'):[today,`${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}`]
  return <details className="secondary-details"><summary>Change log date and time</summary><div className="auth-form"><AppDatePicker label="Log date (device timezone)" value={date} max={today} onChange={next=>onChange(`${next}T${time}`)}/><AppTimeInput label="Log time (24-hour)" value={time} onChange={next=>onChange(`${date}T${next}`)}/>{value&&<button type="button" className="quiet-link" onClick={()=>onChange('')}>Use current time</button>}<small>{value?'Using your device timezone. Your tracking day uses the account timezone.':'Logging now. Change these fields to choose another time.'}</small></div></details>
}

export type ManualFood = { name: string; description: string; food_state: string; base_quantity: {amount: string; unit: string}; nutrients: Nutrition }
export function quickLogNutrition(data:FormData) {
  const zero=data.get('zero')==='on', nutrients: Nutrition={}
  for(const [code,unit] of nutrientFields) {
    const raw=String(data.get(code) || '').trim()
    nutrients[code]=raw ? {status:'KNOWN',amount:raw,unit} : zero && data.has(code) ? {status:'KNOWN',amount:'0',unit} : {status:'UNKNOWN',unit}
  }
  return nutrients
}
export function QuickLog({busy,onAdd,category,onCategory,loggedAt,onLoggedAt}: {busy: boolean; onAdd: (food: ManualFood, save: boolean, review: boolean) => Promise<void>; category:string;onCategory:(value:string)=>void;loggedAt:string;onLoggedAt:(value:string)=>void}) {
  const [error,setError]=useState(''), [entry,setEntry]=useState(0)
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault(); setError('')
    const form=event.currentTarget, data=new FormData(form), values=Object.fromEntries(data)
    const nutrients=quickLogNutrition(data)
    if(!Object.values(nutrients).some(value=>value.status==='KNOWN')) {setError('Enter at least one nutrition value.');return}
    try {
      const review=(event.nativeEvent as SubmitEvent).submitter?.getAttribute('value')==='review'
      await onAdd({name:String(values.name).trim(),description:'',food_state:'PREPARED',base_quantity:{amount:String(values.amount),unit:String(values.unit)},nutrients},data.get('save')==='on',review)
      form.reset()
      setEntry(previous=>previous+1)
    } catch(cause) {setError(cause instanceof Error?cause.message:'Could not add this food.')}
  }
  return <Form className="auth-form quick-log-form" onSubmit={submit}>
    <div><h2>Quick log</h2><p className="muted">Enter nutrition for the amount you ate. Add it to your meal, then review everything before logging.</p></div>
    <fieldset className="tracked-selection auth-form" disabled={busy}>
      <legend className="sr-only">One-time food details</legend>
      <label>Food name<input name="name" placeholder="For example, ramen" minLength={2} maxLength={120} required /></label>
      <div className="custom-food-base"><label>Amount eaten<input name="amount" type="number" min="0.001" step="any" defaultValue="100" required /></label><label>Unit<select name="unit"><option>g</option><option>ml</option></select></label></div>
      <label>Meal<select value={category} onChange={event=>onCategory(event.target.value)}>{['BREAKFAST','LUNCH','DINNER','SNACK','OTHER'].map(value=><option key={value} value={value}>{value.charAt(0)+value.slice(1).toLowerCase()}</option>)}</select></label>
      <LogTimestampInput value={loggedAt} onChange={onLoggedAt}/>
      <NutrientInputs key={entry} />
      <div className="quick-log-choices"><CheckOption name="zero" label="Use zero for empty fields" description="Applies only to the nutrient fields shown here. Other nutrients stay unknown." defaultChecked /><CheckOption name="save" label="Save to Food Library" description="Reuse this food next time without entering its nutrition again." /></div>
      <div className="history-actions"><button className="button button-outline" type="submit" value="add">Add to meal</button><button className="button button-primary" type="submit" value="review">Review meal</button></div><small>You’ll confirm the meal on the next screen before it is logged.</small>
    </fieldset>{error && <p className="form-message form-error" role="alert">{error}</p>}
  </Form>
}
