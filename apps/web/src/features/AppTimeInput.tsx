import { useId } from 'react'

export function AppTimeInput({label,value,onChange,name,required=true}:{label:string;value:string;onChange:(value:string)=>void;name?:string;required?:boolean}) {
  const id=useId(), [hour='',minute='']=value.split(':')
  const valid=/^([01]\d|2[0-3]):[0-5]\d$/.test(value)
  return <div className="app-time-input" role="group" aria-labelledby={id}>
    <span id={id} className="app-date-label">{label}</span>
    {name&&<input type="hidden" name={name} value={valid?value:''}/>}
    <div className="app-time-segments">
      <label><span className="sr-only">{label} hour (24-hour)</span><select required={required} aria-label={label+' hour (24-hour)'} value={hour} onChange={e=>onChange(`${e.target.value}:${minute}`)}><option value="">Hour</option>{Array.from({length:24},(_,n)=>String(n).padStart(2,'0')).map(h=><option key={h} value={h}>{h}</option>)}</select></label>
      <span aria-hidden="true">:</span>
      <label><span className="sr-only">{label} minute</span><select required={required} aria-label={label+' minute'} value={minute} onChange={e=>onChange(`${hour}:${e.target.value}`)}><option value="">Minute</option>{Array.from({length:60},(_,n)=>String(n).padStart(2,'0')).map(m=><option key={m} value={m}>{m}</option>)}</select></label>
      <small>24-hour</small>
    </div>
  </div>
}
