import { Form } from '../ui'
import { AppDatePicker } from '../AppDatePicker'
import { healthStatusLabel } from '../health-model'
import { useEffect, useState } from 'react'
import { getAuthenticatedJson } from '../../api'
import { shiftDay, type Page } from '../nutrition/model'

type Event = { type:string; occurrence_id:string; name:string; status:string; scheduled_at:string; scheduled_local:{date:string;time:string;timezone:string}; completed_at?:string; rescheduled_to?:string }
export function HealthAdherence({revision}:{revision:string}) {
 const [from,setFrom]=useState(()=>shiftDay(new Date().toISOString().slice(0,10),-6)),[to,setTo]=useState(()=>new Date().toISOString().slice(0,10)),[rows,setRows]=useState<Event[]>([]),[busy,setBusy]=useState(true),[error,setError]=useState('')
 async function load(){
  if(!from||!to||from>to||Date.parse(to)-Date.parse(from)>30*86400000){setError('Select up to 31 calendar days.');return}
  setBusy(true);setError('')
  try{const result:Event[]=[];let cursor:string|null=null;do{const query=new URLSearchParams({from:`${from}T00:00:00Z`,to:`${shiftDay(to,1)}T00:00:00Z`,limit:'100',...(cursor?{cursor}:{})});const page:Page<Event>=await getAuthenticatedJson(`/health-items/occurrences?${query}`);result.push(...page.items);cursor=page.page.next_cursor}while(cursor);setRows(result)}catch(cause){setError(cause instanceof Error?cause.message:'Could not load adherence.')}finally{setBusy(false)}
 }
 useEffect(()=>{void load()},[revision])
 const statuses=['TAKEN','MISSED','SCHEDULED'],counts=Object.fromEntries(statuses.map(status=>[status,rows.filter(row=>status==='MISSED'?(row.status==='MISSED'||row.status==='SKIPPED'):row.status===status).length]))
 const due=rows.filter(row=>Date.parse(row.scheduled_at)<=Date.now()),taken=due.filter(row=>row.status==='TAKEN').length
 return <section className="panel section-panel"><h2>Schedule adherence & history</h2><p className="muted">A record of your actions, not medical advice. The date filter uses UTC; each event shows its saved schedule timezone. Tasks marked done and medication/supplements marked taken count as completed. Future events are excluded from the completion percentage.</p><Form className="analysis-filter" onSubmit={e=>{e.preventDefault();void load()}}><AppDatePicker label="History from (UTC)" value={from} onChange={setFrom}/><AppDatePicker label="History through (UTC)" min={from} value={to} onChange={setTo}/><button className="button button-outline" disabled={busy}>{busy?'Loading…':'Review history'}</button></Form>{error&&<p role="alert" className="form-error">{error}</p>}{busy ? <p role="status">Loading schedule history…</p> : !error && <><p>{due.length?`${taken} of ${due.length} due occurrences completed (${Math.round(taken/due.length*100)}%).`:'No due occurrences in this range.'}</p><div className="adherence-counts">{statuses.map(status=><span key={status}>{status === 'TAKEN' ? 'done' : status==='MISSED'?'missed / skipped':'active'}: <strong>{counts[status]}</strong></span>)}</div><div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><thead><tr><th>Scheduled</th><th>Item</th><th>Recorded status</th></tr></thead><tbody>{rows.map(row=><tr key={row.occurrence_id}><th>{row.scheduled_local.date} {row.scheduled_local.time}<small className="wellness-note">{row.scheduled_local.timezone}</small></th><td data-label="Item">{row.name}</td><td data-label="Status">{healthStatusLabel(row.status, row.type)}{row.rescheduled_to&&<small className="wellness-note">Moved to {new Date(row.rescheduled_to).toLocaleString()}</small>}</td></tr>)}</tbody></table></div></>}</section>
}
