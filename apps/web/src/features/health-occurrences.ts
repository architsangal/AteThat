export type HealthOccurrence = {occurrence_id:string;name:string;type:string;status:string;scheduled_at:string;scheduled_local:{date:string;time:string;timezone:string};completed_at?:string;rescheduled_to?:string;original_scheduled_at?:string;event_version?:number}
export const occurrenceTabs=[['ACTIVE','Active'],['TAKEN','Done'],['MISSED','Missed'],['ALL','All']] as const
export type OccurrenceTab=typeof occurrenceTabs[number][0]
export function occurrenceMatches(row:Pick<HealthOccurrence,'status'>,tab:OccurrenceTab) { return tab==='ALL'||(tab==='MISSED'?(row.status==='MISSED'||row.status==='SKIPPED'):row.status===(tab==='ACTIVE'?'SCHEDULED':tab)) }
export function calendarDay(timezone:string,instant=new Date()) {
  const parts=new Intl.DateTimeFormat('en-CA',{timeZone:timezone,year:'numeric',month:'2-digit',day:'2-digit'}).formatToParts(instant)
  return ['year','month','day'].map(type=>parts.find(part=>part.type===type)!.value).join('-')
}
function shifted(day:string,days:number) {return new Date(Date.parse(day+'T00:00:00Z')+days*86400000).toISOString().slice(0,10)}
function validDay(day:string) {return /^\d{4}-\d{2}-\d{2}$/.test(day)&&Number.isFinite(Date.parse(day))&&shifted(day,0)===day}
// Find the first instant of a civil date, not a fixed 24-hour offset. This also
// handles short/long DST days and zones whose midnight is skipped.
function dayBoundary(day:string,timezone:string) {
  const midnight=Date.parse(day+'T00:00:00Z')
  let lo=midnight-36*3600000,hi=midnight+36*3600000
  const formatter=new Intl.DateTimeFormat('en-CA',{timeZone:timezone,year:'numeric',month:'2-digit',day:'2-digit'})
  const at=(n:number)=>{const parts=formatter.formatToParts(new Date(n));return ['year','month','day'].map(type=>parts.find(part=>part.type===type)!.value).join('-')}
  while(lo<hi){const mid=Math.floor((lo+hi)/2);if(at(mid)<day)lo=mid+1;else hi=mid}
  return new Date(lo).toISOString()
}
export function occurrenceRange(from:string,to:string,timezone:string) {
  if(!validDay(from)||!validDay(to)||from>to) throw new Error('Choose a valid start and end date; the end cannot be before the start.')
  if(Date.parse(to)-Date.parse(from)>30*86400000) throw new Error('Choose up to 31 calendar days at a time.')
  const start=dayBoundary(from,timezone),end=dayBoundary(shifted(to,1),timezone)
  if(start>=end) throw new Error('This calendar date does not exist in the selected timezone.')
  return {from:start,to:end}
}
