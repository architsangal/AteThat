import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { deleteAuthenticatedJson, getAuthenticatedJson, patchAuthenticatedJson, postAuthenticatedIdempotentJson, postAuthenticatedJson } from '../../api'
import { display, label, type Nutrition, type Page, type Preview } from '../nutrition/model'

type Reference = { type: 'USDA' | 'CUSTOM'; provider_food_id?: string | null; custom_food_id?: string | null }
type Component = { position: number; food_reference: Reference; quantity: { amount: string; unit: string }; food_state: string }
type Recipe = { id: string; version: number; name: string; recipe_servings: string; preparation: string; components: Component[] }
type Ingredient = Component & { name: string }
type Food = { id?: string; name: string; food_reference?: Reference; food_state: string; base_quantity: { amount: string; unit: string }; nutrients: Nutrition }

export function RecipesPage() {
  const [recipes, setRecipes] = useState<Recipe[]>([])
  const [cursor, setCursor] = useState<string | null>(null)
  const [editing, setEditing] = useState<Recipe | null>(null)
  const [name, setName] = useState('')
  const [yieldCount, setYieldCount] = useState('4')
  const [preparation, setPreparation] = useState('')
  const [ingredients, setIngredients] = useState<Ingredient[]>([])
  const [query, setQuery] = useState('')
  const [source, setSource] = useState<'USDA' | 'CUSTOM'>('CUSTOM')
  const [results, setResults] = useState<Food[]>([])
  const [foodCursor, setFoodCursor] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [message, setMessage] = useState('')
  const [batch, setBatch] = useState<Preview | null>(null)
  const [portion, setPortion] = useState<Preview | null>(null)
  const [logging, setLogging] = useState<Recipe | null>(null)
  const [servings, setServings] = useState('1')
  const [category, setCategory] = useState('OTHER')
  const [logPreview, setLogPreview] = useState<Preview | null>(null)
  const [confirmationKey, setConfirmationKey] = useState('')
  const [portionBasis, setPortionBasis] = useState<Preview | null>(null)
  const [overrides, setOverrides] = useState<Record<number, Nutrition>>({})

  async function load(next?: string) {
    const page = await getAuthenticatedJson<Page<Recipe>>(`/meals?${new URLSearchParams({ limit: '50', ...(next ? { cursor: next } : {}) })}`)
    setRecipes(previous => next ? [...previous, ...page.items] : page.items); setCursor(page.page.next_cursor)
  }
  useEffect(() => { void load().catch(cause => setError(cause.message)) }, [])
  function invalidate() { setBatch(null); setPortion(null); setMessage('') }
  function reset() { setEditing(null); setName(''); setYieldCount('4'); setPreparation(''); setIngredients([]); invalidate() }
  async function search(next?: string) {
    setBusy(true); setError('')
    try {
      const path = source === 'CUSTOM' ? '/foods/custom' : '/foods/search'
      const page = await getAuthenticatedJson<Page<Food>>(`${path}?${new URLSearchParams({ limit: '20', ...(source === 'USDA' ? { q: query } : {}), ...(next ? { cursor: next } : {}) })}`)
      setResults(previous => next ? [...previous, ...page.items] : page.items); setFoodCursor(page.page.next_cursor)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not find ingredients.') }
    finally { setBusy(false) }
  }
  function add(food: Food) {
    invalidate()
    setIngredients(previous => [...previous, { position: previous.length, name: food.name,
      food_reference: food.food_reference || { type: 'CUSTOM', custom_food_id: food.id }, quantity: { ...food.base_quantity }, food_state: food.food_state }])
  }
  function update(index: number, patch: Partial<Ingredient>) { invalidate(); setIngredients(previous => previous.map((item, i) => i === index ? { ...item, ...patch } : item)) }
  async function calculate() {
    setBusy(true); setError(''); setBatch(null); setPortion(null)
    try {
      const count = Number(yieldCount)
      if (!Number.isFinite(count) || count <= 0 || ingredients.length === 0) throw new Error('Add ingredients and a positive serving yield.')
      const base = { logged_at: new Date().toISOString(), meal_category: 'OTHER', label: name || 'Recipe calculation' }
      const items = ingredients.map((item, position) => ({ position, food_reference: item.food_reference, quantity: item.quantity, food_state: item.food_state }))
      const whole = await postAuthenticatedJson<Preview>('/food-log-groups/preview', { ...base, items })
      const one = await postAuthenticatedJson<Preview>('/food-log-groups/preview', { ...base, items, recipe_yield: yieldCount, recipe_portions: '1' })
      setBatch(whole); setPortion(one)
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not calculate nutrition.') }
    finally { setBusy(false) }
  }
  async function save() {
    if (!batch || !portion) return
    setBusy(true); setError('')
    try {
      if (name.trim().length < 2) throw new Error('Give the recipe a name of at least two characters.')
      const body = { name: name.trim(), recipe_servings: yieldCount, preparation, components: ingredients.map(({ name: _name, ...item }, position) => ({ ...item, position })) }
      if (editing) await patchAuthenticatedJson(`/meals/${editing.id}`, { ...body, version: editing.version })
      else await postAuthenticatedJson('/meals', body)
      await load(); reset(); setMessage('Recipe saved. Choose a portion when you are ready to log it.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not save the recipe.') }
    finally { setBusy(false) }
  }
  async function edit(recipe: Recipe) {
    setBusy(true); setError('')
    try {
      const rows = await Promise.all(recipe.components.map(async item => {
        const path = item.food_reference.type === 'CUSTOM' ? `/foods/custom/${item.food_reference.custom_food_id}` : `/foods/provider/${item.food_reference.provider_food_id}`
        const food = await getAuthenticatedJson<Food>(path)
        return { ...item, name: food.name }
      }))
      setEditing(recipe); setName(recipe.name); setYieldCount(recipe.recipe_servings); setPreparation(recipe.preparation || ''); setIngredients(rows); invalidate()
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not load the recipe ingredients.') }
    finally { setBusy(false) }
  }
  async function remove(recipe: Recipe) {
    if (!window.confirm(`Delete ${recipe.name}? Confirmed food records retain their snapshots.`)) return
    setBusy(true); setError('')
    try { await deleteAuthenticatedJson(`/meals/${recipe.id}`); await load(); if (editing?.id === recipe.id) reset(); if (logging?.id === recipe.id) { setLogging(null); setLogPreview(null) } }
    catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not delete recipe.') }
    finally { setBusy(false) }
  }
  async function reviewPortion() {
    if (!logging) return
    setBusy(true); setError(''); setLogPreview(null)
    try {
      const preview = await postAuthenticatedJson<Preview>(`/meals/${logging.id}/preview`, { logged_at: new Date().toISOString(), meal_category: category, label: logging.name, servings, overrides:Object.entries(overrides).map(([position,nutrients])=>({position:Number(position),nutrients})) })
      setLogPreview(preview); setPortionBasis(preview); setConfirmationKey(crypto.randomUUID())
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not review this portion.') }
    finally { setBusy(false) }
  }
  async function confirm() {
    if (!logPreview) return
    setBusy(true); setError('')
    try {
      await postAuthenticatedIdempotentJson('/food-log-groups/confirm', { preview_token: logPreview.preview_token }, confirmationKey)
      setLogPreview(null); setLogging(null); setMessage('Recipe portion logged. Its nutrition is preserved as a historical snapshot.')
    } catch (cause) { setError(cause instanceof Error ? cause.message : 'Could not confirm the food log.') }
    finally { setBusy(false) }
  }
  function overrideIngredient(position:number, code:string, amount:string, unit:string) {
    setOverrides(previous=>{const nutrients={...previous[position]};if(amount==='')delete nutrients[code];else nutrients[code]={status:'KNOWN',amount,unit};return {...previous,[position]:nutrients}})
    setLogPreview(null)
  }
  const overrideEditor = portionBasis && <details className="recipe-review"><summary>Adjust ingredient nutrition for this portion</summary><p className="muted">Overrides are absolute amounts for the portion eaten. Leave blank to use calculated nutrition, then review the portion again.</p>{portionBasis.group.items.map(item => <section key={item.position}><h3>{item.food_source.name}</h3><div className="override-grid">{Object.entries(item.provider_nutrition).map(([code,value]) => <label key={code}>{label(code)} ({value.unit})<input type="number" min="0" step="any" placeholder={value.amount || 'Unknown'} value={overrides[item.position]?.[code]?.amount || ''} onChange={event => overrideIngredient(item.position, code, event.target.value, value.unit)} /></label>)}</div></section>)}</details>
  return <>
    <div className="page-head"><div><span className="eyebrow">From your kitchen</span><h1>Recipes</h1><p className="muted">Build from real ingredients. Review the whole batch and each serving before saving.</p></div></div>
    {error && <p className="form-message form-error" role="alert">{error}</p>}{message && <p className="form-message form-success" role="status">{message}</p>}
    <div className="recipe-layout"><section className="panel"><div className="panel-top"><h2>Your recipe collection</h2><button className="quiet-link" disabled={busy} onClick={reset}>New recipe</button></div>{recipes.filter(recipe => recipe.recipe_servings).length === 0 && <p className="muted">Your first recipe starts here. Add ingredients, set the yield, and review nutrition.</p>}
      {recipes.filter(recipe => recipe.recipe_servings).map(recipe => <article className="recipe-card" key={recipe.id}><h3>{recipe.name}</h3><p className="muted">{recipe.components.length} ingredients · makes {recipe.recipe_servings} servings</p><div className="history-actions"><button className="button button-outline button-small" disabled={busy} onClick={() => void edit(recipe)}>Edit</button><button className="button button-primary button-small" disabled={busy} onClick={() => { setLogging(recipe); setServings('1'); setLogPreview(null); setPortionBasis(null); setOverrides({}); setMessage('') }}>Log a portion</button><button className="quiet-link" disabled={busy} onClick={() => void remove(recipe)}>Delete</button></div></article>)}
      {cursor && <button className="button button-outline" disabled={busy} onClick={() => { setBusy(true); void load(cursor).catch(cause => setError(cause.message)).finally(() => setBusy(false)) }}>Load more meals & recipes</button>}<p className="muted">Ordinary reusable meals are available in <Link to="/app/meals">Meals</Link>.</p>
    </section><section className="panel"><h2>{editing ? 'Edit recipe' : 'Create a recipe'}</h2><fieldset disabled={busy} className="recipe-fieldset">
      <div className="recipe-meta"><label>Recipe name<input maxLength={120} value={name} onChange={e => { setName(e.target.value); invalidate() }} /></label><label>Batch yield (servings)<input type="number" min="0.001" step="any" value={yieldCount} onChange={e => { setYieldCount(e.target.value); invalidate() }} /></label></div>
      <label>Preparation notes<textarea maxLength={4000} rows={3} value={preparation} onChange={e => { setPreparation(e.target.value); invalidate() }} /></label>
      <p className="muted">Enter quantities in the food state you selected. Cooking does not automatically change nutrition or estimate cooking losses.</p>
      <div className="recipe-search"><select aria-label="Ingredient source" value={source} onChange={e => { setSource(e.target.value as 'USDA' | 'CUSTOM'); setResults([]); setFoodCursor(null) }}><option value="CUSTOM">My custom foods</option><option value="USDA">USDA foods</option></select>{source === 'USDA' && <input aria-label="Search ingredients" placeholder="Rice, paneer, oats…" value={query} onChange={e => setQuery(e.target.value)} />}<button className="button button-outline" disabled={source === 'USDA' && query.trim().length < 2} onClick={() => void search()}>Find ingredients</button></div>
      <div className="recipe-search-results">{results.map(food => <div className="contribution-food" key={food.id || food.food_reference?.provider_food_id}><span>{food.name}<small>{food.food_state.toLowerCase()} · {food.base_quantity.amount} {food.base_quantity.unit}</small></span><button className="quiet-link" disabled={ingredients.length >= 50} onClick={() => add(food)}>Add</button></div>)}{foodCursor && <button className="quiet-link" onClick={() => void search(foodCursor)}>More ingredients</button>}</div>
      {ingredients.map((item, index) => <div className="recipe-ingredient" key={index}><strong>{index + 1}. {item.name}</strong><div className="recipe-quantity"><label>Quantity ({item.quantity.unit})<input type="number" min="0.001" step="any" value={item.quantity.amount} onChange={e => update(index, { quantity: { ...item.quantity, amount: e.target.value } })} /></label><label>Food state<select value={item.food_state} onChange={e => update(index, { food_state: e.target.value })}>{['RAW', 'COOKED', 'PREPARED', 'UNKNOWN'].map(state => <option key={state}>{state}</option>)}</select></label><button className="quiet-link" onClick={() => { invalidate(); setIngredients(previous => previous.filter((_, i) => i !== index)) }}>Remove</button></div></div>)}
      <button className="button button-primary" disabled={!ingredients.length} onClick={() => void calculate()}>Calculate & review</button>
      {batch && portion && <div className="recipe-review"><h3>Nutrition review</h3><p className="muted">Ingredient-based calculation · unknown values remain unknown. Saving a definition does not log food.</p><div className="nutrition-table-wrap"><table className="nutrition-table"><thead><tr><th>Nutrient</th><th>Whole batch</th><th>Per serving</th></tr></thead><tbody>{Object.keys(batch.group.aggregate_final_nutrition).map(code => <tr key={code}><th>{label(code)}</th><td>{display(batch.group.aggregate_final_nutrition[code])}</td><td>{display(portion.group.aggregate_final_nutrition[code])}</td></tr>)}</tbody></table></div><button className="button button-primary" onClick={() => void save()}>Save reviewed recipe</button></div>}
    </fieldset></section></div>
    {logging && <section className="panel recipe-review"><h2>Log {logging.name}</h2><fieldset className="recipe-fieldset" disabled={busy}><div className="recipe-meta"><label>Servings eaten<input type="number" min="0.001" step="any" value={servings} onChange={e => { setServings(e.target.value); setLogPreview(null); setPortionBasis(null); setOverrides({}) }} /></label><label>Meal<select value={category} onChange={e => { setCategory(e.target.value); setLogPreview(null) }}>{['BREAKFAST', 'LUNCH', 'DINNER', 'SNACK', 'OTHER'].map(value => <option key={value}>{value}</option>)}</select></label></div><div className="history-actions"><button className="button button-outline" onClick={() => void reviewPortion()}>Review portion</button><button className="quiet-link" onClick={() => { setLogging(null); setLogPreview(null) }}>Cancel</button></div>{overrideEditor}{logPreview && <><h3>Confirm your food record</h3>{logPreview.group.items.map(item => <p key={item.position}>{item.food_source.name} · {item.quantity.amount} {item.quantity.unit}</p>)}<div className="nutrition-table-wrap"><table className="nutrition-table"><tbody>{Object.entries(logPreview.group.aggregate_final_nutrition).map(([code, value]) => <tr key={code}><th>{label(code)}</th><td>{display(value)}</td></tr>)}</tbody></table></div><p className="muted">Review expires at {new Date(logPreview.expires_at).toLocaleTimeString()}.</p><button className="button button-primary" onClick={() => void confirm()}>Confirm & log portion</button></>}</fieldset></section>}
  </>
}
