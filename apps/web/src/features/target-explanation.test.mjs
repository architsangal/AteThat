import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createRequire } from 'node:module'
import { build } from 'esbuild'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'

const compiled = await build({ entryPoints: [new URL('./TargetSettings.tsx', import.meta.url).pathname], bundle: true, packages: 'external', platform: 'node', format: 'cjs', write: false, define: { 'import.meta.env.VITE_API_BASE_URL': '"/api/v1"' } })
const loaded = { exports: {} }
new Function('module','exports','require',compiled.outputFiles[0].text)(loaded,loaded.exports,createRequire(import.meta.url))
const { TargetCalculationGuide } = loaded.exports
const plan = { inputs: { height_cm:'175',weight_kg:'70',activity:'MODERATE',assumed_activity:true,age:26 }, policy_version:'INTERNAL_VERSION_SHOULD_NOT_RENDER',explanation:'INTERNAL_POLICY_SHOULD_NOT_RENDER',pending_decisions:['INTERNAL_PENDING_SHOULD_NOT_RENDER'],rounding_tolerance_kcal:'1' }
const selected = { ENERGY:['2000','2000'],PROTEIN:['50','50'],CARBOHYDRATE:['275','275'],FAT:['79','78'],FIBER:['28','28'] }
const active = { targets: Object.fromEntries(Object.entries(selected).map(([key,[active_value,reference_value]]) => [key,{active_value,reference_value,provenance:key==='FAT'?'CUSTOM':''}])) }
function render(props) { return renderToStaticMarkup(React.createElement(MemoryRouter,null,React.createElement(TargetCalculationGuide,props))) }

test('guide explains the actual saved-macro calculation without internal policy jargon', () => {
  const html = render({plan,active,macroEnergy:'2011'})
  for (const text of ['How your nutrition targets are calculated','Calories = (protein × 4) + (carbohydrate × 4) + (fat × 9)','(50 × 4) + (275 × 4) + (79 × 9) = 2011 kcal','23 × 1.75 × 1.75 = 70.4375 kg','Calculated defaults from your profile','From your saved profile','Default used','Protein = profile weight (kg) × activity protein rate','Fibre (g) = daily calories × 14 ÷ 1,000']) assert(html.includes(text),text)
  for (const hidden of ['INTERNAL_VERSION','INTERNAL_POLICY','INTERNAL_PENDING','Versioned policy','Pending policy decisions','GENERAL_BENCHMARK']) assert(!html.includes(hidden),hidden)
  assert(html.includes('The difference is 11 kcal'))
  assert(html.includes('Custom target'))
})
test('calculated reset comparison contains all five old/new values and no writes', () => {
 const { ResetComparison } = loaded.exports
 const preview = {...plan,calculated_defaults:{ENERGY:'2500.123456789',PROTEIN:'84.525',CARBOHYDRATE:'350',FAT:'80',FIBER:'35'}}
 const html = renderToStaticMarkup(React.createElement(ResetComparison,{active,preview}))
 for (const text of ['Current and calculated targets','Current','Calculated','2,500.12','84.53','Fibre','35 g']) assert(html.includes(text),text)
 assert.equal(preview.calculated_defaults.ENERGY,'2500.123456789')
})
test('guide separates absent targets and failed calculation from valid profile information', () => {
  const html = render({plan,active:null,macroEnergy:'',macroError:'unavailable'})
  assert(html.includes('You have not saved any nutrition targets yet'))
  assert(!html.includes('Using your saved macros'))
  assert(html.includes('Your saved macro calculation could not be loaded'))
  assert(html.includes('not a measurement, BMI assessment or recommended weight'))
})
test('outside-reference review shows the exact selection, energy basis and source without confirmation secrets', () => {
 const html = renderToStaticMarkup(React.createElement(loaded.exports.ReferenceWarnings,{warnings:[{nutrient:'PROTEIN',selected_value:'500',unit:'g',reference_range:{minimum:'50',maximum:'175',energy_basis:'2000',source_url:'https://example.org/reference',reference_revision:'internal-revision'}}]}))
 for (const text of ['500','50','175','2,000','https://example.org/reference']) assert(html.includes(text),text)
 assert(!html.includes('internal-revision'))
})
