import assert from 'node:assert/strict'
import { test } from 'node:test'
import { known, nutrientStats, rangeDays, targetForDay } from './model.ts'

test('unknown values and blank amounts never become zero', () => {
  assert.equal(known({ status: 'UNKNOWN', unit: 'g' }), null)
  assert.equal(known({ status: 'KNOWN', amount: '', unit: 'g' }), null)
  assert.equal(known({ status: 'KNOWN', amount: '0', unit: 'g' }), 0)
})
test('averages exclude unknown days and incompatible units', () => {
  const rows = ['10', undefined, '20'].map((amount, i) => ({ totals: { IRON: { status: amount ? 'KNOWN' : 'UNKNOWN', amount, unit: i === 2 ? 'mcg' : 'mg' } } }))
  const result = nutrientStats(rows, 'IRON')
  assert.equal(result.average, 10)
  assert.equal(result.count, 1)
  assert.equal(result.complete, false)
})
test('date range crosses leap day without timezone drift', () => {
  assert.deepEqual(rangeDays('2024-02-28', '2024-03-01'), ['2024-02-28', '2024-02-29', '2024-03-01'])
})
test('historical targets choose an effective accepted revision, never a proposal or future value', () => {
 const version=(revision,status,date,value)=>({revision,status,effective_from_tracking_day:date,targets:{ENERGY:{active_value:value,unit:'kcal'}}})
 const history=[version(1,'SUPERSEDED','2026-10-01','2000'),version(2,'SUPERSEDED','2026-10-05','2100'),version(3,'SUPERSEDED','2026-10-05','2200'),version(4,'ACTIVE','2026-10-20','2300'),version(5,'PROPOSED','2026-10-06','3000')]
 assert.equal(targetForDay(history,'2026-09-30','ENERGY'),undefined)
 assert.equal(targetForDay(history,'2026-10-03','ENERGY').active_value,'2000')
 assert.equal(targetForDay(history,'2026-10-10','ENERGY').active_value,'2200')
 assert.equal(targetForDay(history,'2026-10-21','ENERGY').active_value,'2300')
 assert.equal(targetForDay(history,'2026-10-21','IRON'),undefined)
})
