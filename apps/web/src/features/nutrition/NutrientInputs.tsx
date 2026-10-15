import { label, nutrientFields, nutrientGroup, type Nutrition } from './model'

export function NutrientInputs({values}:{values?:Nutrition}) {
 return <>{['Energy & other nutrients','Fats','Vitamins','Minerals','Amino acids'].map(group=><details className="nutrient-section" key={group} open={group==='Energy & other nutrients'}><summary>{group}</summary><div className="override-grid">{nutrientFields.filter(([code])=>nutrientGroup(code)===group).map(([code,unit])=><label key={code}>{label(code)} ({unit})<input name={code} type="number" min="0" step="any" defaultValue={values?.[code]?.status==='KNOWN'?values[code].amount:''}/></label>)}</div></details>)}</>
}

export function NutrientOverrideInputs({values,onChange}:{values:Nutrition;onChange:(code:string,amount:string,unit:string)=>void}) {
 return <>{['Energy & other nutrients','Fats','Vitamins','Minerals','Amino acids'].map(group=><details className="nutrient-section" key={group} open={group==='Energy & other nutrients'}><summary>{group}</summary><div className="override-grid">{nutrientFields.filter(([code])=>nutrientGroup(code)===group).map(([code,unit])=><label key={code}>{label(code)} ({unit})<input type="number" min="0" step="any" value={values[code]?.amount||''} onChange={event=>onChange(code,event.target.value,unit)} placeholder="Use source"/></label>)}</div></details>)}</>
}
