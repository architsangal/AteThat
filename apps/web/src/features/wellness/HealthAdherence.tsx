import { useEffect, useState } from 'react'
import { getAuthenticatedJson } from '../../api'
import { shiftDay, type Page } from '../nutrition/model'

type Event = { occurrence_id:string; name:string; status:string; scheduled_at:string; scheduled_local:{date:string;time:string;timezone:string}; completed_at?:string; rescheduled_to?:string }
export function HealthAdherence({revision}:{revision:string}) {
 const [from,setFrom]=useState(()=>shiftDay(new Date().toISOString().slice(0,10),-6)),[to,setTo]=useState(()=>new Date().toISOString().slice(0,10)),[rows,setRows]=useState<Event[]>([]),[busy,setBusy]=useState(false),[error,setError]=useState('')
 async function load(){
  if(!from||!to||from>to||Date.parse(to)-Date.parse(from)>30*86400000){setError('Select up to 31 calendar days.');return}
  setBusy(true);setError('')
  try{const result:Event[]=[];let cursor:string|null=null;do{const query=new URLSearchParams({from:`${from}T00:00:00Z`,to:`${shiftDay(to,1)}T00:00:00Z`,limit:'100',...(cursor?{cursor}:{})});const page:Page<Event>=await getAuthenticatedJson(`/health-items/occurrences?${query}`);result.push(...page.items);cursor=page.page.next_cursor}while(cursor);setRows(result)}catch(cause){setError(cause instanceof Error?cause.message:'Could not load adherence.')}finally{setBusy(false)}
 }
 useEffect(()=>{void load()},[revision])
 const statuses=['TAKEN','MISSED','SKIPPED','RESCHEDULED','SCHEDULED'],counts=Object.fromEntries(statuses.map(status=>[status,rows.filter(row=>row.status===status).length]))
 const due=rows.filter(row=>Date.parse(row.scheduled_at)<=Date.now()),taken=due.filter(row=>row.status==='TAKEN').length
 return <section className="panel section-panel"><h2>Schedule adherence & history</h2><p className="muted">A record of your actions, not medical advice. The date filter uses UTC; each event shows its saved schedule timezone. Future events are excluded from the taken/due percentage.</p><form className="analysis-filter" onSubmit={e=>{e.preventDefault();void load()}}><label>From<input type="date" required value={from} onChange={e=>setFrom(e.target.value)}/></label><label>Through<input type="date" required min={from} value={to} onChange={e=>setTo(e.target.value)}/></label><button className="button button-outline" disabled={busy}>{busy?'Loading…':'Review history'}</button></form>{error&&<p role="alert" className="form-error">{error}</p>}<p>{due.length?`${taken} of ${due.length} due occurrences marked taken (${Math.round(taken/due.length*100)}%).`:'No due occurrences in this range.'}</p><div className="adherence-counts">{statuses.map(status=><span key={status}>{status.toLowerCase()}: <strong>{counts[status]}</strong></span>)}</div><div className="nutrition-table-wrap"><table className="nutrition-table"><thead><tr><th>Scheduled</th><th>Item</th><th>Recorded status</th></tr></thead><tbody>{rows.map(row=><tr key={row.occurrence_id}><td>{row.scheduled_local.date} {row.scheduled_local.time}<small className="wellness-note">{row.scheduled_local.timezone}</small></td><td>{row.name}</td><td>{row.status.toLowerCase()}{row.rescheduled_to&&<small className="wellness-note">Moved to {new Date(row.rescheduled_to).toLocaleString()}</small>}</td></tr>)}</tbody></table></div></section>
}
