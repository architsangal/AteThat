// Empty protobuf repeated fields may arrive as null in older API responses.
// A paused schedule deliberately has no times; retain a usable resume default.
export function scheduleTimesText(times?: string[] | null) {
  return times?.length ? times.join(', ') : '08:00'
}
