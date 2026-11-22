const API = '/api'

export async function ingest(inputDir: string) {
  const r = await fetch(`${API}/ingest`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input_dir: inputDir }),
  })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}

export async function runFeatures(inputDir: string, detector: string, workers: number) {
  const r = await fetch(`${API}/features/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input_dir: inputDir, detector, workers }),
  })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}

export async function listDir(path: string) {
  const r = await fetch(`${API}/fs/list?path=${encodeURIComponent(path)}`)
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}

export async function runSfM(inputDir: string) {
  const r = await fetch(`${API}/sfm/init`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input_dir: inputDir }),
  })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}

export async function runMVS(inputDir: string) {
  const r = await fetch(`${API}/mvs/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input_dir: inputDir }),
  })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}

export async function runMesh(inputDir: string) {
  const r = await fetch(`${API}/mesh/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ input_dir: inputDir }),
  })
  if (!r.ok) throw new Error(await r.text())
  return r.json()
}
