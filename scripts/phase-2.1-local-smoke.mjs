// Disposable local fixtures only. Uses cached synthetic provider payloads so
// quota/concurrency tests neither depend on USDA nor consume its upstream budget.
import assert from 'node:assert/strict'
import {createHash} from 'node:crypto'
import {execFileSync} from 'node:child_process'
const base='http://localhost:18080/api/v1'
const suffix=Date.now().toString()
const email=`phase21-quota-${suffix}@example.invalid`
const other=`phase21-other-${suffix}@example.invalid`
const password='Local-only-phase21-test-2026!'
let token=''
async function call(path,method='GET',body,expected=200) {
 const response=await fetch(base+path,{method,headers:{'Content-Type':'application/json',...(token?{Authorization:`Bearer ${token}`}:{})},...(body?{body:JSON.stringify(body)}:{})})
 const result=await response.json()
 assert.equal(response.status,expected,`${method} ${path}: ${JSON.stringify(result)}`)
 return result
}
function mongo(code) { return execFileSync('docker',['compose','-f','deploy/local/compose.yaml','exec','-T','mongo','mongosh','--quiet','atethat_dev','--eval',code],{encoding:'utf8'}) }
for(const address of [email,other]) await call('/auth/register','POST',{email:address,password,display_name:'Disposable quota fixture',timezone:'Asia/Kolkata'},202)
mongo(`db.users.updateMany({email_normalized:{$in:${JSON.stringify([email,other])}}},{$set:{email_verified_at:new Date(),status:'ACTIVE'}})`)
await call('/foods/search?source=STATUS','GET',undefined,401)
token=(await call('/auth/login','POST',{email,password})).access_token
const privateFood=await call('/foods/custom','POST',{name:`Private fixture ${suffix}`,food_state:'UNKNOWN',base_quantity:{amount:'100',unit:'g'},nutrients:{ENERGY:{status:'KNOWN',amount:'10',unit:'kcal'}}},201)
let profile=await call('/profile');assert.equal(profile.onboarding_complete,false)
profile=await call('/profile','PATCH',{version:profile.version,onboarding_complete:true})
assert.equal(profile.onboarding_complete,true)
token=(await call('/auth/login','POST',{email,password})).access_token
assert.equal((await call('/profile')).onboarding_complete,true)
console.log('PASS: welcome completion/skip is account-persisted across sign-ins.')
const query=`qa catalogue ${suffix}`
const badQuery=`qa malformed ${suffix}`
const emptyQuery=`qa empty ${suffix}`
const cacheKey=q=>'search:'+createHash('sha256').update(q.toLowerCase()+'||20').digest('hex').slice(0,24)+':1'
const fixture={totalHits:2,foods:[{fdcId:999999001,description:query+' cooked test oats',dataType:'Foundation',foodNutrients:[{nutrientId:1008,unitName:'KCAL',value:200}]},{fdcId:999999002,description:query+' cooked test oats',dataType:'Foundation',foodNutrients:[]}]}
for(const [q,payload] of [[query,fixture],[badQuery,{totalHits:'invalid'}],[emptyQuery,{totalHits:0,foods:[]}]]) {
 const b64=Buffer.from(JSON.stringify(payload)).toString('base64')
 mongo(`db.usda_food_cache.updateOne({cache_key:${JSON.stringify(cacheKey(q))}},{$set:{payload:Binary.createFromBase64('${b64}',0),expires_at:new Date(Date.now()+3600000)}},{upsert:true})`)
}
let status=(await call('/foods/search?source=STATUS')).usage
assert.equal(status.usda_remaining,10);assert.equal(status.local_searches_today,0)
await call('/foods/search?source=USDA&q='+encodeURIComponent(badQuery),'GET',undefined,503)
assert.equal((await call('/foods/search?source=STATUS')).usage.usda_remaining,10)
console.log('PASS: provider parse failures refund reserved quota.')
const discovery=await call('/foods/search?source=USDA&q='+encodeURIComponent(query))
assert.equal(discovery.items.length,2);assert.equal(discovery.usage.usda_remaining,9)
const local=await call('/foods/search?source=LOCAL&q='+encodeURIComponent(query))
assert.equal(local.items.length,2);assert.equal(local.usage.matched_count,2);assert.equal(local.usage.usda_remaining,9);assert.equal(local.usage.local_searches_today,1)
assert.equal(new Set(local.items.map(item=>item.food_reference.provider_food_id)).size,2)
const empty=await call('/foods/search?source=USDA&q='+encodeURIComponent(emptyQuery));assert.equal(empty.usage.usda_remaining,8)
const responses=await Promise.all(Array.from({length:12},()=>fetch(base+'/foods/search?source=USDA&q='+encodeURIComponent(query),{headers:{Authorization:`Bearer ${token}`}})))
assert.equal(responses.filter(r=>r.status===200).length,8)
assert.equal(responses.filter(r=>r.status===429).length,4)
status=(await call('/foods/search?source=STATUS')).usage;assert.equal(status.usda_remaining,0)
const unlimited=await call('/foods/search?source=LOCAL&q='+encodeURIComponent(query));assert.equal(unlimited.items.length,2);assert.equal(unlimited.usage.usda_remaining,0)
assert.equal((await call('/foods/search?source=STATUS')).usage.catalogue_count,status.catalogue_count)
console.log('PASS: local imports deduplicate by FDC ID, keep distinct same-name foods, count matches, and remain searchable after quota exhaustion. Concurrent USDA requests cannot exceed 10; empty results count.')
token=(await call('/auth/login','POST',{email:other,password})).access_token
status=(await call('/foods/search?source=STATUS')).usage;assert.equal(status.usda_remaining,10);assert.equal(status.local_searches_today,0)
assert.equal((await call('/foods/search?source=LOCAL&q='+encodeURIComponent(query))).items.length,2)
await call(`/foods/custom/${privateFood.id}`,'GET',undefined,404)
assert.equal((await call('/foods/search?source=LOCAL&q='+encodeURIComponent(`Private fixture ${suffix}`))).items.length,0)
console.log('PASS: usage requires authentication; local public search does not expose another account’s private foods.')
// Old-day usage cannot reduce today's allowance; all changes are scoped to this disposable account.
mongo(`const u=db.users.findOne({email_normalized:${JSON.stringify(other)}});db.food_search_usage.updateOne({_id:u._id.toHexString()+':2000-01-01'},{$set:{usda:10,local:99,expires_at:new Date(Date.now()+3600000)}},{upsert:true})`)
status=(await call('/foods/search?source=STATUS')).usage;assert.equal(status.usda_remaining,10);assert.equal(status.local_searches_today,2)
assert.equal(new Date(status.resets_at).getUTCHours(),0)
console.log('PASS: quotas and local counters are isolated per account, old-day usage is excluded, reset is midnight UTC, and the shared catalogue is visible to the second account.')
