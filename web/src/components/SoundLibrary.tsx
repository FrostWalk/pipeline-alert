import { useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Music, Upload } from 'lucide-react'
import { piListSounds, piSelectSound, piUploadSound } from '@/client'
import type { SoundInfo } from '@/client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function SoundLibrary() {
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['sounds'],
    queryFn: () => piListSounds({ throwOnError: true }).then((r) => r.data),
  })

  const selectMutation = useMutation({
    mutationFn: (fileName: string) =>
      piSelectSound({ body: { fileName }, throwOnError: true }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['sounds'] }),
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) =>
      piUploadSound({ body: { file }, throwOnError: true }),
    onSuccess: () => {
      setUploadError(null)
      qc.invalidateQueries({ queryKey: ['sounds'] })
    },
    onError: (err: Error) => setUploadError(err.message),
  })

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) uploadMutation.mutate(file)
    e.target.value = ''
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Music className="h-4 w-4" />
            Sound Library
          </CardTitle>
          <Button
            size="sm"
            variant="outline"
            onClick={() => fileRef.current?.click()}
            disabled={uploadMutation.isPending}
          >
            <Upload className="h-3.5 w-3.5 mr-1.5" />
            {uploadMutation.isPending ? 'Uploading…' : 'Upload'}
          </Button>
          <input
            ref={fileRef}
            type="file"
            accept="audio/*"
            className="hidden"
            onChange={handleFileChange}
          />
        </div>
      </CardHeader>
      <CardContent>
        {uploadError && (
          <p className="text-sm text-destructive mb-3">{uploadError}</p>
        )}

        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : !data?.sounds.length ? (
          <p className="text-sm text-muted-foreground">No sounds uploaded yet.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>File</TableHead>
                <TableHead>Size</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="w-24">Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.sounds.map((sound: SoundInfo) => {
                const isSelected = sound.fileName === data.selectedFileName
                return (
                  <TableRow key={sound.fileName} className={isSelected ? 'bg-muted/50' : ''}>
                    <TableCell className="font-mono text-xs">
                      <div className="flex items-center gap-2">
                        {isSelected && <Check className="h-3.5 w-3.5 text-green-500 shrink-0" />}
                        {sound.fileName}
                        {isSelected && <Badge variant="secondary" className="text-xs">active</Badge>}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {formatBytes(sound.sizeBytes)}
                    </TableCell>
                    <TableCell className="text-muted-foreground text-sm">
                      {sound.contentType}
                    </TableCell>
                    <TableCell>
                      {!isSelected && (
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={selectMutation.isPending}
                          onClick={() => selectMutation.mutate(sound.fileName)}
                        >
                          Select
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}
