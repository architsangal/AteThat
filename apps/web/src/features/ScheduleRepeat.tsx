import {useId} from 'react'

export function repeatCadence(frequency:string,interval:number) {
  if(frequency==='PAUSED')return 'Schedule paused'
  if(!Number.isInteger(interval)||interval<1||interval>365)return 'Choose a whole-number repeat interval from 1 to 365'
  const unit=frequency==='WEEKLY'?'week':frequency==='MONTHLY'?'month':'day'
  return interval===1?`Every ${unit}`:interval===2?`Every other ${unit}`:`Every ${interval} ${unit}s`
}

export function RepeatInterval({frequency,value,onChange}:{frequency:string;value:string;onChange:(value:string)=>void}) {
  const id=useId()
  const unit=frequency==='WEEKLY'?'week':frequency==='MONTHLY'?'month':'day'
  return <div className="repeat-interval"><label htmlFor={id}>Repeat every</label><div className="repeat-interval-control"><input id={id} name="interval" type="number" min="1" max="365" step="1" required value={value} onChange={e=>onChange(e.target.value)} aria-label={`Repeat interval (${unit}s)`} aria-describedby={id+'-help'}/><span>{Number(value)===1?unit:unit+'s'}</span></div><p id={id+'-help'}>1 = every {unit} · 2 = every other {unit}. This controls which dates repeat, not how many times each day.</p></div>
}

export function SchedulePreview({frequency,interval,times,weekdays=[],monthDays=[],start}:{frequency:string;interval:number;times:string[];weekdays?:string[];monthDays?:number[];start:string}) {
  const validTimes=times.length>0&&times.every(time=>/^([01]\d|2[0-3]):[0-5]\d$/.test(time))&&new Set(times).size===times.length
  const startText=/^\d{4}-\d{2}-\d{2}$/.test(start)&&Number.isFinite(Date.parse(start))?new Intl.DateTimeFormat('en-GB',{day:'numeric',month:'short',year:'numeric',timeZone:'UTC'}).format(new Date(start+'T12:00:00Z')):''
  return <div className="schedule-preview" role="status" aria-live="polite"><strong>Your schedule</strong><p>{repeatCadence(frequency,interval)}{frequency==='WEEKLY'?weekdays.length?` on ${weekdays.join(', ')}`:' — choose the weekdays':frequency==='MONTHLY'?monthDays.length?` on dates ${monthDays.slice().sort((a,b)=>a-b).join(', ')}`:' — choose the dates':''}.</p>{frequency!=='PAUSED'&&<p>{validTimes?`${times.length===1?'Once':times.length===2?'Twice':`${times.length} times`} on each scheduled day, at ${times.join(' and ')} (24-hour time).`:'Choose complete, different times for each occurrence.'}</p>}{startText&&<p>Starting {startText}. {frequency!=='PAUSED'?'The repeat interval starts from this date.':''}</p>}</div>
}
