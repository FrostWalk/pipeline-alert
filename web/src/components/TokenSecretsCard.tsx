import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Check, Copy, Eye, EyeOff, KeyRound } from 'lucide-react'
import { integrationGetTokens } from '@/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'

function maskValue(value: string): string {
  const len = Math.min(value.length, 32)
  return '•'.repeat(Math.max(len, 8))
}

type TokenFieldProps = {
  id: string
  label: string
  description: string
  value: string
}

function TokenField({ id, label, description, value }: TokenFieldProps) {
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div className="space-y-2">
      <div>
        <Label htmlFor={id}>{label}</Label>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>
      <div className="flex items-center gap-2">
        <code
          id={id}
          className="flex-1 min-w-0 rounded-md border border-border bg-muted/50 px-3 py-2 text-xs font-mono break-all"
        >
          {revealed ? value : maskValue(value)}
        </code>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="shrink-0"
          onClick={() => setRevealed((v) => !v)}
          aria-label={revealed ? `Hide ${label}` : `Reveal ${label}`}
        >
          {revealed ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="shrink-0"
          onClick={() => void handleCopy()}
          aria-label={`Copy ${label}`}
        >
          {copied ? <Check className="h-4 w-4 text-green-500" /> : <Copy className="h-4 w-4" />}
        </Button>
      </div>
    </div>
  )
}

export function TokenSecretsCard() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['integration-tokens'],
    queryFn: () => integrationGetTokens({ throwOnError: true }).then((r) => r.data),
  })

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="h-4 w-4" />
            Integration Tokens
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">Loading…</p>
        </CardContent>
      </Card>
    )
  }

  if (isError || !data) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="h-4 w-4" />
            Integration Tokens
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-destructive">Failed to load tokens</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="h-4 w-4" />
          Integration Tokens
        </CardTitle>
        <CardDescription>
          GitLab webhook secret and Pi websocket Bearer token. Masked by default; reveal only when needed.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <TokenField
          id="webhook-secret"
          label="GitLab webhook token"
          description="Set as GitLab group webhook secret; sent in TOKEN_HEADER (default X-Gitlab-Token)."
          value={data.webhookSecret}
        />
        <TokenField
          id="websocket-secret"
          label="WebSocket token"
          description="Pi client uses Authorization: Bearer on GET /ws."
          value={data.websocketSecret}
        />
      </CardContent>
    </Card>
  )
}
