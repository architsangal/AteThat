import assert from 'node:assert/strict'
import { test } from 'node:test'
import { validTarget, calorieDifference, exceedsCalorieTolerance } from './target-math.ts'
test('target decimal validation never interprets blank as zero', () => {
  for (const value of ['', '-1', 'NaN', 'Infinity', '1e3', '01', ' 1 ']) assert.equal(validTarget(value), false)
  assert.equal(validTarget('0'), true)
  assert.equal(validTarget('0.000', true), false)
  assert.equal(validTarget('0.001', true), true)
})
test('calorie mismatch preserves decimal precision and one-kcal inclusive tolerance', () => {
  assert.equal(calorieDifference('2000.0000000000000001','2000'), '0.0000000000000001')
  assert.equal(calorieDifference('3140','2000'), '1140')
  assert.equal(exceedsCalorieTolerance('2001','2000'), false)
  assert.equal(exceedsCalorieTolerance('2001.0000000000000001','2000'), true)
  assert.equal(calorieDifference('1.20','1.20'), '0')
})
