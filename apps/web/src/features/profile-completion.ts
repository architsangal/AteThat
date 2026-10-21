export type ProfileDetailKey = 'date_of_birth' | 'height' | 'weight' | 'activity_level' | 'sex'
export type ProfileDetails = { date_of_birth?: string; height?: object | null; weight?: object | null; activity_level?: string; sex?: string }
export type MissingProfileDetail = { key: ProfileDetailKey; label: string; fieldId: string; hint: string }
const definitions: Omit<MissingProfileDetail, 'fieldId'>[] = [
  { key: 'date_of_birth', label: 'Date of birth', hint: 'Not provided. You can add your date of birth here, or leave this optional detail blank.' },
  { key: 'height', label: 'Height', hint: 'Not provided — using the 175 cm default. Enter your height here to replace this assumption.' },
  { key: 'weight', label: 'Profile weight', hint: 'Not provided — using the 70.4375 kg default. Logged weight measurements do not fill this profile field.' },
  { key: 'activity_level', label: 'Activity level', hint: 'Not provided — using Moderate by default. Select an activity level, including Moderate, to confirm your choice.' },
  { key: 'sex', label: 'Sex', hint: 'Not provided. Select an option, including “Prefer not to provide”, to record your choice. This is optional.' },
]
export function missingProfileDetails(profile: ProfileDetails): MissingProfileDetail[] {
  // DECLINED is an explicit answer, not missing data that the user must disclose.
  return definitions.filter(field => !profile[field.key]).map(field => ({ ...field, fieldId: `profile-field-${field.key}` }))
}
export function profileAttentionHref(fields: MissingProfileDetail[]): string {
  return fields.length === 1 ? `/app/settings#${fields[0].fieldId}` : '/app/settings#profile-attention'
}
export function missingPlanningDetails(plan: { inputs: { assumed_height?: boolean; assumed_weight?: boolean; assumed_activity?: boolean; activity: string; sex?: string }; missing_inputs?: string[] }): MissingProfileDetail[] {
  return missingProfileDetails({
    date_of_birth: plan.missing_inputs?.includes('Date of birth') ? '' : 'provided',
    height: plan.inputs.assumed_height ? null : {},
    weight: plan.inputs.assumed_weight ? null : {},
    activity_level: plan.inputs.assumed_activity ? '' : plan.inputs.activity,
    sex: plan.inputs.sex,
  })
}
export function focusProfileField(fieldId: string) {
  const field = document.getElementById(fieldId)
  if (!field) return
  const details = field.closest('details')
  if (details) details.open = true
  field.scrollIntoView({ behavior: 'smooth', block: 'center' })
  field.querySelector<HTMLElement>('input:not([type="hidden"]), select, button')?.focus({ preventScroll: true })
}
