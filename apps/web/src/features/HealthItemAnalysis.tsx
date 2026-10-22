import {useEffect,useId,useRef,useState} from 'react'
import {Link} from 'react-router-dom'
import {Plus,Settings2} from 'lucide-react'
import {getAuthenticatedJson,patchAuthenticatedJson} from '../api'
import {Form,Loading,Modal} from './ui'
import {itemDailyHealth,readAnalysisPages,readHealthAnalysis,type AnalysisContext,type AnalysisItem,type AnalysisOccurrence,type AnalysisRange,type DailyHealth} from './health-analysis-model'

type Preferences={version:number;analysis_health_item_ids:string[]}
const labels={done:'Done / taken',missed:'Missed',skipped:'Skipped by you',active:'Active / upcoming'} as const
const states=Object.keys(labels) as (keyof typeof labels)[]

export function HealthItemChart({item,days,percent}:{item:AnalysisItem;days:DailyHealth[];percent:boolean}) {
  const chartId=useId()
  const chartHost=useRef<HTMLDivElement>(null)
  const [availableWidth,setAvailableWidth]=useState(680)
  const totals=days.reduce((sum,day)=>({done:sum.done+day.done,missed:sum.missed+day.missed,skipped:sum.skipped+day.skipped,active:sum.active+day.active,due:sum.due+day.due,dueDone:sum.dueDone+day.dueDone}),{done:0,missed:0,skipped:0,active:0,due:0,dueDone:0})
  const scheduled=totals.done+totals.missed+totals.skipped+totals.active
  const max=percent?100:Math.max(1,...days.map(day=>day.done+day.missed+day.skipped+day.active))
  const hasSchedule=scheduled>0
  useEffect(()=>{
    const host=chartHost.current
    if(!host)return
    const observer=new ResizeObserver(entries=>{
      const size=entries[0]?.contentRect.width
      if(size)setAvailableWidth(Math.floor(size))
    })
    observer.observe(host)
    return()=>observer.disconnect()
  },[hasSchedule])
  // Draw in actual CSS pixels instead of stretching a narrow SVG viewBox.
  // Long ranges overflow horizontally, but glyphs and bars keep their proportions.
  const width=Math.max(320,availableWidth,days.length*20+70),step=(width-70)/days.length
  const ticks=new Set([0,Math.floor((days.length-1)/2),days.length-1])
  return <article className="panel health-analysis-card" aria-labelledby={chartId}>
    <div className="panel-top"><div><span className="eyebrow">{item.type==='TASK'?'Task':item.type==='MEDICATION'?'Medication':'Supplement'}{!item.active?' · Paused item':''}</span><h3 id={chartId}>{item.name}</h3></div><div className="health-completion"><strong>{totals.due?`${Math.round(100*totals.dueDone/totals.due)}%`:'—'}</strong><span>of due occurrences completed</span></div></div>
    <div className="health-analysis-counts">{states.map(state=><span key={state}><i className={'health-dot health-bar-'+state}/>{labels[state]} <strong>{totals[state]}</strong></span>)}</div>
    {!scheduled?<p className="occurrence-empty">No scheduled occurrences for this item in the selected range. This is not a missed day.</p>:<figure className="health-analysis-figure"><div ref={chartHost} className="health-chart-scroll" tabIndex={0} role="region" aria-label={`${item.name} chart, scroll horizontally for longer ranges`}><svg width={width} height={215} viewBox={`0 0 ${width} 215`} role="img" aria-label={`${item.name}: daily ${percent?'completion percentage':'occurrence status counts'}. Exact values follow in the daily table.`}>
      {[0,.5,1].map(ratio=><g key={ratio}><line x1="44" x2={width-18} y1={170-ratio*140} y2={170-ratio*140} className="chart-grid"/><text x="5" y={174-ratio*140}>{Number((ratio*max).toFixed(1))}{percent?'%':''}</text></g>)}
      {days.map((day,index)=>{
        let height=0
        const x=44+index*step+step*.15,barWidth=Math.max(3,step*.7)
        if(percent){const value=day.due?100*day.dueDone/day.due:null;return <g key={day.day}>{value!==null&&<rect x={x} y={170-value/max*140} width={barWidth} height={Math.max(1,value/max*140)} className="health-bar-done"><title>{day.day}: {value.toFixed(1)}% · {day.dueDone}/{day.due} due completed</title></rect>}{ticks.has(index)&&<text x={x+barWidth/2} y="198" textAnchor="middle">{day.day.slice(5)}</text>}</g>}
        return <g key={day.day}>{states.map(state=>{const size=day[state]/max*140;height+=size;return size>0&&<rect key={state} x={x} y={170-height} width={barWidth} height={size} className={'health-bar-'+state}><title>{day.day}: {labels[state]} {day[state]}</title></rect>})}{ticks.has(index)&&<text x={x+barWidth/2} y="198" textAnchor="middle">{day.day.slice(5)}</text>}</g>
      })}
    </svg></div><figcaption>{percent?'Daily completion uses due occurrences only; days with no due occurrences have no percentage.':'Daily scheduled occurrence counts, not a medical assessment. Blank dates have no scheduled occurrences.'} Scroll horizontally for long ranges; exact values are below.</figcaption></figure>}
    <p className="muted">{totals.dueDone} of {totals.due} due occurrences completed · {scheduled} scheduled in this range. Future occurrences are excluded from the completion percentage.</p>
    <details className="health-daily-details"><summary>Daily values · {days.length} Tracking Days</summary><div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><thead><tr><th>Tracking Day</th><th>Done / taken</th><th>Missed</th><th>Skipped</th><th>Upcoming</th><th>Due completion</th></tr></thead><tbody>{days.map(day=><tr key={day.day}><th>{day.day}</th><td data-label="Done / taken">{day.done}</td><td data-label="Missed">{day.missed}</td><td data-label="Skipped">{day.skipped}</td><td data-label="Upcoming">{day.active}</td><td data-label="Due completion">{day.due?`${day.dueDone}/${day.due} (${Math.round(100*day.dueDone/day.due)}%)`:'No due occurrences'}</td></tr>)}</tbody></table></div></details>
  </article>
}

export function HealthItemAnalysis({range,context}:{range:AnalysisRange|null;context:AnalysisContext|null}) {
  const [items,setItems]=useState<AnalysisItem[]>([]),[selected,setSelected]=useState<string[]>([]),[draft,setDraft]=useState<string[]>([]),[preferences,setPreferences]=useState<Preferences|null>(null)
  const [configOpen,setConfigOpen]=useState(false),[configBusy,setConfigBusy]=useState(true),[configError,setConfigError]=useState(''),[saveError,setSaveError]=useState(''),[saving,setSaving]=useState(false),[attempt,setAttempt]=useState(0)
  const [rows,setRows]=useState<AnalysisOccurrence[]>([]),[busy,setBusy]=useState(false),[error,setError]=useState(''),[percent,setPercent]=useState(false),[loaded,setLoaded]=useState(false),[asOf,setAsOf]=useState(0)
  const request=useRef(0)
  useEffect(()=>{let active=true;setConfigBusy(true);setConfigError('');Promise.all([readAnalysisPages<AnalysisItem>(getAuthenticatedJson,'/health-items',{limit:'50'},()=>active),getAuthenticatedJson<Preferences>('/preferences')]).then(([list,prefs])=>{if(active){setItems(list);setPreferences(prefs);setSelected(prefs.analysis_health_item_ids||[])}}).catch(cause=>{if(active)setConfigError(cause.message)}).finally(()=>{if(active)setConfigBusy(false)});return()=>{active=false}},[attempt])
  // Refresh both sections on Explore; selecting items also refreshes this section
  // against the already applied range, without silently changing dates.
  useEffect(()=>{
    const sequence=++request.current
    setRows([]);setError('');setLoaded(false)
    if(!range||!context||configBusy||configError||!selected.length){setBusy(false);return}
    setBusy(true)
    readHealthAnalysis(getAuthenticatedJson,range,context,()=>sequence===request.current).then(result=>{if(sequence===request.current){setRows(result);setAsOf(Date.now());setLoaded(true)}}).catch(cause=>{if(sequence===request.current)setError(cause.message)}).finally(()=>{if(sequence===request.current)setBusy(false)})
    return()=>{request.current++}
  },[range,context,selected,configBusy,configError])
  async function save(){
    if(!preferences)return
    setSaving(true);setSaveError('')
    try{const next=await patchAuthenticatedJson<Preferences>('/preferences',{version:preferences.version,analysis_health_item_ids:draft});setPreferences(next);setSelected(next.analysis_health_item_ids||[]);setConfigOpen(false)}
    catch(cause){setSaveError((cause as Error).message+' Your selection has not been saved. Close and reload the page if another tab changed your settings.')}
    finally{setSaving(false)}
  }
  const chosen=selected.map(id=>items.find(item=>item.id===id)).filter((item):item is AnalysisItem=>!!item)
  return <section className="health-analysis-section" aria-labelledby="health-analysis-title">
    <div className="panel health-analysis-heading"><div className="panel-top"><div><span className="eyebrow">Your routines, in focus</span><h2 id="health-analysis-title">Health item analysis</h2><p className="muted">Track individual tasks, medications and supplements over the shared timeline.</p></div><button className="button button-outline" type="button" aria-label="Configure tracked health items" disabled={configBusy||!!configError} onClick={()=>{setDraft(selected);setSaveError('');setConfigOpen(true)}}><Settings2 size={17}/> Choose items <Plus size={16}/></button></div>
      {context&&<p className="analysis-boundaries">Health charts use effective scheduled times grouped by your current Tracking Day: {context.timezone}, reset {context.daily_reset_time}. Rescheduled events count on their new day. Nutrition retains its saved historical day boundaries.</p>}
      {configBusy?<Loading>Loading your tracked items…</Loading>:configError?<div className="form-message form-error" role="alert">{configError}<button className="button button-outline" onClick={()=>setAttempt(n=>n+1)}>Retry item settings</button></div>:!items.length?<p className="occurrence-empty">No health items yet. <Link to="/app/health">Add a task, medication or supplement</Link>, then choose it here.</p>:!selected.length?<p className="occurrence-empty">Choose the health items you want to see. Your selection is saved to your account.</p>:<><div className="tracked-item-chips" aria-label="Selected health items">{chosen.map(item=><span className="filter-chip" key={item.id}>{item.name}</span>)}</div><label className="health-chart-mode">Chart view<select value={percent?'percent':'counts'} onChange={e=>setPercent(e.target.value==='percent')}><option value="counts">Daily status counts</option><option value="percent">Daily completion percentage</option></select></label></>}
    </div>
    {!!selected.length&&!configError&&!configBusy&&(!range?<p className="panel muted">Choose your timeline above and select Explore to load health charts.</p>:!context?<p className="panel form-error" role="alert">Your Tracking Day context is unavailable. Reload before exploring health charts.</p>:busy?<div className="panel"><Loading>Loading all scheduled occurrences…</Loading></div>:error?<div className="panel form-message form-error" role="alert"><p>{error} No partial chart is shown.</p><p>Use Explore to retry.</p></div>:loaded&&<>{selected.length!==chosen.length&&<p className="panel form-error" role="status">Some selected items are no longer available. Choose items to update your selection.</p>}{chosen.map(item=><HealthItemChart key={item.id} item={item} days={itemDailyHealth(rows,item.id,range.from,range.to,context,asOf)} percent={percent}/>)}</>)}
    {configOpen&&<Modal title="Choose tracked health items" busy={saving} onClose={()=>setConfigOpen(false)}><Form onSubmit={e=>{e.preventDefault();void save()}}><div className="health-item-picker"><p>Choose up to 12 items. Paused items remain available so you can review their history. This does not change any schedule or health record.</p><fieldset><legend>Health items to chart · {draft.length}/12</legend>{items.map(item=><label className="health-picker-option" key={item.id}><input type="checkbox" checked={draft.includes(item.id)} disabled={saving||(!draft.includes(item.id)&&draft.length>=12)} onChange={e=>setDraft(e.target.checked?[...draft,item.id]:draft.filter(id=>id!==item.id))}/><span><strong>{item.name}</strong><small>{item.type==='TASK'?'Task':item.type==='MEDICATION'?'Medication':'Supplement'}{!item.active?' · Paused':''}</small></span></label>)}</fieldset><p className="muted">Removing an item here only hides its chart. Your saved data stays unchanged.</p>{saveError&&<p className="form-message form-error" role="alert">{saveError}</p>}</div><div className="dialog-actions"><button className="button button-outline" type="button" disabled={saving} onClick={()=>setConfigOpen(false)}>Cancel</button><button className="button button-primary" disabled={saving}>{saving?'Saving…':'Save tracked items'}</button></div></Form></Modal>}
  </section>
}
