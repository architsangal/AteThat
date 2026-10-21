import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isTextTooShort } from './form-validation.ts'

test('correction reason rejects one or two characters without accepting padding', () => {
  for (const value of ['a', 'ab', ' a ', ' ab ', '   ', '\t a \n']) {
    assert.equal(isTextTooShort(value, 3), true, JSON.stringify(value))
  }
})

test('correction reason accepts three meaningful characters or a longer explanation', () => {
  for (const value of ['abc', ' abc ', 'Incorrect amount']) {
    assert.equal(isTextTooShort(value, 3), false, JSON.stringify(value))
  }
})

test('empty values remain the required validator responsibility; unrelated fields are unchanged', () => {
  assert.equal(isTextTooShort('', 3), false)
  assert.equal(isTextTooShort('a', 0), false)
  assert.equal(isTextTooShort('', 0), false)
})
