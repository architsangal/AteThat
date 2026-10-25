import assert from 'node:assert/strict'
import {test} from 'node:test'
import {build} from 'esbuild'
import {createRequire} from 'node:module'
import {readFile} from 'node:fs/promises'
import React from 'react'
import {renderToStaticMarkup} from 'react-dom/server'
const output=await build({entryPoints:[new URL('./wellness/EnergyWater.tsx',import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'}})
const module={exports:{}};new Function('module','exports','require',output.outputFiles[0].text)(module,module.exports,createRequire(import.meta.url))
const {ConnectedTrend,missingDeficitReason}=module.exports
test('connected trends bridge missing days without inventing zero values or moving dates',()=>{
 const html=renderToStaticMarkup(React.createElement(ConnectedTrend,{dates:['2026-10-20','2026-10-21','2026-10-22','2026-10-23'],values:[100,null,-200,0],title:'Deficit',unit:'kcal'}))
 assert.equal((html.match(/class="chart-line"/g)||[]).length,2)
 assert.equal((html.match(/<circle/g)||[]).length,3)
 assert(html.includes('2026-10-22: -200 kcal'));assert(html.includes('class="chart-zero"'));assert(!html.includes('NaN'))
 assert(!html.includes('2026-10-21:'));assert(html.includes('2026-10-23: 0 kcal'))
})
test('empty, single-day and water-goal trends remain finite and accessible',()=>{
 for(const values of [[null],[0],[500]]){const html=renderToStaticMarkup(React.createElement(ConnectedTrend,{dates:['2026-10-25'],values,unit:'ml',title:'Water',goal:2000}));assert(!html.includes('NaN'));assert(html.includes('role="region"'));assert(html.includes('Current water goal')||html.includes('current water goal'));assert(html.includes('chart-goal'))}
})
test('missing food, baseline and workout calories have distinct explanations',()=>{
 const row={maintenance_kcal:'',food_kcal:'',workouts_without_calories:0};assert.match(missingDeficitReason(row),/Save maintenance/)
 row.maintenance_kcal='2000';assert.match(missingDeficitReason(row),/Missing meals/)
 row.food_kcal='1800';row.workouts_without_calories=1;assert.match(missingDeficitReason(row),/Add calories to 1 workout/)
})
test('analysis keeps deficit above Daily rhythm and water immediately below it',async()=>{
 const source=await readFile(new URL('./nutrition/NutritionAnalysisPage.tsx',import.meta.url),'utf8')
 const deficit=source.indexOf('<DeficitAnalysis'),rhythm=source.indexOf('<h2>Daily rhythm'),water=source.indexOf('<WaterAnalysis'),health=source.indexOf('<HealthItemAnalysis')
 assert(deficit<rhythm&&rhythm<water&&water<health)
})

const nutritionOutput=await build({entryPoints:[new URL('./nutrition/NutritionAnalysisPage.tsx',import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'}})
const nutritionModule={exports:{}};new Function('module','exports','require',nutritionOutput.outputFiles[0].text)(nutritionModule,nutritionModule.exports,createRequire(import.meta.url))
test('Daily rhythm connects two recorded points directly across a missing day',()=>{
 const days=[{tracking_day:'2026-10-20',totals:{ENERGY:{status:'KNOWN',amount:'3',unit:'kcal'}}},{tracking_day:'2026-10-22',totals:{ENERGY:{status:'KNOWN',amount:'6',unit:'kcal'}}}]
 const html=renderToStaticMarkup(React.createElement(nutritionModule.exports.Trend,{days,from:'2026-10-20',to:'2026-10-22',code:'ENERGY'}))
 assert.equal((html.match(/class="chart-line"/g)||[]).length,1)
 assert.equal((html.match(/<circle/g)||[]).length,2)
 assert.match(html,/x1="34" y1="97.5" x2="658" y2="30" class="chart-line"/)
 assert(!html.includes('2026-10-21:'));assert(html.includes('not counted as zero'))
})
test('water and deficit connect 3 directly to 6 without a synthetic zero dot',()=>{
 const html=renderToStaticMarkup(React.createElement(ConnectedTrend,{dates:['2026-10-20','2026-10-21','2026-10-22'],values:[3,null,6],title:'Water',unit:'ml'}))
 assert.equal((html.match(/class="chart-line"/g)||[]).length,1)
 assert.equal((html.match(/<circle/g)||[]).length,2)
 assert(html.includes('2026-10-20: 3 ml'));assert(html.includes('2026-10-22: 6 ml'));assert(!html.includes('2026-10-21:'))
})
