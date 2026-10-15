// Creates a synthetic account in the local-only development stack, then checks
// the REST recipe workflow. Never point this fixture setup at production.
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'

const base = 'http://localhost:18080/api/v1'
const email = 'phase2-local-test@example.invalid'
const password = 'Local-only-phase2-test-2026!'
let token = ''
async function request(path, method = 'GET', body, headers = {}) {
  const response = await fetch(base + path, { method, headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...headers }, ...(body ? { body: JSON.stringify(body) } : {}) })
  const result = response.status === 204 ? undefined : await response.json()
  if (!response.ok) throw new Error(`${method} ${path}: ${response.status} ${JSON.stringify(result)}`)
  return result
}
await request('/auth/register', 'POST', { email, password, display_name: 'Phase 2 Local Test', timezone: 'Asia/Kolkata' })
execFileSync('docker', ['compose', '-f', 'deploy/local/compose.yaml', 'exec', '-T', 'mongo', 'mongosh', '--quiet', 'atethat_dev', '--eval', `
db.users.updateOne({email_normalized:'${email}'}, {$set:{email_verified_at:new Date(),status:'ACTIVE'}});
const fixture=db.users.findOne({email_normalized:'${email}'});
// Normalize only this disposable account's fixtures from earlier local iterations.
db.wellness_records.updateMany({user_id:fixture._id,amount:{$type:'string'}},[{$set:{amount:{$toDecimal:'$amount'},estimated_kcal:{$cond:[{$eq:['$estimated_kcal','']},'$$REMOVE',{$toDecimal:'$estimated_kcal'}]}}}]);
db.wellness_settings.updateMany({user_id:fixture._id,water_target_ml:{$type:'string'}},[{$set:{water_target_ml:{$cond:[{$eq:['$water_target_ml','']},'$$REMOVE',{$toDecimal:'$water_target_ml'}]}}}]);
`], { stdio: 'pipe' })
token = (await request('/auth/login', 'POST', { email, password })).access_token
assert.ok(token)
if (process.argv.includes('--provider')) {
  const results = await request('/foods/search?q=rice&limit=1')
  assert.ok(results.items.length > 0)
  assert.equal(results.items[0].nutrients.IRON.unit, 'mg')
  assert.equal(results.items[0].nutrients.VITAMIN_D.unit, 'mcg')
  console.log('PASS: USDA food search exposes expanded nutrient mappings through the local API.')
}
if (process.argv.includes('--seed-only')) {
  console.log('Local fixture ready: phase2-local-test@example.invalid (password is the synthetic constant in this script).')
  process.exit(0)
}
const food = await request('/foods/custom', 'POST', { name: 'Local test oats', description: 'Synthetic fixture', food_state: 'RAW', base_quantity: { amount: '100', unit: 'g' }, nutrients: { ENERGY: { status: 'KNOWN', amount: '200', unit: 'kcal' }, PROTEIN: { status: 'KNOWN', amount: '10', unit: 'g' }, IRON: { status: 'UNKNOWN', unit: 'mg' }, CALCIUM: { status: 'KNOWN', amount: '0', unit: 'mg' } } })
const component = { position: 0, food_reference: { type: 'CUSTOM', custom_food_id: food.id }, quantity: { amount: '300', unit: 'g' }, food_state: 'RAW' }
const recipe = await request('/meals', 'POST', { name: 'Local test porridge', recipe_servings: '4', preparation: 'Synthetic fixture', components: [component] })
const savedMeal = await request('/meals','POST',{name:'QA saved breakfast',components:[component]})
const savedMealReview = await request(`/meals/${savedMeal.id}/preview`,'POST',{logged_at:new Date().toISOString(),meal_category:'BREAKFAST',label:savedMeal.name})
assert.equal(savedMealReview.group.items[0].quantity.amount,'300')
const personalServing=await request('/servings','POST',{label:'QA bowl (100 g)',food_reference:component.food_reference,quantity:{amount:'100',unit:'g'},food_state:'RAW'})
assert.equal(personalServing.food_reference.custom_food_id,food.id)
assert.equal((await request('/servings?limit=50')).items.some(value=>value.id===personalServing.id),true)
const portion = await request(`/meals/${recipe.id}/preview`, 'POST', { logged_at: new Date().toISOString(), meal_category: 'BREAKFAST', label: recipe.name, servings: '1.5' })
assert.equal(portion.group.items[0].quantity.amount, '112.5')
assert.equal(portion.group.aggregate_final_nutrition.ENERGY.amount, '225')
assert.equal(portion.group.aggregate_final_nutrition.IRON.status, 'UNKNOWN')
assert.equal(portion.group.aggregate_final_nutrition.CALCIUM.amount, '0')
const key = crypto.randomUUID()
const confirmed = await request('/food-log-groups/confirm', 'POST', { preview_token: portion.preview_token }, { 'Idempotency-Key': key })
const replay = await request('/food-log-groups/confirm', 'POST', { preview_token: portion.preview_token }, { 'Idempotency-Key': key })
assert.equal(replay.id, confirmed.id)
await request(`/meals/${recipe.id}`, 'PATCH', { version: recipe.version, name: recipe.name, recipe_servings: '8', preparation: 'Updated fixture', components: [component] })
const history = await request(`/food-log-groups/${confirmed.id}`)
assert.equal(history.aggregate_final_nutrition.ENERGY.amount, '225')
if (process.argv.includes('--trends')) {
  for (const [offset, amount] of [[1, '150'], [2, '250'], [4, '80'], [5, '120'], [6, '220']]) {
    const loggedAt = new Date(Date.now() - offset * 86400000).toISOString()
    const preview = await request('/food-log-groups/preview', 'POST', { logged_at: loggedAt, meal_category: 'BREAKFAST', label: 'Synthetic trend fixture', items: [{ position: 0, food_reference: component.food_reference, quantity: { amount, unit: 'g' }, food_state: 'RAW' }] })
    await request('/food-log-groups/confirm', 'POST', { preview_token: preview.preview_token }, { 'Idempotency-Key': crypto.randomUUID() })
  }
}
console.log('PASS: REST recipe yield, fractional portions, unknown/zero values, idempotent confirmation, and historical snapshot preservation.')
const override = await request(`/meals/${recipe.id}/preview`, 'POST', { logged_at:new Date().toISOString(), meal_category:'LUNCH', label:recipe.name, servings:'1', overrides:[{position:0,nutrients:{ENERGY:{status:'KNOWN',amount:'80',unit:'kcal'}}}] })
assert.equal(override.group.aggregate_final_nutrition.ENERGY.amount,'80')
assert.equal(override.group.items[0].provider_nutrition.ENERGY.amount,'75')
const overrideLog=await request('/food-log-groups/confirm','POST',{preview_token:override.preview_token},{'Idempotency-Key':crypto.randomUUID()})
const overrideHistory=await request(`/food-log-groups/${overrideLog.id}`)
assert.equal(overrideHistory.aggregate_final_nutrition.ENERGY.amount,'80')
assert.equal(overrideHistory.items[0].provider_nutrition.ENERGY.amount,'75')
const now=new Date().toISOString()
const water=await request('/wellness/records','POST',{kind:'WATER',amount:'0.75',unit:'L',logged_at:now})
assert.equal(water.amount,'750');assert.equal(water.unit,'ml')
const changed=await request(`/wellness/records/${water.id}`,'PATCH',{version:water.version,kind:water.kind,unit:water.unit,logged_at:water.logged_at,amount:'800'})
assert.equal(changed.version,2);assert.equal(changed.tracking_day,water.tracking_day)
const weight=await request('/wellness/records','POST',{kind:'WEIGHT',amount:'150',unit:'lb',logged_at:now,notes:'Synthetic local measurement'})
assert.equal(weight.amount,'68.038856')
await request('/wellness/records','POST',{kind:'ACTIVITY',amount:'25',unit:'min',logged_at:now,name:'Local test walk',intensity:'LIGHT',estimated_kcal:'80'})
const entries=await request(`/wellness/records?from_tracking_day=${water.tracking_day}&to_tracking_day=${water.tracking_day}&limit=50`)
assert.ok(entries.items.some(entry=>entry.id===water.id))
const settings=await request('/wellness/settings')
const goal=await request('/wellness/settings','PUT',{version:settings.version,water_target_ml:'2000'})
assert.equal(goal.water_target_ml,'2000')
await request(`/wellness/records/${water.id}?version=${changed.version}`,'DELETE')
const tomorrow=new Date(Date.now()+86400000).toISOString().slice(0,10)
const start=new Date(Date.now()-86400000).toISOString().slice(0,10)
const item=await request('/health-items','POST',{type:'SUPPLEMENT',name:'Local schedule fixture',start_date:start,dosage:{amount_text:'1',unit_text:'tablet'},schedule:{timezone:'UTC',effective_from_date:start,effective_to_date:'',recurrence:{frequency:'MONTHLY',interval:1,month_days:[1,15,28],weekdays:[],local_times:['08:00','20:00']}}})
const paused=await request(`/health-items/${item.id}`,'PATCH',{version:item.version,type:item.type,name:item.name,dosage:item.dosage,start_date:item.start_date,schedule:{timezone:'UTC',effective_from_date:tomorrow,effective_to_date:'',recurrence:{frequency:'PAUSED',interval:1,weekdays:[],local_times:[],month_days:[]}}})
assert.equal(paused.schedule.recurrence.frequency,'PAUSED')
const schedules=await request(`/health-items/${item.id}/schedule-history?limit=1`)
assert.equal(schedules.items[0].version,2);assert.equal(schedules.next_before_version,2)
console.log('PASS: REST saved meals, owned serving mappings, recipe overrides, wellness conversions/CRUD/goals, monthly schedule, versioned pause, and paginated schedule history.')
console.log('Synthetic fixtures remain in local development for browser testing.')
