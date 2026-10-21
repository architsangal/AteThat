// TASK shares the existing occurrence lifecycle without changing wire statuses.
// Only its user-facing completion language differs from medication/supplements.
export function healthStatusLabel(status: string, type?: string): string {
  return status === 'TAKEN' && type === 'TASK' ? 'done' : status.toLowerCase()
}
