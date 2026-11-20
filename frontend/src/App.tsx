import { useEffect, useState } from 'react'
import Viewport3D from './components/Viewport3D'
import ImagesPanel from './components/ImagesPanel'

const API = '/api'

type Health = { status: string; time: string }

export default function App() {
  const [health, setHealth] = useState<Health | null>(null)
  const [log, setLog] = useState<string[]>([])

  useEffect(() => {
    fetch(`${API}/health`)
      .then((r) => r.json())
      .then(setHealth)
      .catch((e) => setLog((l) => [...l, `health error: ${e}`]))
    const es = new EventSource(`${API}/events`)
    es.onmessage = (ev) => setLog((l) => [...l.slice(-200), ev.data])
    es.addEventListener('ping', () => {})
    return () => es.close()
  }, [])

  return (
    <div className="flex h-full">
      <aside className="w-64 border-r border-gray-200 dark:border-gray-700 p-4 flex flex-col gap-2">
        <h1 className="text-xl font-bold">PhotoGen</h1>
        <p className="text-sm text-gray-500">Web GUI Photogrammetry</p>
        <nav className="flex flex-col gap-1 mt-4">
          <button className="text-left px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800">Project</button>
          <button className="text-left px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800">Images</button>
          <button className="text-left px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800">Pipeline</button>
          <button className="text-left px-2 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-800">Viewport</button>
        </nav>
        <div className="mt-auto text-xs text-gray-400">
          Backend: {health ? health.status : 'connecting...'}
        </div>
      </aside>
      <main className="flex-1 flex flex-col">
        <header className="h-12 border-b border-gray-200 dark:border-gray-700 flex items-center px-4">
          <span className="font-medium">PhotoGen Web</span>
        </header>
        <section className="flex-1 p-4 grid grid-cols-2 gap-4">
          <div className="border border-gray-200 dark:border-gray-700 rounded p-4">
            <h2 className="font-semibold mb-2">Images</h2>
            <ImagesPanel />
          </div>
          <div className="border border-gray-200 dark:border-gray-700 rounded p-4">
            <h2 className="font-semibold mb-2">3D Viewport</h2>
            <div className="h-64 rounded overflow-hidden">
              <Viewport3D />
            </div>
          </div>
          <div className="border border-gray-200 dark:border-gray-700 rounded p-4 flex flex-col">
            <h2 className="font-semibold mb-2">Log Console</h2>
            <div className="flex-1 overflow-auto text-xs font-mono bg-gray-50 dark:bg-gray-900 rounded p-2">
              {log.map((l, i) => (
                <div key={i}>{l}</div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  )
}
