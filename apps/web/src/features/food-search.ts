export type SearchUsage = { usda_remaining:number; usda_limit:number; catalogue_count:number; matched_count:number; local_searches_today:number; resets_at:string }

type SearchFood = { name: string; data_type?: string; food_reference: { provider_food_id: string } }

// Rank only the returned page; retain distinct source records and their values.
// Generic reference foods are easier to identify than context-free branded names.
export function rankFoods<T extends SearchFood>(foods: T[], query: string): T[] {
  const words = query.toLowerCase().trim().split(/\s+/).filter(Boolean)
  const score = (food: T) => {
    const name = food.name.toLowerCase()
    return (food.data_type === 'Branded' ? 10 : 0)
      + (words.every(word => name.includes(word)) ? 0 : 20)
      + (/\braw\b/.test(name) ? -1 : 0)
  }
  return [...new Map(foods.map(food => [food.food_reference.provider_food_id, food])).values()]
    .sort((a, b) => score(a) - score(b))
}
