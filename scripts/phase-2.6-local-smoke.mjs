// Disposable local fixtures only. No production or upstream provider access.
import assert from 'node:assert/strict'
import {execFileSync} from 'node:child_process'
import {calendarDay,occurrenceRange} from '../apps/web/src/features/health-occurrences.ts'
const base='http://localhost:18080/api/v1',email=`phase26-${Date.now()}@example.invalid`,password='Local-only-phase26-test-2026!'
let token=''
async function call(path,method='GET',body,expected=200){const response=await fetch(base+path,{method,headers:{'Content-Type':'application/json',...(token?{Authorization:`Bearer ${token}`}:{})},...(body?{body:JSON.stringify(body)}:{})});const result=await response.json();assert.equal(response.status,expected,`${method} ${path}: ${JSON.stringify(result)}`);return result}
await call('/auth/register','POST',{email,password,display_name:'Phase 2.6 disposable fixture',timezone:'Asia/Kolkata'},202)
execFileSync('docker',['compose','-f','deploy/local/compose.yaml','exec','-T','mongo','mongosh','--quiet','atethat_dev','--eval',`db.users.updateOne({email_normalized:${JSON.stringify(email)}},{$set:{email_verified_at:new Date(),status:'ACTIVE'}})`],{stdio:'pipe'})
token=(await call('/auth/login','POST',{email,password})).access_token
const today=calendarDay('Asia/Kolkata'),tomorrow=new Date(Date.parse(today)+86400000).toISOString().slice(0,10),yesterday=new Date(Date.parse(today)-86400000).toISOString().slice(0,10)
const item=await call('/health-items','POST',{type:'TASK',name:'Brush teeth — UI fixture',instructions:'Synthetic local task',notes:'',start_date:yesterday,schedule:{timezone:'Asia/Kolkata',effective_from_date:yesterday,recurrence:{frequency:'DAILY',interval:1,weekdays:[],local_times:['08:00','09:00','10:00','11:00','23:59']}}},201)
const list=day=>call('/health-items/occurrences?'+new URLSearchParams({...occurrenceRange(day,day,'Asia/Kolkata'),limit:'100'}))
let rows=(await list(today)).items.filter(row=>row.health_item_id===item.id)
assert.equal(rows.length,5)
const done=rows.find(row=>row.scheduled_local.time==='08:00'),skip=rows.find(row=>row.scheduled_local.time==='09:00'),move=rows.find(row=>row.scheduled_local.time==='10:00')
await call(`/health-items/occurrences/${done.occurrence_id}`,'PUT',{status:'TAKEN',completed_at:new Date().toISOString(),expected_event_version:0})
await call(`/health-items/occurrences/${skip.occurrence_id}`,'PUT',{status:'SKIPPED',expected_event_version:0})
const destination=tomorrow+'T14:00:00+05:30'
const moved=await call(`/health-items/occurrences/${move.occurrence_id}`,'PUT',{status:'RESCHEDULED',rescheduled_to:destination,expected_event_version:0})
assert.equal(moved.status,'SCHEDULED');assert.equal(moved.scheduled_local.date,tomorrow);assert.equal(moved.scheduled_local.time,'14:00');assert.equal(moved.original_scheduled_at,move.scheduled_at)
assert(!(await list(today)).items.some(row=>row.occurrence_id===move.occurrence_id))
assert((await list(tomorrow)).items.some(row=>row.occurrence_id===move.occurrence_id&&row.status==='SCHEDULED'))
await call(`/health-items/occurrences/${move.occurrence_id}`,'PUT',{status:'SKIPPED',expected_event_version:0},409)
await call(`/health-items/occurrences/${done.occurrence_id}`,'PUT',{status:'RESCHEDULED',rescheduled_to:destination,expected_event_version:1},422)
console.log(`PASS: task completion/skip, effective-date rescheduling, range removal/inclusion, unchanged identity, stale rejection and done protection. Browser fixture: ${email}; today ${today}, tomorrow ${tomorrow}.`)
