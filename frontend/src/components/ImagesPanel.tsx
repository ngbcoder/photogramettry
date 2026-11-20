import { useState } from 'react'
import { ingest, runFeatures } from '../api'

type Frame = { id: number; path: string; width: number; height: number }

export default function ImagesPanel() {
  const [inputDir, setInputDir] = useState('')
  const [frames, setFrames] = useState<Frame[]>([])
  const [busy, setBusy] = useState(false)
  const [status, setStatus] = useState('')

  const onIngest = async () => {
    setBusy(true)
    setStatus('ingesting...')
    try {
      const res = await ingest(inputDir)
      setFrames(res.frames || [])
      setStatus(`ingested ${res.count} images`)
    } catch (e) {
      setStatus(`error: ${e}`)
    } finally {
      setBusy(false)
    }
  }

  const onFeatures = async () => {
    setBusy(true)
    setStatus('running features...')
    try {
      const res = await runFeatures(inputDir, 'akaze', 4)
      setStatus(`features done: ${res.count}`)
    } catch (e) {
      setStatus(`error: ${e}`)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        <input
          className="flex-1 border rounded px-2 py-1 text-sm"
          placeholder="Input directory path"
          value={inputDir}
          onChange={(e) => setInputDir(e.target.value)}
        />
        <button
          className="px-3 py-1 rounded bg-blue-600 text-white text-sm disabled:opacity-50"
          onClick={onIngest}
          disabled={busy || !inputDir}
        >
          Ingest
        </button>
        <button
          className="px-3 py-1 rounded bg-green-600 text-white text-sm disabled:opacity-50"
          onClick={onFeatures}
          disabled={busy || frames.length === 0}
        >
          Features
        </button>
      </div>
      <div className="text-sm text-gray-500">{status}</div>
      <div className="grid grid-cols-4 gap-2 overflow-auto max-h-96">
        {frames.map((f) => (
          <div key={f.id} className="border rounded p-1 text-xs">
            <div className="truncate" title={f.path}>{f.path.split('/').pop()}</div>
            <div className="text-gray-400">{f.width}x{f.height}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
