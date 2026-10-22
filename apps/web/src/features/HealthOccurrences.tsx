import {useEffect,useId,useRef,useState} from 'react'
import {getAuthenticatedJson} from '../api'
import {AppDatePicker} from './AppDatePicker'
import {calendarDay,occurrenceMatches,occurrenceRange,occurrenceTabs,type HealthOccurrence,type OccurrenceTab} from './health-occurrences'
import {healthStatusLabel} from './health-model'
import {Form,Loading} from './ui'

export function HealthOccurrenceResults({rows,tab,busy,onAction,onReschedule}:{rows:HealthOccurrence[];tab:OccurrenceTab;busy:boolean;onAction:(row:HealthOccurrence,status:'TAKEN'|'SKIPPED')=>void;onReschedule:(row:HealthOccurrence)=>void}) {
  const visible=rows.filter(row=>occurrenceMatches(row,tab))
  return <>{visible.length===0?<div className="occurrence-empty"><h3>No {tab==='ALL'?'scheduled':occurrenceTabs.find(([key])=>key===tab)?.[1].toLowerCase()} occurrences in this range</h3><p>Choose another status or change the dates. Your saved routines are still available below.</p></div>:visible.map(row=>{
    const pending=row.status==='SCHEDULED'||row.status==='MISSED'||row.status==='SKIPPED'
    return <article className="occurrence-card" key={row.occurrence_id}>
      <div className="occurrence-description"><div className="occurrence-title"><h3>{row.name}</h3><span className={'occurrence-status occurrence-status-'+row.status.toLowerCase()}>{row.status==='TAKEN'?'Done':row.status==='SKIPPED'?'Skipped by you':row.status==='SCHEDULED'?'Active':healthStatusLabel(row.status,row.type)}</span></div><p><time dateTime={row.scheduled_at}>{row.scheduled_local.date} · {row.scheduled_local.time}</time><span> · {row.scheduled_local.timezone}</span></p><small>{row.type==='TASK'?'Task':row.type==='MEDICATION'?'Medication':'Supplement'}{row.type!=='TASK'&&row.status==='TAKEN'?' · recorded as taken':''}</small>{row.original_scheduled_at&&<p className="muted">Rescheduled from {new Date(row.original_scheduled_at).toLocaleString()} (device timezone). This occurrence now belongs to the date shown above.</p>}</div>
      <div className="occurrence-actions">{pending&&<><button className="button button-outline button-small" type="button" disabled={busy} onClick={()=>onAction(row,'TAKEN')}>{row.type==='TASK'?'Mark done':'Mark taken'}</button><button className="quiet-link" type="button" disabled={busy||row.status==='SKIPPED'} onClick={()=>onAction(row,'SKIPPED')}>Skip</button></>}<details><summary>{pending?'More actions':'Change status'}</summary><div className="occurrence-secondary">{!pending&&<><button className="quiet-link" type="button" disabled={busy||row.status==='TAKEN'} onClick={()=>onAction(row,'TAKEN')}>{row.type==='TASK'?'Mark done':'Mark taken'}</button><button className="quiet-link" type="button" disabled={busy||row.status==='SKIPPED'} onClick={()=>onAction(row,'SKIPPED')}>Mark skipped</button></>}{row.status!=='TAKEN'&&<button className="quiet-link" type="button" disabled={busy} onClick={()=>onReschedule(row)}>Reschedule</button>}</div></details></div>
    </article>
  })}</>
}

export function HealthOccurrences({timezone,revision,busy,onAction,onReschedule}:{timezone:string;revision:number;busy:boolean;onAction:(row:HealthOccurrence,status:'TAKEN'|'SKIPPED')=>void;onReschedule:(row:HealthOccurrence)=>void}) {
  const [now,setNow]=useState(()=>new Date()),[followToday,setFollowToday]=useState(true)
  const today=calendarDay(timezone,now),[range,setRange]=useState({from:today,to:today}),[from,setFrom]=useState(today),[to,setTo]=useState(today)
  const [tab,setTab]=useState<OccurrenceTab>('ACTIVE'),[rows,setRows]=useState<HealthOccurrence[]>([]),[loading,setLoading]=useState(true),[error,setError]=useState(''),[filterError,setFilterError]=useState(''),[attempt,setAttempt]=useState(0)
  const request=useRef(0),id=useId()
  useEffect(()=>{const tick=()=>{if(!document.hidden)setNow(new Date())};const timer=window.setInterval(tick,60000);window.addEventListener('focus',tick);document.addEventListener('visibilitychange',tick);return()=>{clearInterval(timer);window.removeEventListener('focus',tick);document.removeEventListener('visibilitychange',tick)}},[])
  useEffect(()=>{if(followToday){setRange({from:today,to:today});setFrom(today);setTo(today)}},[today,followToday])
  const liveTick=range.from<=today&&today<=range.to?Math.floor(now.getTime()/60000):0
  useEffect(()=>{
    const sequence=++request.current
    setLoading(true);setError('');setRows([])
    async function load(){
      try {
        const bounds=occurrenceRange(range.from,range.to,timezone),result:HealthOccurrence[]=[],seen=new Set<string>()
        let cursor:string|null=null
        do {
          const query=new URLSearchParams({...bounds,limit:'100',...(cursor?{cursor}:{})})
          const page:{items:HealthOccurrence[];page:{next_cursor:string|null;has_more:boolean}}=await getAuthenticatedJson(`/health-items/occurrences?${query}`)
          if(sequence!==request.current)return
          result.push(...page.items)
          if(result.length>10000) throw new Error('Too many occurrences in this range. Choose a shorter range; incomplete results are not shown.')
          const next=page.page.next_cursor
          if(page.page.has_more&&!next)throw new Error('Could not load the complete range. Try again or choose fewer dates.')
          if(next&&seen.has(next))throw new Error('Could not load the complete range. Please try again.')
          if(next)seen.add(next)
          cursor=next
        }while(cursor)
        if(sequence===request.current)setRows([...new Map(result.map(row=>[row.occurrence_id,row])).values()])
      }catch(cause){if(sequence===request.current)setError(cause instanceof Error?cause.message:'Could not load scheduled occurrences.')}
      finally{if(sequence===request.current)setLoading(false)}
    }
    void load()
    return()=>{request.current++}
  },[range.from,range.to,timezone,revision,attempt,liveTick])
  function apply(next:{from:string;to:string}) {
    try{occurrenceRange(next.from,next.to,timezone);setRange(next);setFollowToday(next.from===today&&next.to===today);setAttempt(n=>n+1);setFilterError('')}catch(cause){setFilterError((cause as Error).message)}
  }
  return <section className="panel health-occurrences" aria-labelledby={id+'-title'}>
    <div className="panel-top"><div><h2 id={id+'-title'}>Scheduled occurrences</h2><p className="muted">Today’s tasks, medication and supplements, with completed events kept separate.</p></div></div>
    <Form className="occurrence-filters" onSubmit={e=>{e.preventDefault();apply({from,to})}}><AppDatePicker label="From date" value={from} onChange={setFrom}/><AppDatePicker label="Through date" value={to} min={from} onChange={setTo}/><div className="occurrence-range-actions"><button className="button button-primary" disabled={busy}>Apply range</button><button className="button button-outline" type="button" disabled={busy} onClick={()=>{const date=calendarDay(timezone);setFrom(date);setTo(date);apply({from:date,to:date})}}>Today</button></div></Form>
    {filterError&&<p className="form-message form-error" role="alert">{filterError}</p>}
    <p className="occurrence-range-caption">{range.from===range.to?range.from:`${range.from} – ${range.to}`} · {timezone} calendar dates · up to 31 days. This range applies to every status tab.</p>
    <div className="occurrence-tabs" role="tablist" aria-label="Occurrence status">{occurrenceTabs.map(([key,label])=><button key={key} id={id+'-'+key} role="tab" aria-selected={tab===key} aria-controls={id+'-results'} tabIndex={tab===key?0:-1} onKeyDown={e=>{if(!['ArrowLeft','ArrowRight','Home','End'].includes(e.key))return;e.preventDefault();const index=occurrenceTabs.findIndex(([value])=>value===key),next=e.key==='Home'?0:e.key==='End'?occurrenceTabs.length-1:(index+(e.key==='ArrowRight'?1:-1)+occurrenceTabs.length)%occurrenceTabs.length;setTab(occurrenceTabs[next][0]);document.getElementById(id+'-'+occurrenceTabs[next][0])?.focus()}} onClick={()=>setTab(key)}>{label}<span aria-hidden="true">{loading?'…':error?'—':rows.filter(row=>occurrenceMatches(row,key)).length}</span></button>)}</div>
    <div id={id+'-results'} role="tabpanel" aria-labelledby={id+'-'+tab} aria-busy={loading}>{loading?<Loading>Loading occurrences for this range…</Loading>:error?<div className="form-message form-error" role="alert"><p>{error}</p><button className="button button-outline" onClick={()=>setAttempt(n=>n+1)}>Try again</button></div>:<HealthOccurrenceResults rows={rows} tab={tab} busy={busy} onAction={onAction} onReschedule={onReschedule}/>}</div>
    <p className="occurrence-help">Active means pending. Done includes medication and supplements recorded as taken. Missed also includes occurrences you deliberately skipped, labelled “Skipped by you”. Rescheduling moves an unfinished occurrence to its new date and makes it active again. Times on each event use its saved schedule timezone.</p>
  </section>
}
