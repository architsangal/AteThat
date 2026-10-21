const decimal = /^(0|[1-9][0-9]*)(\.[0-9]+)?$/
export function validTarget(value: string, energy = false) {
  return value.length <= 32 && decimal.test(value) && (!energy || /[1-9]/.test(value))
}
export function calorieDifference(left: string, right: string): string | null {
  if (!validTarget(left) || !validTarget(right)) return null
  const scale = Math.max(left.split('.')[1]?.length || 0, right.split('.')[1]?.length || 0)
  const integer = (v: string) => { const [whole, fraction = ''] = v.split('.'); return BigInt(whole + fraction.padEnd(scale, '0')) }
  const difference = integer(left) - integer(right)
  const absolute = (difference < 0n ? -difference : difference).toString().padStart(scale + 1, '0')
  return scale ? `${absolute.slice(0, -scale)}.${absolute.slice(-scale)}`.replace(/\.?0+$/, '') || '0' : absolute
}
export function exceedsCalorieTolerance(left: string, right: string) {
  const difference = calorieDifference(left, right)
  if (difference === null) return false
  const [whole, fraction = ''] = difference.split('.')
  return BigInt(whole) > 1n || (whole === '1' && /[1-9]/.test(fraction))
}
