import assert from 'node:assert/strict'
import {test} from 'node:test'
import {build} from 'esbuild'
import {createRequire} from 'node:module'
import {readFile} from 'node:fs/promises'
import React from 'react'
import {renderToStaticMarkup} from 'react-dom/server'
const built=await build({entryPoints:[new URL('./ScheduleRepeat.tsx',import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false})
const mod={exports:{}};new Function('module','exports','require',built.outputFiles[0].text)(mod,mod.exports,createRequire(import.meta.url))
const {repeatCadence,RepeatInterval,SchedulePreview}=mod.exports
test('repeat wording distinguishes every day, alternate days and every N days',()=>{
 assert.equal(repeatCadence('DAILY',1),'Every day')
 assert.equal(repeatCadence('DAILY',2),'Every other day')
 assert.equal(repeatCadence('DAILY',3),'Every 3 days')
 assert.equal(repeatCadence('WEEKLY',2),'Every other week')
 assert.equal(repeatCadence('MONTHLY',3),'Every 3 months')
 for(const invalid of [0,NaN,1.5,366])assert.match(repeatCadence('DAILY',invalid),/whole-number/)
})
test('repeat field retains canonical interval validation and singular/plural units',()=>{
 const render=value=>renderToStaticMarkup(React.createElement(RepeatInterval,{frequency:'DAILY',value,onChange:()=>{}}))
 assert(render('1').includes('<span>day</span>'))
 assert(render('2').includes('<span>days</span>'))
 assert(render('2').includes('name="interval"'))
 assert(render('2').includes('min="1" max="365" step="1" required=""'))
 assert(render('2').includes('not how many times each day'))
})
test('preview separates twice on each scheduled day from an alternate-day interval',()=>{
 const html=renderToStaticMarkup(React.createElement(SchedulePreview,{frequency:'DAILY',interval:2,times:['08:00','20:00'],start:'2026-10-23'}))
 assert(html.includes('Every other day'))
 assert(html.includes('Twice on each scheduled day, at 08:00 and 20:00'))
 assert(html.includes('Starting 23 Oct 2026'))
})
test('paused, incomplete and duplicate times never produce a misleading timed preview',()=>{
 const render=(frequency,times)=>renderToStaticMarkup(React.createElement(SchedulePreview,{frequency,interval:1,times,start:'2026-10-23'}))
 assert(!render('PAUSED',['08:00']).includes('Once'))
 for(const times of [['08:'],['08:00','08:00'],['24:00'],[]])assert(render('DAILY',times).includes('Choose complete, different times'))
})
test('create/edit share repeat controls and the interval is not hidden under advanced options',async()=>{
 const app=await readFile(new URL('../App.tsx',import.meta.url),'utf8')
 const editor=await readFile(new URL('./wellness/ScheduleEditor.tsx',import.meta.url),'utf8')
 assert(app.includes('<RepeatInterval frequency={frequency} value={repeatEvery}'))
 assert(app.includes('Optional end date'))
 assert(!app.includes('<label>Repeat every {frequency'))
 assert(editor.includes('<RepeatInterval'));assert(editor.includes('<SchedulePreview'));assert(editor.includes('repeatCadence(schedule.recurrence.frequency'))
})
test('all app scrollbars share theme styling without hiding scrolling or shrinking every element',async()=>{
 const css=await readFile(new URL('../scrollbars.css',import.meta.url),'utf8')
 assert(css.includes('scrollbar-gutter: stable'))
 assert(css.includes('scrollbar-color: var(--scrollbar-thumb) transparent'))
 assert(css.includes('*::-webkit-scrollbar-thumb'))
 assert(css.includes('width: 8px; height: 8px'))
 assert(css.includes('@media (forced-colors: active)'))
 assert(!css.includes('scrollbar-width: none'))
 assert(!css.includes('* { scrollbar-gutter: stable'))
 const main=await readFile(new URL('../main.tsx',import.meta.url),'utf8')
 assert(main.indexOf("import './scrollbars.css'")>main.indexOf("import './features/health-analysis.css'"))
})
