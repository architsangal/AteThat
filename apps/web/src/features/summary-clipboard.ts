// Keep permission failure handling independent of the browser permission state.
export async function copySummaryText(text: string, write: (text: string) => Promise<void>, selectFallback: () => void): Promise<string> {
  try {
    await write(text)
    return 'Full edited summary copied.'
  } catch {
    selectFallback()
    return 'Clipboard unavailable. Text selected; use your browser’s Copy command.'
  }
}
