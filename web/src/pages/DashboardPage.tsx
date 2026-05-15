import { LogOut, Server, Cpu } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { LogStream } from '@/components/LogStream'
import { PiStatusCard } from '@/components/PiStatusCard'
import { SoundLibrary } from '@/components/SoundLibrary'
import { TokenSecretsCard } from '@/components/TokenSecretsCard'
import { useLogStream } from '@/hooks/useLogStream'
import { useAuthStore } from '@/lib/auth-store'

export function DashboardPage() {
  const navigate = useNavigate()
  const token = useAuthStore((s) => s.token)
  const clearToken = useAuthStore((s) => s.clearToken)

  const serverLogs = useLogStream('/api/logs/server/stream', token)
  const piLogs = useLogStream('/api/logs/pi/stream', token)

  const handleLogout = () => {
    clearToken()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border bg-card">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-14 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-xl">📯</span>
            <h1 className="font-semibold text-lg">Pipeline Horn</h1>
          </div>
          <Button variant="ghost" size="sm" onClick={handleLogout}>
            <LogOut className="h-4 w-4 mr-1.5" />
            Sign out
          </Button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
        <TokenSecretsCard />

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PiStatusCard />
          <SoundLibrary />
        </div>

        <LogStream
          title="Server Logs"
          icon={<Server className="h-4 w-4" />}
          entries={serverLogs.entries}
          status={serverLogs.status}
          onClear={serverLogs.clear}
        />

        <LogStream
          title="Pi Logs"
          icon={<Cpu className="h-4 w-4" />}
          entries={piLogs.entries}
          status={piLogs.status}
          onClear={piLogs.clear}
        />
      </main>
    </div>
  )
}
