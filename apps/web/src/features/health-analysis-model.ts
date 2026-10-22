import {calendarDay, occurrenceRange, type HealthOccurrence} from './health-occurrences'
import {rangeDays, shiftDay} from './nutrition/model'

export type AnalysisItem = {id:string;name:string;type:string;active:boolean}
export type AnalysisOccurrence = HealthOccurrence & {health_item_id:string}
export type AnalysisContext = {timezone:string;daily_reset_time:string}
export type AnalysisRange = {from:string;to:string;revision:number}
export type DailyHealth = {day:string;done:number;missed:number;skipped:number;active:number;due:number;dueDone:number}
export type JsonGetter = <T>(path:string)=>Promise<T>
type Page<T> = {items:T[];page:{next_cursor:string|null;has_more?:boolean}}

export function validateAnalysisRange(from:string,to:string) {
  const valid=(day:string)=>/^\d{4}-\d{2}-\d{2}$/.test(day)&&Number.isFinite(Date.parse(day))&&new Date(Date.parse(day)).toISOString().slice(0,10)===day
  if(!valid(from)||!valid(to)||from>to||Date.parse(to)-Date.parse(from)>365*86400000) throw new Error('Choose a valid range of at most 366 Tracking Days.')
}
// Same wall-clock rule as Core trackingday.Derive, including repeated DST hours.
// Do not subtract a fixed elapsed duration from an instant.
export function healthTrackingDay(instant:string,context:AnalysisContext) {
  const date=new Date(instant)
  if(!Number.isFinite(date.getTime())||!/^([01]\d|2[0-3]):[0-5]\d$/.test(context.daily_reset_time)) throw new Error('Invalid health analysis time context.')
  const parts=new Intl.DateTimeFormat('en-GB',{timeZone:context.timezone,hour:'2-digit',minute:'2-digit',hourCycle:'h23'}).formatToParts(date)
  const time=['hour','minute'].map(key=>parts.find(p=>p.type===key)!.value).join(':')
  const day=calendarDay(context.timezone,date)
  return time<context.daily_reset_time?shiftDay(day,-1):day
}
export function itemDailyHealth(rows:AnalysisOccurrence[],id:string,from:string,to:string,context:AnalysisContext,asOf:number):DailyHealth[] {
  const buckets=rangeDays(from,to).map(day=>({day,done:0,missed:0,skipped:0,active:0,due:0,dueDone:0}))
  const byDay=new Map(buckets.map(bucket=>[bucket.day,bucket]))
  const seen=new Set<string>()
  for(const row of rows) {
    if(row.health_item_id!==id||seen.has(row.occurrence_id))continue
    seen.add(row.occurrence_id)
    const bucket=byDay.get(healthTrackingDay(row.scheduled_at,context))
    if(!bucket)continue
    const due=Date.parse(row.scheduled_at)<=asOf
    const status=row.status==='RESCHEDULED'?(due?'MISSED':'SCHEDULED'):row.status
    if(status==='TAKEN')bucket.done++
    else if(status==='SKIPPED')bucket.skipped++
    else if(status==='MISSED'||(status==='SCHEDULED'&&due))bucket.missed++
    else if(status==='SCHEDULED')bucket.active++
    else throw new Error('An occurrence has an unsupported status. Refresh before interpreting these charts.')
    if(due){bucket.due++;if(status==='TAKEN')bucket.dueDone++}
  }
  return buckets
}
export async function readAnalysisPages<T>(get:JsonGetter,path:string,query:Record<string,string>,current:()=>boolean=()=>true):Promise<T[]> {
  const rows:T[]=[],seen=new Set<string>()
  let cursor:string|null=null
  do {
    if(!current())throw new Error('Analysis request replaced.')
    const params:URLSearchParams=new URLSearchParams({...query,...(cursor?{cursor}:{})})
    const page:Page<T>=await get<Page<T>>(`${path}?${params}`)
    if(!current())throw new Error('Analysis request replaced.')
    rows.push(...page.items)
    if(rows.length>50000)throw new Error('This range has too many records. Choose fewer dates; incomplete results are not shown.')
    const next:string|null=page.page.next_cursor
    if((page.page.has_more&&!next)||(next&&seen.has(next)))throw new Error('Could not load the complete analysis. Try again or choose fewer dates.')
    if(next)seen.add(next)
    cursor=next
  }while(cursor)
  return rows
}
export async function readHealthAnalysis(get:JsonGetter,range:AnalysisRange,context:AnalysisContext,current:()=>boolean=()=>true) {
  validateAnalysisRange(range.from,range.to)
  // Read broad civil-date windows, then bucket by reset time. Extra final date
  // covers events before reset on the morning after the selected last day.
  // 28-day chunks stay under the API's 31-elapsed-day cap across DST.
  const rows:AnalysisOccurrence[]=[]
  const last=shiftDay(range.to,1)
  for(let from=range.from;from<=last;from=shiftDay(from,28)) {
    const to=shiftDay(from,27)<last?shiftDay(from,27):last
    rows.push(...await readAnalysisPages<AnalysisOccurrence>(get,'/health-items/occurrences',{...occurrenceRange(from,to,context.timezone),limit:'100'},current))
    if(rows.length>50000)throw new Error('This range has too many occurrences. Choose fewer dates; incomplete results are not shown.')
  }
  const unique=[...new Map(rows.map(row=>[row.occurrence_id,row])).values()]
  for(const row of unique) {
    if(!row.health_item_id||!Number.isFinite(Date.parse(row.scheduled_at))||!['TAKEN','SKIPPED','MISSED','SCHEDULED','RESCHEDULED'].includes(row.status))throw new Error('Some health records could not be interpreted. Refresh before interpreting these charts.')
  }
  return unique
}
