import { useQuery } from '@tanstack/react-query'
import { Wifi, WifiOff, Clock, Radio } from 'lucide-react'
import { piGetStatus } from '@/client'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

function formatRelative(iso: string): string {
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

export function PiStatusCard() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['pi-status'],
    queryFn: () => piGetStatus({ throwOnError: true }).then((r) => r.data),
    refetchInterval: 5000,
  })

  if (isLoading) {
    return (
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Radio className="h-4 w-4" />Pi Status</CardTitle></CardHeader>
        <CardContent><p className="text-sm text-muted-foreground">Loading…</p></CardContent>
      </Card>
    )
  }

  if (isError || !data) {
    return (
      <Card>
        <CardHeader><CardTitle className="flex items-center gap-2"><Radio className="h-4 w-4" />Pi Status</CardTitle></CardHeader>
        <CardContent><p className="text-sm text-destructive">Failed to load status</p></CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Radio className="h-4 w-4" />
          Pi Status
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center gap-2">
          {data.isConnected ? (
            <>
              <Wifi className="h-4 w-4 text-green-500" />
              <Badge variant="default" className="bg-green-500 hover:bg-green-600">Connected</Badge>
            </>
          ) : (
            <>
              <WifiOff className="h-4 w-4 text-muted-foreground" />
              <Badge variant="secondary">Disconnected</Badge>
            </>
          )}
        </div>

        {data.connectedSince && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span>Connected {formatRelative(data.connectedSince)}</span>
          </div>
        )}

        {data.lastSeenAt && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Clock className="h-3.5 w-3.5" />
            <span>Last seen {formatRelative(data.lastSeenAt)}</span>
          </div>
        )}

        <div className="text-sm">
          <span className="text-muted-foreground">Active sound: </span>
          {data.selectedFileName ? (
            <span className="font-mono text-xs">{data.selectedFileName}</span>
          ) : (
            <span className="text-muted-foreground">default fallback</span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
