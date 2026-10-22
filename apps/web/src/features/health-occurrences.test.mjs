import assert from 'node:assert/strict'
import {test} from 'node:test'
import {calendarDay,occurrenceMatches,occurrenceRange,occurrenceTabs} from './health-occurrences.ts'
test('four tabs separate completed and active events while grouping deliberate skips under missed',()=>{
 assert.deepEqual(occurrenceTabs.map(([,name])=>name),['Active','Done','Missed','All'])
 for(const status of ['SCHEDULED','TAKEN','MISSED','SKIPPED'])assert(occurrenceMatches({status},'ALL'))
 assert(occurrenceMatches({status:'SKIPPED'},'MISSED'))
 assert(!occurrenceMatches({status:'TAKEN'},'ACTIVE'))
 assert(!occurrenceMatches({status:'MISSED'},'ACTIVE'))
 assert(occurrenceMatches({status:'SCHEDULED'},'ACTIVE'))
})
test('today and inclusive ranges follow the account calendar, not UTC or the nutrition reset',()=>{
 assert.equal(calendarDay('Asia/Kolkata',new Date('2026-10-21T20:00:00Z')),'2026-10-22')
 assert.deepEqual(occurrenceRange('2026-10-22','2026-10-22','Asia/Kolkata'),{from:'2026-10-21T18:30:00.000Z',to:'2026-10-22T18:30:00.000Z'})
 assert.deepEqual(occurrenceRange('2026-10-21','2026-10-23','UTC'),{from:'2026-10-21T00:00:00.000Z',to:'2026-10-24T00:00:00.000Z'})
})
test('DST days are not forced to 24 hours and invalid/overlong ranges fail',()=>{
 for(const [day,hours]of [['2026-03-08',23],['2026-11-01',25]]){const range=occurrenceRange(day,day,'America/New_York');assert.equal((Date.parse(range.to)-Date.parse(range.from))/3600000,hours)}
 assert.throws(()=>occurrenceRange('2026-02-31','2026-03-04','UTC'),/valid start/)
 assert.throws(()=>occurrenceRange('2026-10-22','2026-10-21','UTC'),/before/)
 assert.throws(()=>occurrenceRange('2026-01-01','2026-02-01','UTC'),/31/)
})
