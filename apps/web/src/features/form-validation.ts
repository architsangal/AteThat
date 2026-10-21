// Required/empty values are handled separately by Form's required check.
// Trimming also validates programmatically populated values, for which native
// minlength validity is not reliably reported by browsers.
export function isTextTooShort(value: string, minLength: number): boolean {
  return value.length > 0 && value.trim().length < minLength
}
