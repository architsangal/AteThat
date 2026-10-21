import assert from 'node:assert/strict'
import { test } from 'node:test'
import { healthStatusLabel } from './health-model.ts'

test('task completion is displayed as done without changing medication terminology', () => {
  assert.equal(healthStatusLabel('TAKEN', 'TASK'), 'done')
  assert.equal(healthStatusLabel('TAKEN', 'MEDICATION'), 'taken')
  assert.equal(healthStatusLabel('TAKEN', 'SUPPLEMENT'), 'taken')
})

test('tasks retain the existing skipped, missed, rescheduled and upcoming statuses', () => {
  for (const status of ['SKIPPED', 'MISSED', 'RESCHEDULED', 'SCHEDULED']) {
    assert.equal(healthStatusLabel(status, 'TASK'), status.toLowerCase())
  }
})
