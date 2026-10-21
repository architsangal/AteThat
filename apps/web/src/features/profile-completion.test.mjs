import assert from 'node:assert/strict'
import { test } from 'node:test'
import { missingProfileDetails, missingPlanningDetails, profileAttentionHref } from './profile-completion.ts'

test('counter, list and deep-link identify the exact same missing field', () => {
  const fields = missingProfileDetails({date_of_birth:'2000-01-01',height:{},weight:{},activity_level:'MODERATE'})
  assert.deepEqual(fields.map(f=>f.key),['sex'])
  assert.equal(profileAttentionHref(fields),'/app/settings#profile-field-sex')
  assert(fields[0].hint.includes('Prefer not to provide'))
})
test('all defaults stay identifiable until the profile choice is saved', () => {
  const fields=missingProfileDetails({})
  assert.equal(fields.length,5)
  assert.equal(new Set(fields.map(f=>f.fieldId)).size,5)
  assert.equal(profileAttentionHref(fields),'/app/settings#profile-attention')
  assert(fields.find(f=>f.key==='activity_level').hint.includes('Moderate'))
  assert(fields.find(f=>f.key==='weight').hint.includes('70.4375'))
})
test('explicit decline is a completed optional answer and planning agrees with sidebar', () => {
  const profile={date_of_birth:'2000-01-01',height:{},weight:{},activity_level:'MODERATE',sex:'DECLINED'}
  assert.deepEqual(missingProfileDetails(profile),[])
  assert.deepEqual(missingPlanningDetails({inputs:{activity:'MODERATE',sex:'DECLINED'},missing_inputs:['Equation-specific sex (optional; not inferred)']}),[])
  assert.deepEqual(missingPlanningDetails({inputs:{activity:'MODERATE',sex:'',assumed_activity:true},missing_inputs:['Equation-specific sex (optional; not inferred)']}).map(f=>f.key),['activity_level','sex'])
})
