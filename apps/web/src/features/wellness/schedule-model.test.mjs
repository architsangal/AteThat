import assert from 'node:assert/strict'
import { test } from 'node:test'
import { scheduleTimesText } from './schedule-model.ts'

test('paused schedule times can be null, absent, or empty without crashing', () => {
  for (const times of [null, undefined, []]) assert.equal(scheduleTimesText(times), '08:00')
  assert.equal(scheduleTimesText(['08:00', '20:00']), '08:00, 20:00')
})
