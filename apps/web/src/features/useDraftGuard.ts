import { useEffect, useRef } from 'react'
import { useBlocker } from 'react-router-dom'
import { useDialogs } from './ui'
import { getAccessToken } from '../session'

export function useDraftGuard(dirty: boolean, description: string) {
  // An expired/logged-out session must never be held behind an unsaved-data
  // dialog. The session-expired listener clears all in-memory summary content.
  const blocker = useBlocker(() => dirty && !!getAccessToken())
  const dialogs = useDialogs()
  const pending = useRef(false)
  useEffect(() => {
    if (blocker.state !== 'blocked' || pending.current) return
    pending.current = true
    void dialogs.confirm(description).then(ok => {
      pending.current = false
      if (ok) blocker.proceed?.(); else blocker.reset?.()
    })
  }, [blocker, dialogs, description])
  useEffect(() => {
    if (!dirty) return
    const before = (event: BeforeUnloadEvent) => { event.preventDefault(); event.returnValue = '' }
    window.addEventListener('beforeunload', before)
    return () => window.removeEventListener('beforeunload', before)
  }, [dirty])
}
