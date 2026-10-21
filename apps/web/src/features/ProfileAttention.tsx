import { focusProfileField, type MissingProfileDetail } from './profile-completion'

export function ProfileAttention({ fields }: { fields: MissingProfileDetail[] }) {
  if (!fields.length) return null
  return <section className="panel profile-attention" id="profile-attention" tabIndex={-1} aria-labelledby="profile-attention-title">
    <h2 id="profile-attention-title">{fields.length} optional profile {fields.length === 1 ? 'detail is' : 'details are'} not provided</h2>
    <p>This is what the number beside Settings means. Choose a detail below to go straight to its field. You can continue logging without providing these details.</p>
    <ul>{fields.map(field => <li key={field.key}><button className="quiet-link" type="button" onClick={() => focusProfileField(field.fieldId)}>Review {field.label.toLowerCase()} <span aria-hidden="true">→</span></button><p>{field.hint}</p></li>)}</ul>
    <p className="muted">Use <strong>Save profile</strong> after making a change. The badge and these labels update after your choice is saved.</p>
  </section>
}
export function ProfileFieldHint({ field }: { field?: MissingProfileDetail }) {
  if (!field) return null
  return <small id={`${field.fieldId}-hint`} className="profile-field-hint"><strong>Not provided · optional</strong><span>{field.hint}</span></small>
}
