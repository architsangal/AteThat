import assert from 'node:assert/strict'
import {test} from 'node:test'
import {copySummaryText} from './summary-clipboard.ts'

test('full edited summary is copied without truncation',async()=>{
 const text='Edited report\n'.repeat(2000)
 let copied=''
 const status=await copySummaryText(text,async value=>{copied=value},()=>assert.fail('unexpected fallback'))
 assert.equal(copied,text)
 assert.equal(status,'Full edited summary copied.')
})
test('clipboard denial selects the existing draft and reports manual copy without losing text',async()=>{
 let selected=false
 const status=await copySummaryText('Private draft',async()=>{throw new Error('NotAllowedError')},()=>{selected=true})
 assert(selected)
 assert.match(status,/Clipboard unavailable.*Text selected/)
})
