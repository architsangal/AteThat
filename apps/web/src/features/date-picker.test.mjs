import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createRequire } from 'node:module'
import { build } from 'esbuild'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
const built = await build({entryPoints:[new URL('./AppDatePicker.tsx',import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false})
const loaded={exports:{}}
new Function('module','exports','require',built.outputFiles[0].text)(loaded,loaded.exports,createRequire(import.meta.url))
const {AppDatePicker,DatePickerProvider}=loaded.exports
const render=props=>renderToStaticMarkup(React.createElement(DatePickerProvider,null,React.createElement(AppDatePicker,{label:'From Tracking Day',value:'2026-10-15',onChange:()=>{},...props})))
test('shared calendar displays an unambiguous date and exposes accessible popup controls',()=>{
 const html=render()
 assert(html.includes('From Tracking Day: 15 Oct 2026'))
 assert(html.includes('aria-haspopup="dialog"'))
 assert(html.includes('aria-expanded="false"'))
 assert(!html.includes('type="date"'))
})
test('DOB and form dates preserve their canonical ISO value and descriptions',()=>{
 const html=render({label:'Date of birth',name:'date_of_birth',value:'2000-02-29',describedBy:'dob-help'})
 assert(html.includes('Date of birth: 29 Feb 2000'))
 assert(html.includes('type="hidden" name="date_of_birth" value="2000-02-29"'))
 assert(html.includes('aria-describedby="dob-help"'))
 assert(render({value:''}).includes('Select date'))
})
