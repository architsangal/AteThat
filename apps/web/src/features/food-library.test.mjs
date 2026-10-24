import assert from 'node:assert/strict'
import {test} from 'node:test'
import {createRequire} from 'node:module'
import {build} from 'esbuild'
import React from 'react'
import {renderToStaticMarkup} from 'react-dom/server'

const compile=async entry=>{
 const contents=`export * from ${JSON.stringify(new URL(entry,import.meta.url).pathname)}; export {DatePickerProvider} from ${JSON.stringify(new URL('./AppDatePicker.tsx',import.meta.url).pathname)};`
 const result=await build({stdin:{contents,resolveDir:new URL('.',import.meta.url).pathname,loader:'tsx'},bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'}})
 const module={exports:{}}
 new Function('module','exports','require',result.outputFiles[0].text)(module,module.exports,createRequire(import.meta.url))
 return module.exports
}
const {QuickLog,quickLogNutrition,DatePickerProvider}=await compile('./nutrition/QuickLog.tsx')
const {normalizeTracked}=await compile('./nutrition/TrackedNutrients.tsx')

test('Quick Log zeros only blank visible fields when explicitly enabled, preserving hidden unknowns',()=>{
 const data=new FormData()
 data.set('ENERGY','220.125');data.set('PROTEIN','');data.set('zero','on')
 const result=quickLogNutrition(data)
 assert.deepEqual(result.ENERGY,{status:'KNOWN',amount:'220.125',unit:'kcal'})
 assert.deepEqual(result.PROTEIN,{status:'KNOWN',amount:'0',unit:'g'})
 assert.deepEqual(result.VITAMIN_C,{status:'UNKNOWN',unit:'mg'})
 data.delete('zero')
 assert.equal(quickLogNutrition(data).PROTEIN.status,'UNKNOWN')
})
test('tracked selection supports vitamins with digits, all nutrients, and energy-only selection',()=>{
 assert.deepEqual(normalizeTracked(['VITAMIN_B12','PROTEIN','PROTEIN','INVALID']),['ENERGY','PROTEIN','VITAMIN_B12'])
 assert.deepEqual(normalizeTracked(['ENERGY']),['ENERGY'])
 assert.deepEqual(normalizeTracked(null),['ENERGY','PROTEIN','CARBOHYDRATE','FAT','FIBER'])
})
test('Quick Log offers selected nutrient fields, explicit reuse choice and two logging paths',()=>{
 const html=renderToStaticMarkup(React.createElement(DatePickerProvider,null,React.createElement(QuickLog,{busy:false,onAdd:async()=>{},category:'OTHER',onCategory:()=>{},loggedAt:'',onLoggedAt:()=>{}})))
 for(const field of ['ENERGY','PROTEIN','CARBOHYDRATE','FAT','FIBER'])assert(html.includes(`name="${field}"`))
 assert(!html.includes('name="VITAMIN_C"'))
 assert(html.includes('Save to Food Library'))
 assert(html.includes('Use zero for empty fields'))
 assert(html.includes('Add to meal')&&html.includes('Review meal'))
})
