import { TextSummary } from '../TextSummary'
import { AppDatePicker } from '../AppDatePicker'
import { Form } from '../ui'
import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { getAuthenticatedJson } from '../../api'
import { display, known, label, nutrientFields, nutrientGroup, nutrientStats, rangeDays, shiftDay, targetForDay, type HistoricalTarget, type FoodGroup, type NutritionDay, type Page } from './model'

type Targets = { active: { targets: Record<string, { active_value: string; unit: string }> } | null }

function Trend({ days, from, to, code }: { days: NutritionDay[]; from: string; to: string; code: string }) {
  const dates = rangeDays(from, to)
  const unit = days.find(day => day.totals[code])?.totals[code].unit
  const values = dates.map(day => {
    const nutrient = days.find(row => row.tracking_day === day)?.totals[code]
    return nutrient?.unit === unit ? known(nutrient) : null
  })
  const max = Math.max(1, ...values.filter((v): v is number => v !== null))
  const points = values.map((value, i) => value === null ? null : { x: 34 + i * 624 / Math.max(1, dates.length - 1), y: 165 - value / max * 135, value, day: dates[i] })
  return <figure className="nutrition-trend"><svg viewBox="0 0 700 205" role="img" aria-label={`${label(code)} by Tracking Day; gaps indicate no known value`}>
    {[0, .5, 1].map(ratio => <g key={ratio}><line x1="34" x2="658" y1={165 - ratio * 135} y2={165 - ratio * 135} className="chart-grid" /><text x="2" y={169 - ratio * 135}>{Math.round(max * ratio)}</text></g>)}
    {points.map((point, i) => point && <g key={point.day}>{i > 0 && points[i - 1] && <line x1={points[i - 1]!.x} y1={points[i - 1]!.y} x2={point.x} y2={point.y} className="chart-line" />}<circle cx={point.x} cy={point.y} r="4" className="chart-point"><title>{point.day}: {point.value}</title></circle></g>)}
    <text x="34" y="195">{from}</text><text x="658" y="195" textAnchor="end">{to}</text>
  </svg><figcaption>Daily {label(code).toLowerCase()} · gaps mean no known value. See exact values in the daily table below.</figcaption></figure>
}

export function NutritionAnalysisPage() {
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [range, setRange] = useState<{ from: string; to: string } | null>(null)
  const [days, setDays] = useState<NutritionDay[]>([])
  const [targets, setTargets] = useState<Targets>({ active: null })
  const [targetHistory, setTargetHistory] = useState<HistoricalTarget[]>([])
  const [code, setCode] = useState('ENERGY')
  const [detailDay, setDetailDay] = useState('')
  const [groups, setGroups] = useState<FoodGroup[]>([])
  const [detailBusy, setDetailBusy] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const detailRequest = useRef(0)
  useEffect(() => {
    let active = true
    getAuthenticatedJson<{ tracking_day: string }>('/tracking-day/current').then(day => {
      if (active) { setTo(day.tracking_day); setFrom(shiftDay(day.tracking_day, -6)) }
    }).catch(cause => { if (active) setError(cause.message) })
    return () => { active = false; detailRequest.current++ }
  }, [])

  async function load() {
    if (!from || !to || from > to || Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`) > 365 * 86400000) {
      setError('Choose a valid range of at most 366 Tracking Days.'); return
    }
    setBusy(true); setError(''); setDetailDay(''); setGroups([]); detailRequest.current++
    try {
      const rows: NutritionDay[] = []
      let cursor: string | null = null
      do {
        const query = new URLSearchParams({ from_tracking_day: from, to_tracking_day: to, limit: '50' })
        if (cursor) query.set('cursor', cursor)
        const page: Page<NutritionDay> = await getAuthenticatedJson(`/history/nutrition?${query}`)
        rows.push(...page.items); cursor = page.page.next_cursor
      } while (cursor)
      const targetSelection = await getAuthenticatedJson<Targets>('/nutrition/targets')
      const versions:HistoricalTarget[]=[]
      let targetCursor:string|null=null
      do {
        const query=new URLSearchParams({limit:'50',...(targetCursor?{cursor:targetCursor}:{})})
        const page:Page<HistoricalTarget>=await getAuthenticatedJson(`/nutrition/targets/history?${query}`)
        versions.push(...page.items);targetCursor=page.page.next_cursor
      } while(targetCursor)
      setTargetHistory(versions)
      setTargets(targetSelection); setDays(rows.sort((a, b) => a.tracking_day.localeCompare(b.tracking_day))); setRange({ from, to })
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load nutrition.') }
    finally { setBusy(false) }
  }
  async function drill(day: NutritionDay) {
    const request = ++detailRequest.current
    setDetailDay(day.tracking_day); setGroups([]); setDetailError(''); setDetailBusy(true)
    if (day.detail_compacted) { setDetailError('Only daily totals remain for this day. Individual food details have been compacted.'); setDetailBusy(false); return }
    try {
      const result: FoodGroup[] = []
      let cursor: string | null = null
      do {
        const query = new URLSearchParams({ tracking_day: day.tracking_day, limit: '50' })
        if (cursor) query.set('cursor', cursor)
        const page: Page<FoodGroup> = await getAuthenticatedJson(`/food-log-groups?${query}`)
        const details = await Promise.all(page.items.map(group => getAuthenticatedJson<FoodGroup>(`/food-log-groups/${group.id}`)))
        if (request !== detailRequest.current) return
        result.push(...details); cursor = page.page.next_cursor
      } while (cursor)
      setGroups(result)
    } catch (cause) { if (request === detailRequest.current) setDetailError(cause instanceof Error ? cause.message : 'Could not load food contributions.') }
    finally { if (request === detailRequest.current) setDetailBusy(false) }
  }
  const codes = [...new Set([...nutrientFields.map(([key]) => key), ...days.flatMap(day => Object.keys(day.totals))])]
  const stats = nutrientStats(days, code)
  function historicalTarget(day:NutritionDay) {
    const target=targetForDay(targetHistory,day.tracking_day,code)
    const amount=known(day.totals[code])
    if(!target)return 'No active target'
    const percent=amount!==null&&target.unit===day.totals[code]?.unit&&Number(target.active_value)>0 ? ` · ${(amount/Number(target.active_value)*100).toFixed(0)}%` : ''
    return `${target.active_value} ${target.unit}${percent}`
  }
  const target = targets.active?.targets[code]
  const progress = target && target.unit === stats.unit && Number(target.active_value) > 0 && stats.average !== null ? stats.average / Number(target.active_value) * 100 : null
  return <>
    <div className="page-head"><div><span className="eyebrow">Your nutrition, in focus</span><h1 tabIndex={-1}>Nutrition analysis</h1><p className="muted">Explore patterns, then follow each nutrient back to the foods you recorded.</p></div></div>
    <section className="panel"><Form className="analysis-filter" onSubmit={event => { event.preventDefault(); void load() }}>
      <AppDatePicker label="From Tracking Day" value={from} onChange={setFrom} max={to || undefined} />
      <AppDatePicker label="Through Tracking Day" value={to} onChange={setTo} min={from || undefined} />
      <button className="button button-primary" disabled={busy || !from}>{busy ? 'Loading all days…' : 'Explore nutrition'}</button>
    </Form><div className="filter-row">{[7, 30, 90].map(length => <button className="filter-chip" key={length} disabled={!to || busy} onClick={() => setFrom(shiftDay(to, 1 - length))}>Last {length} days</button>)}</div>
      {error && <p className="form-message form-error" role="alert">{error}</p>}
    </section>
    {!range ? <section className="panel nutrition-empty"><h2>{busy ? 'Loading your nutrition…' : 'A clearer picture of your food'}</h2><p>Choose a range to see trends, known-day averages, and food contributions.</p></section> : <>
      <div className="nutrition-stat-grid"><article className="panel"><span>Recorded days</span><h2>{days.length} <small>/ {rangeDays(range.from, range.to).length}</small></h2><p className="muted">Days without records are excluded from averages.</p></article><article className="panel"><span>Known-day average · {label(code)}</span><h2>{stats.average === null ? 'Unknown' : `${stats.average.toLocaleString(undefined, { maximumFractionDigits: 1 })} ${stats.unit}`}</h2><p className="muted">Based on {stats.count} days with known values.</p></article><article className="panel"><span>Against current daily target</span><h2>{progress === null ? '—' : `${progress.toFixed(0)}%`}</h2><p className="muted">{target ? `${target.active_value} ${target.unit}. Current target only; historical targets may differ.` : 'No active target for this nutrient.'}</p></article></div>
      <section className="panel"><div className="panel-top"><div><h2>Daily rhythm</h2><p className="muted">Tracking Day boundaries follow your saved timezone and reset time.</p></div><label>Nutrient<select value={code} onChange={e => setCode(e.target.value)}>{codes.map(key => <option key={key} value={key}>{label(key)}</option>)}</select></label></div><Trend days={days} from={range.from} to={range.to} code={code} /></section>
      <section className="panel"><h2>The complete nutrient picture</h2><p className="muted">Averages use known recorded days only. Unknown or missing nutrients never become zero.</p>{['Energy & other nutrients', 'Fats', 'Vitamins', 'Minerals', 'Amino acids'].map(group => <details className="nutrient-section" open={group==='Energy & other nutrients'||group==='Fats'} key={group}><summary>{group}</summary><div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><thead><tr><th>Nutrient</th><th>Known-day average</th><th>Coverage</th><th>Recorded total</th></tr></thead><tbody>{codes.filter(key => nutrientGroup(key) === group).map(key => {
        const value = nutrientStats(days, key)
        return <tr key={key}><th><button className="quiet-link" onClick={() => setCode(key)}>{label(key)}</button></th><td data-label="Daily average">{value.average === null ? 'Unknown' : `${value.average.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${value.unit}`}</td><td data-label="Known days">{value.count}/{days.length} days</td><td data-label="Recorded total">{value.complete ? `${value.total.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${value.unit}` : 'Incomplete'}</td></tr>
      })}</tbody></table></div></details>)}</section>
      <div className="two-column"><section className="panel"><h2>{label(code)} by day</h2>{days.length === 0 && <p className="muted">No recorded food in this range.</p>}<div className="nutrition-table-wrap"><table className="nutrition-table responsive-table"><thead><tr><th>Tracking Day</th><th>Recorded intake</th><th>Target on that day</th><th>Details</th></tr></thead><tbody>{days.map(day => <tr key={day.tracking_day}><th>{day.tracking_day}</th><td data-label="Recorded intake">{display(day.totals[code])}</td><td data-label="Daily target">{historicalTarget(day)}</td><td data-label="Food details"><button className="quiet-link" onClick={() => void drill(day)} aria-pressed={detailDay === day.tracking_day}>Contributions</button></td></tr>)}</tbody></table></div></section>
      <section className="panel"><h2>Food & meal contributions</h2><p className="muted">{detailDay || 'Choose a day to inspect its foods.'}</p>{detailBusy && <p role="status">Loading food details…</p>}{detailError && <p role="status" className="form-message">{detailError}</p>}{groups.map(group => <article className="contribution-group" key={group.id}><h3>{label(group.meal_category)} <span>{display(group.aggregate_final_nutrition[code])}</span></h3>{group.items.slice().sort((a, b) => (known(b.final_nutrition[code]) ?? -1) - (known(a.final_nutrition[code]) ?? -1)).map(item => <div className="contribution-food" key={item.position}><span>{item.food_source.name}<small>{item.quantity.amount} {item.quantity.unit}</small></span><strong>{display(item.final_nutrition[code])}</strong></div>)}</article>)}{detailDay && <Link className="quiet-link" to={`/app/history?tracking_day=${detailDay}`}>Open this day in history →</Link>}</section></div>
    </>}
    {from && to && <TextSummary from={range?.from || from} to={range?.to || to} mode="RANGE" nutrient={code} />}
  </>
}
