import { useEffect, useRef, useState } from 'react'
import type { PiLogEvent, ServerLogEvent } from '@/client'

type LogEntry = ServerLogEvent | PiLogEvent
export type StreamStatus = 'connecting' | 'open' | 'closed' | 'error'

const MAX_ENTRIES = 500

export function useLogStream(url: string, token: string | null) {
  const [entries, setEntries] = useState<LogEntry[]>([])
  const [status, setStatus] = useState<StreamStatus>('connecting')
  const esRef = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!token) {
      setStatus('closed')
      return
    }

    const fullUrl = `${url}?accessToken=${encodeURIComponent(token)}`
    const es = new EventSource(fullUrl)
    esRef.current = es

    setStatus('connecting')

    es.onopen = () => setStatus('open')

    es.addEventListener('log', (e) => {
      try {
        const data = JSON.parse((e as MessageEvent).data) as LogEntry
        setEntries((prev) => {
          const next = [...prev, data]
          return next.length > MAX_ENTRIES ? next.slice(next.length - MAX_ENTRIES) : next
        })
      } catch {
        // ignore malformed events
      }
    })

    es.onerror = () => {
      setStatus('error')
      es.close()
    }

    return () => {
      es.close()
      esRef.current = null
    }
  }, [url, token])

  const clear = () => setEntries([])

  return { entries, status, clear }
}
