import assert from 'node:assert/strict'
import { test } from 'node:test'
import { rankFoods } from './food-search.ts'

test('ordinary reference banana appears ahead of a context-free packaged match', () => {
  const packaged = { name: 'BANANA', data_type: 'Branded', food_reference: { provider_food_id: '1' } }
  const raw = { name: 'Banana, raw', data_type: 'Survey (FNDDS)', food_reference: { provider_food_id: '2' } }
  const input = [packaged, raw]
  assert.deepEqual(rankFoods(input, 'banana'), [raw, packaged])
  assert.deepEqual(input, [packaged, raw], 'ranking must not mutate the response held by React')
})

test('pagination removes repeat IDs but preserves distinct source records with the same name', () => {
  const food = id => ({ name: 'BANANA', data_type: 'Branded', food_reference: { provider_food_id: id } })
  assert.deepEqual(rankFoods([food('1'), food('2'), food('1')], 'banana').map(food => food.food_reference.provider_food_id), ['1', '2'])
})

test('an explicit cooked query outranks an unrelated raw reference', () => {
  const raw = { name: 'Rice, raw', food_reference: { provider_food_id: '1' } }
  const cooked = { name: 'Rice, cooked', food_reference: { provider_food_id: '2' } }
  assert.equal(rankFoods([raw, cooked], 'rice cooked')[0], cooked)
})
