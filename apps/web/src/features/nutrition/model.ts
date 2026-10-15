export type Nutrient = { status: string; amount?: string; unit: string }
export type Nutrition = Record<string, Nutrient>
export type NutritionDay = { tracking_day: string; totals: Nutrition; detail_compacted: boolean; confirmed_item_count: number }
export type FoodGroup = { id: string; meal_category: string; aggregate_final_nutrition: Nutrition; items: { position: number; food_source: { name: string }; quantity: { amount: string; unit: string }; final_nutrition: Nutrition; provider_nutrition: Nutrition }[] }
export type HistoricalTarget = { revision:number; status:string; effective_from_tracking_day:string; targets:Record<string,{active_value:string;unit:string}> }
export function targetForDay(history:HistoricalTarget[],day:string,code:string) {
  return history.filter(version=>['ACTIVE','SUPERSEDED'].includes(version.status)&&version.effective_from_tracking_day<=day)
    .sort((a,b)=>b.effective_from_tracking_day.localeCompare(a.effective_from_tracking_day)||b.revision-a.revision)[0]?.targets[code]
}
export type Preview = { preview_token: string; expires_at: string; group: FoodGroup }
export type Page<T> = { items: T[]; page: { next_cursor: string | null } }
export const nutrientFields = [
  ['ENERGY', 'kcal'], ['PROTEIN', 'g'], ['CARBOHYDRATE', 'g'], ['FAT', 'g'], ['FIBER', 'g'],
  ['SUGARS', 'g'], ['SATURATED_FAT', 'g'], ['MONOUNSATURATED_FAT', 'g'], ['POLYUNSATURATED_FAT', 'g'], ['CHOLESTEROL', 'mg'],
  ['CALCIUM', 'mg'], ['IRON', 'mg'], ['MAGNESIUM', 'mg'], ['PHOSPHORUS', 'mg'], ['POTASSIUM', 'mg'], ['SODIUM', 'mg'], ['ZINC', 'mg'], ['SELENIUM', 'mcg'],
  ['VITAMIN_A', 'mcg'], ['VITAMIN_C', 'mg'], ['VITAMIN_D', 'mcg'], ['VITAMIN_E', 'mg'], ['VITAMIN_K', 'mcg'],
  ['THIAMIN', 'mg'], ['RIBOFLAVIN', 'mg'], ['NIACIN', 'mg'], ['VITAMIN_B6', 'mg'], ['VITAMIN_B12', 'mcg'], ['FOLATE_DFE', 'mcg'],
  ['SOLUBLE_FIBER','g'], ['INSOLUBLE_FIBER','g'],
  ...['TRYPTOPHAN','THREONINE','ISOLEUCINE','LEUCINE','LYSINE','METHIONINE','CYSTINE','PHENYLALANINE','TYROSINE','VALINE','ARGININE','HISTIDINE','ALANINE','ASPARTIC_ACID','GLUTAMIC_ACID','GLYCINE','PROLINE','SERINE'].map(code=>[code,'g']),
]

export function label(code: string) {
  const special: Record<string, string> = { VITAMIN_A: 'Vitamin A (RAE)', FOLATE_DFE: 'Folate (DFE)', VITAMIN_E: 'Vitamin E (alpha-tocopherol)', VITAMIN_K: 'Vitamin K (phylloquinone)' }
  if (special[code]) return special[code]
  return code.toLowerCase().replaceAll('_', ' ').replace(/\b\w/g, letter => letter.toUpperCase())
}
export function known(value?: Nutrient): number | null {
  if (value?.status !== 'KNOWN' || value.amount === undefined || value.amount === '') return null
  const number = Number(value.amount)
  return Number.isFinite(number) && number >= 0 ? number : null
}
export function display(value?: Nutrient) {
  const number = known(value)
  return number === null ? 'Unknown' : `${number.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${value!.unit}`
}
export function shiftDay(day: string, offset: number) {
  const date = new Date(`${day}T12:00:00Z`)
  date.setUTCDate(date.getUTCDate() + offset)
  return date.toISOString().slice(0, 10)
}
export function rangeDays(from: string, to: string) {
  const result: string[] = []
  for (let day = from; day <= to && result.length < 366; day = shiftDay(day, 1)) result.push(day)
  return result
}
export function nutrientGroup(code: string) {
  if (['TRYPTOPHAN','THREONINE','ISOLEUCINE','LEUCINE','LYSINE','METHIONINE','CYSTINE','PHENYLALANINE','TYROSINE','VALINE','ARGININE','HISTIDINE','ALANINE','ASPARTIC_ACID','GLUTAMIC_ACID','GLYCINE','PROLINE','SERINE'].includes(code)) return 'Amino acids'
  if (code.startsWith('VITAMIN') || ['THIAMIN', 'RIBOFLAVIN', 'NIACIN', 'FOLATE', 'FOLATE_DFE'].includes(code)) return 'Vitamins'
  if (['CALCIUM', 'IRON', 'MAGNESIUM', 'PHOSPHORUS', 'POTASSIUM', 'SODIUM', 'ZINC', 'SELENIUM'].includes(code)) return 'Minerals'
  if (code.includes('FAT') || code === 'CHOLESTEROL') return 'Fats'
  return 'Energy & other nutrients'
}

// A missing day is not evidence of zero consumption. A unit mismatch cannot
// be combined. The caller displays how many recorded days have known values.
export function nutrientStats(days: NutritionDay[], code: string) {
  const unit = days.find(day => day.totals[code])?.totals[code].unit || ''
  const values = days.map(day => day.totals[code]?.unit === unit ? known(day.totals[code]) : null)
    .filter((value): value is number => value !== null)
  const total = values.reduce((sum, value) => sum + value, 0)
  return { unit, count: values.length, total, average: values.length ? total / values.length : null,
    complete: days.length > 0 && values.length === days.length }
}
