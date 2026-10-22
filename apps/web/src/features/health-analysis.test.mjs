import assert from 'node:assert/strict'
import {test} from 'node:test'
import {build} from 'esbuild'
import {createRequire} from 'node:module'
import {readFile} from 'node:fs/promises'
import React from 'react'
import {renderToStaticMarkup} from 'react-dom/server'
async function load(file){const built=await build({entryPoints:[new URL(file,import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'}});const mod={exports:{}};new Function('module','exports','require',built.outputFiles[0].text)(mod,mod.exports,createRequire(import.meta.url));return mod.exports}
const {healthTrackingDay,itemDailyHealth,validateAnalysisRange,readAnalysisPages,readHealthAnalysis}=await load('./health-analysis-model.ts')
const {HealthItemChart}=await load('./HealthItemAnalysis.tsx')
const context={timezone:'Asia/Kolkata',daily_reset_time:'04:00'}
const row=(id,item,at,status)=>({occurrence_id:id,health_item_id:item,scheduled_at:at,status})
test('health analysis applies reset in account timezone including before-reset next morning and DST fallback',()=>{
 assert.equal(healthTrackingDay('2026-10-23T03:59:00+05:30',context),'2026-10-22')
 assert.equal(healthTrackingDay('2026-10-23T04:00:00+05:30',context),'2026-10-23')
 const dst={timezone:'America/New_York',daily_reset_time:'01:30'}
 assert.equal(healthTrackingDay('2026-11-01T01:45:00-04:00',dst),'2026-11-01')
 assert.equal(healthTrackingDay('2026-11-01T01:15:00-05:00',dst),'2026-10-31')
})
test('individual charts use item identity not name, deduplicate, exclude future from due percentage, and retain skip detail',()=>{
 const rows=[row('1','a','2026-10-22T08:00:00+05:30','TAKEN'),row('2','a','2026-10-22T09:00:00+05:30','SKIPPED'),row('3','a','2026-10-22T10:00:00+05:30','MISSED'),row('4','a','2026-10-22T23:59:00+05:30','SCHEDULED'),row('5','b','2026-10-22T08:00:00+05:30','TAKEN')]
 const days=itemDailyHealth([...rows,rows[0]],'a','2026-10-22','2026-10-23',context,Date.parse('2026-10-22T12:00:00+05:30'))
 assert.deepEqual(days[0],{day:'2026-10-22',done:1,skipped:1,missed:1,active:1,due:3,dueDone:1})
 assert.deepEqual(days[1],{day:'2026-10-23',done:0,skipped:0,missed:0,active:0,due:0,dueDone:0})
})
test('rescheduled occurrences are bucketed by effective time, never original time',()=>{
 const moved={...row('move','a','2026-10-24T06:00:00+05:30','SCHEDULED'),original_scheduled_at:'2026-10-22T06:00:00+05:30'}
 const days=itemDailyHealth([moved],'a','2026-10-22','2026-10-24',context,Date.parse('2026-10-23T12:00:00Z'))
 assert.equal(days[0].active,0);assert.equal(days[2].active,1)
})
test('analysis supports 366 real dates and rejects invalid or reversed ranges',()=>{
 assert.doesNotThrow(()=>validateAnalysisRange('2024-01-01','2024-12-31'))
 for(const [from,to] of [['2026-02-30','2026-03-01'],['2026-10-23','2026-10-22'],['2024-01-01','2025-01-01']])assert.throws(()=>validateAnalysisRange(from,to))
})
test('pagination is complete and rejects loops, missing cursors, and superseded requests',async()=>{
 let calls=0
 const get=async()=>++calls===1?{items:[1],page:{next_cursor:'next',has_more:true}}:{items:[2],page:{next_cursor:null,has_more:false}}
 assert.deepEqual(await readAnalysisPages(get,'/test',{limit:'1'}),[1,2])
 await assert.rejects(()=>readAnalysisPages(async()=>({items:[],page:{next_cursor:'same'}}),'/test',{}),/complete/)
 await assert.rejects(()=>readAnalysisPages(async()=>({items:[],page:{next_cursor:null,has_more:true}}),'/test',{}),/complete/)
 await assert.rejects(()=>readAnalysisPages(get,'/test',{},()=>false),/replaced/)
})
test('long health ranges are chunked without gaps, include next morning and remain within API limits across DST',async()=>{
 const windows=[]
 await readHealthAnalysis(async path=>{const url=new URL(path,'https://example.invalid');windows.push([Date.parse(url.searchParams.get('from')),Date.parse(url.searchParams.get('to'))]);return{items:[],page:{next_cursor:null}}},{from:'2026-01-01',to:'2026-12-31',revision:1},{timezone:'America/New_York',daily_reset_time:'04:00'})
 assert(windows.length>12)
 windows.forEach(([a,b],i)=>{assert(b>a&&b-a<=32*86400000);if(i)assert.equal(windows[i-1][1],a)})
 assert.equal(new Date(windows.at(-1)[1]).toISOString(),'2027-01-02T05:00:00.000Z')
})
test('charts expose exact daily tables, no-schedule/zero-denominator states and separate status legend',()=>{
 const item={id:'a',name:'Brushing',type:'TASK',active:true}
 const days=itemDailyHealth([],'a','2026-10-22','2026-10-23',context,Date.now())
 const html=renderToStaticMarkup(React.createElement(HealthItemChart,{item,days,percent:false}))
 assert(html.includes('No scheduled occurrences'));assert(html.includes('No due occurrences'));assert(html.includes('Daily values'));assert(html.includes('Skipped by you'));assert(!html.includes('NaN'))
})
test('shared timeline precedes nutrition, health analysis precedes summary, and pending date changes are explicit',async()=>{
 const source=await readFile(new URL('./nutrition/NutritionAnalysisPage.tsx',import.meta.url),'utf8')
 assert(source.indexOf('>Timeline<')<source.indexOf('>Nutrition analysis<'))
 assert(source.indexOf('<HealthItemAnalysis')<source.indexOf('<TextSummary'))
 assert(source.includes("'Explore'"));assert(!source.includes('Explore nutrition'));assert(source.includes('Dates changed.'))
})
test('charts measure their panel and retain pixel-proportional labels rather than stretching the SVG',async()=>{
 const source=await readFile(new URL('./HealthItemAnalysis.tsx',import.meta.url),'utf8')
 assert(source.includes('new ResizeObserver'))
 assert(source.includes('width={width} height={215} viewBox='))
 assert(!source.includes('preserveAspectRatio="none"'))
 assert.equal((source.match(/x=\{x\+barWidth\/2\} y="198" textAnchor="middle"/g)||[]).length,2)
 const css=await readFile(new URL('./health-analysis.css',import.meta.url),'utf8')
 assert(!css.includes('.health-chart-scroll svg { display: block; width: 100%;'))
})
