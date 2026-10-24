import { label, nutrientFields, nutrientGroup, type Nutrition } from './model'
import { useState } from 'react'
import { useTrackedNutrients } from './TrackedNutrients'

export function NutrientInputs({values}:{values?:Nutrition}) {
 const {codes} = useTrackedNutrients(); const [extra,setExtra] = useState<string[]>([])
 const visible = [...codes,...extra]
 return <>{['Energy & other nutrients','Fats','Vitamins','Minerals','Amino acids'].filter(group=>nutrientFields.some(([code])=>visible.includes(code)&&nutrientGroup(code)===group)).map(group=><details className="nutrient-section" key={group} open><summary>{group}</summary><div className="override-grid">{nutrientFields.filter(([code])=>visible.includes(code)&&nutrientGroup(code)===group).map(([code,unit])=><label key={code}>{label(code)} ({unit})<input name={code} type="number" min="0" step="any" defaultValue={values?.[code]?.status==='KNOWN'?values[code].amount:''}/></label>)}</div></details>)}<label>Add another nutrient<select value="" onChange={event=>{if(event.target.value)setExtra(previous=>[...previous,event.target.value])}}><option value="">Choose a nutrient for this entry</option>{nutrientFields.filter(([code])=>!visible.includes(code)).map(([code])=><option key={code} value={code}>{label(code)}</option>)}</select></label></>
}

export function NutrientOverrideInputs({values,onChange}:{values:Nutrition;onChange:(code:string,amount:string,unit:string)=>void}) {
 const {codes} = useTrackedNutrients(); const [extra,setExtra] = useState<string[]>([])
 const visible=[...new Set([...codes,...extra,...Object.keys(values)])]
 return <>{['Energy & other nutrients','Fats','Vitamins','Minerals','Amino acids'].filter(group=>nutrientFields.some(([code])=>visible.includes(code)&&nutrientGroup(code)===group)).map(group=><details className="nutrient-section" key={group} open><summary>{group}</summary><div className="override-grid">{nutrientFields.filter(([code])=>visible.includes(code)&&nutrientGroup(code)===group).map(([code,unit])=><label key={code}>{label(code)} ({unit})<input type="number" min="0" step="any" value={values[code]?.amount||''} onChange={event=>onChange(code,event.target.value,unit)} placeholder="Use source"/></label>)}</div></details>)}<label>Add another nutrient<select value="" onChange={event=>{if(event.target.value)setExtra(previous=>[...previous,event.target.value])}}><option value="">Choose a nutrient</option>{nutrientFields.filter(([code])=>!visible.includes(code)).map(([code])=><option key={code} value={code}>{label(code)}</option>)}</select></label></>
}
