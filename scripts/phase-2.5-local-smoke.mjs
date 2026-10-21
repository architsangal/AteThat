// Synthetic local account only. Never point this script at production.
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
const base = 'http://localhost:18080/api/v1'
const email = `phase25-${Date.now()}@example.invalid`
const password = 'Local-only-phase25-test-2026!'
let token = ''
async function call(path, method = 'GET', body, expected = 200) {
  const response = await fetch(base + path, { method, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}) }, ...(body ? { body: JSON.stringify(body) } : {}) })
  const result = await response.json(); assert.equal(response.status, expected, `${method} ${path}: ${JSON.stringify(result)}`); return result
}
await call('/auth/register','POST',{email,password,display_name:'Phase 2.5 disposable fixture',timezone:'Asia/Kolkata'},202)
execFileSync('docker',['compose','-f','deploy/local/compose.yaml','exec','-T','mongo','mongosh','--quiet','atethat_dev','--eval',`db.users.updateOne({email_normalized:${JSON.stringify(email)}},{$set:{email_verified_at:new Date(),status:'ACTIVE'}})`],{stdio:'pipe'})
token = (await call('/auth/login','POST',{email,password})).access_token
let profile = await call('/profile')
profile = await call('/profile','PATCH',{version:profile.version,onboarding_complete:true})
if (process.env.PROPOSAL_FIXTURE === '1') {
 profile = await call('/profile','PATCH',{version:profile.version,date_of_birth:'2000-01-01',sex:'MALE'})
 const proposal = await call('/nutrition/targets/proposal','POST',{})
 assert.equal(proposal.status,'PROPOSED')
 assert.equal((await call('/nutrition/targets')).active,null)
 console.log(`Initial-proposal browser fixture: ${email}`)
 process.exit(0)
}
const plan = await call('/nutrition/targets/planning')
assert.equal(plan.inputs.height_cm,'175'); assert.equal(plan.inputs.weight_kg,'70.4375'); assert.equal(plan.inputs.assumed_weight,true); assert.equal(plan.personalized_enabled,false)
const changes = Object.fromEntries(Object.entries({ENERGY:'3140.0',PROTEIN:'160',CARBOHYDRATE:'400',FAT:'100',FIBER:'28.123456789'}).map(([key,active_value])=>[key,{active_value}]))
let active = await call('/nutrition/targets/custom','POST',{expected_active_id:'',expected_revision:0,profile_version:profile.version,changes,energy_provenance:'MACRO_DERIVED'},201)
assert.equal(active.targets.ENERGY.active_value,'3140.0')
active = await call('/nutrition/targets/custom','POST',{expected_active_id:active.id,expected_revision:active.revision,profile_version:profile.version,changes:{PROTEIN:{active_value:'150'}}},201)
assert.equal(active.targets.ENERGY.active_value,'3140.0'); assert.equal(active.targets.FIBER.active_value,'28.123456789')
const calories = await call('/nutrition/targets/planning','POST',{macros:{PROTEIN:'160',CARBOHYDRATE:'400',FAT:'100'}})
assert.equal(calories.macro_calories,'3140')
const day = await call('/tracking-day/current')
await call('/wellness/records','POST',{kind:'WEIGHT',amount:'85',unit:'kg',logged_at:new Date().toISOString()},201)
const afterWeight = await call('/nutrition/targets/planning')
assert.equal(afterWeight.inputs.weight_kg,'70.4375'); assert.equal(afterWeight.inputs.assumed_weight,true)
const summary = await call(`/text-summary?${new URLSearchParams({from:day.tracking_day,to:day.tracking_day,mode:'DAY',nutrient:'ALL'})}`)
assert(summary.text.includes('No recorded nutrition days')); assert(!summary.text.includes(email)); assert(!summary.text.includes('Phase 2.5 disposable fixture'))
profile = await call('/profile','PATCH',{version:profile.version,date_of_birth:'2000-01-01',sex:'MALE'})
const preview = await call('/nutrition/targets/planning')
assert.equal(preview.personalized_enabled,true); assert.equal(preview.protein_rate,'1.2')
assert.equal(preview.inputs.weight_kg,'70.4375'); assert.equal(preview.inputs.assumed_weight,true)
const beforeReset = await call('/nutrition/targets')
assert.equal(beforeReset.active.id,active.id) // Preview has no write side effects.
const resetBody = {expected_active_id:active.id,expected_revision:active.revision,profile_version:profile.version,preview_token:preview.reset_preview_token,adult_scope_confirmed:true}
active = await call('/nutrition/targets/reset','POST',resetBody,201)
for (const [key,value] of Object.entries(preview.calculated_defaults)) {
 assert.equal(active.targets[key].active_value,value)
 assert.equal(active.targets[key].provenance,'CALCULATED_DEFAULT')
}
await call('/nutrition/targets/reset','POST',resetBody,409)
assert((await call('/profile')).weight == null)
const customBody = {expected_active_id:active.id,expected_revision:active.revision,profile_version:profile.version,changes:{PROTEIN:{active_value:'500'}}}
const review = await call('/nutrition/targets/custom/review','POST',customBody)
assert(review.warnings.some(w=>w.nutrient==='PROTEIN')); assert(review.acknowledgement_token)
await call('/nutrition/targets/custom','POST',customBody,409)
active = await call('/nutrition/targets/custom','POST',{...customBody,acknowledgement_token:review.acknowledgement_token},201)
assert.equal(active.targets.PROTEIN.outside_reference,true); assert(active.targets.PROTEIN.acknowledged_at)
assert.equal((await call('/nutrition/targets')).active.targets.PROTEIN.acknowledged_at,active.targets.PROTEIN.acknowledged_at)
console.log(`PASS: defaults, macro refresh, independent edits, precision, summary, calculated reset, stale rejection and signed outside-reference confirmation. Browser fixture: ${email}`)
