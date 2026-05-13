import { useEffect, useRef } from 'react'
import type { PiLogEvent, ServerLogEvent } from '@/client'
import type { StreamStatus } from '@/hooks/useLogStream'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

type LogEntry = ServerLogEvent | PiLogEvent

const levelColors: Record<string, string> = {
  debug: 'text-slate-400',
  info: 'text-blue-400',
  warn: 'text-yellow-400',
  warning: 'text-yellow-400',
  error: 'text-red-400',
  fatal: 'text-red-600 font-bold',
  dpanic: 'text-red-500',
  panic: 'text-red-600 font-bold',
}

function StatusDot({ status }: { status: StreamStatus }) {
  const colors: Record<StreamStatus, string> = {
    connecting: 'bg-yellow-400 animate-pulse',
    open: 'bg-green-400',
    closed: 'bg-slate-400',
    error: 'bg-red-400',
  }
  return <span className={cn('inline-block h-2 w-2 rounded-full', colors[status])} />
}

function LogRow({ entry }: { entry: LogEntry }) {
  const ts = new Date(entry.timestamp).toLocaleTimeString()
  const colorClass = levelColors[entry.level?.toLowerCase() ?? ''] ?? 'text-foreground'
  const eventType = 'eventType' in entry ? entry.eventType : undefined

  return (
    <div className="flex gap-2 font-mono text-xs py-0.5 hover:bg-muted/40 px-1 rounded">
      <span className="text-muted-foreground shrink-0 w-20">{ts}</span>
      <span className={cn('shrink-0 w-12 uppercase', colorClass)}>{entry.level}</span>
      {eventType && (
        <span className="text-purple-400 shrink-0">[{eventType}]</span>
      )}
      <span className="text-foreground break-all">{entry.message}</span>
    </div>
  )
}

interface Props {
  title: string
  icon: React.ReactNode
  entries: LogEntry[]
  status: StreamStatus
  onClear: () => void
}

export function LogStream({ title, icon, entries, status, onClear }: Props) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2 text-base">
            {icon}
            {title}
            <StatusDot status={status} />
          </CardTitle>
          <Button size="sm" variant="ghost" onClick={onClear} className="text-xs h-7">
            Clear
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <ScrollArea className="h-64 px-3 pb-3">
          {entries.length === 0 ? (
            <p className="text-xs text-muted-foreground py-4 text-center">Waiting for log events…</p>
          ) : (
            entries.map((entry, i) => <LogRow key={i} entry={entry} />)
          )}
          <div ref={bottomRef} />
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
