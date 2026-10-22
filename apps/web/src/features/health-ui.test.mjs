import assert from 'node:assert/strict'
import {test} from 'node:test'
import {build} from 'esbuild'
import {createRequire} from 'node:module'
import React from 'react'
import {renderToStaticMarkup} from 'react-dom/server'
import {readFile} from 'node:fs/promises'
async function load(file){const built=await build({entryPoints:[new URL(file,import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'}});const mod={exports:{}};new Function('module','exports','require',built.outputFiles[0].text)(mod,mod.exports,createRequire(import.meta.url));return mod.exports}
const {AppTimeInput}=await load('./AppTimeInput.tsx'),{HealthOccurrenceResults}=await load('./HealthOccurrences.tsx')
test('the shared calendar provider encloses dialog rendering, including reschedule fields',async()=>{
 const source=await readFile(new URL('../App.tsx',import.meta.url),'utf8')
 assert(source.includes('<ThemeProvider><DatePickerProvider><DialogProvider>'))
})
test('time field uses labelled themed 24-hour segments and submits canonical times only',()=>{
 const render=value=>renderToStaticMarkup(React.createElement(AppTimeInput,{label:'Time 1',name:'schedule_time',value,onChange:()=>{}}))
 assert(render('23:59').includes('name="schedule_time" value="23:59"'))
 assert(render('08:').includes('name="schedule_time" value=""'))
 assert(render('08:00').includes('Time 1 hour (24-hour)'))
 assert(!render('08:00').includes('type="time"'))
})
test('done tasks have no reschedule action and do not leak into the active tab',()=>{
 const row={occurrence_id:'test',name:'Brush teeth',type:'TASK',status:'TAKEN',scheduled_at:'2026-10-22T08:00:00Z',scheduled_local:{date:'2026-10-22',time:'08:00',timezone:'UTC'}}
 const render=tab=>renderToStaticMarkup(React.createElement(HealthOccurrenceResults,{rows:[row],tab,busy:false,onAction:()=>{},onReschedule:()=>{}}))
 assert(!render('ACTIVE').includes('Brush teeth'))
 assert(render('TAKEN').includes('Brush teeth'))
 assert(!render('TAKEN').includes('>Reschedule<'))
})
