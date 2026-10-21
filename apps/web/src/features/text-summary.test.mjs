import assert from 'node:assert/strict'
import { test } from 'node:test'
import { createRequire } from 'node:module'
import { build } from 'esbuild'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

const compiled = await build({ entryPoints:[new URL('./TextSummary.tsx',import.meta.url).pathname],bundle:true,packages:'external',platform:'node',format:'cjs',write:false,define:{'import.meta.env.VITE_API_BASE_URL':'"/api/v1"'} })
const loaded = {exports:{}}
new Function('module','exports','require',compiled.outputFiles[0].text)(loaded,loaded.exports,createRequire(import.meta.url))
const {TextSummaryCard} = loaded.exports
const noop = () => {}
const base = {from:'2026-10-21',to:'2026-10-21',mode:'DAY',nutrient:'ALL',source:{text:'Full factual report',context:'DAY|2026-10-21|2026-10-21|ALL',generated_at:'2026-10-21T18:27:19Z',partial:false},draft:'Full factual report',busy:false,error:'',status:'',expanded:false,dirty:false,stale:false,onToggle:noop,onChange:noop,onCopy:noop,onRegenerate:noop}
const render = (extra={}) => renderToStaticMarkup(React.createElement(TextSummaryCard,{...base,...extra}))

test('summary starts collapsed with accessible disclosure and always-visible actions',()=>{
 const html=render()
 assert(html.includes('aria-expanded="false"'))
 assert.match(html,/<div[^>]*class="text-summary-editor"[^>]*hidden=""/)
 assert(html.includes('Copy full summary'))
 assert(html.includes('Regenerate summary'))
 assert(html.indexOf('Summary actions') < html.indexOf('text-summary-disclosure'))
 assert(html.includes('21 Oct 2026'))
 assert(!html.includes('Requested context:'))
 assert(!html.includes('DAY|'))
})
test('expanded summary keeps its draft and explicitly labels privacy and editing',()=>{
 const html=render({expanded:true,dirty:true,draft:'Edited report with all lines'})
 assert(html.includes('aria-expanded="true"'))
 assert(!/<div[^>]*class="text-summary-editor"[^>]*hidden/.test(html))
 for(const text of ['Hide summary','Edited report with all lines','Unsaved edits','sensitive health information','Not saved to your records']) assert(html.includes(text),text)
})
test('analysis shares the card and stale, partial and failure notices remain visible while closed',()=>{
 const html=render({mode:'RANGE',to:'2026-10-22',nutrient:'PROTEIN',stale:true,error:'Network unavailable',source:{...base.source,partial:true}})
 for(const text of ['21 Oct 2026 – 22 Oct 2026','Analysis · Protein','previous selection','Some sections are incomplete','Network unavailable']) assert(html.includes(text),text)
 assert(html.includes('aria-expanded="false"'))
})
test('regeneration disables only regeneration; existing text remains copyable',()=>{
 const html=render({busy:true})
 assert.match(html,/<button[^>]*class="button button-primary"[^>]*>.*?Copy full summary<\/button>/)
 assert(!/<button[^>]*class="button button-primary"[^>]*disabled/.test(html))
 assert.match(html,/<button[^>]*class="button button-outline"[^>]*disabled/)
 assert(html.includes('Preparing report'))
})
