import { useState, useEffect, useCallback } from 'react'
import { App } from '../../../bindings/monika'
import { IconDatabase, IconRefresh, IconZap, IconPlus, IconChevronDown, IconChevronRight } from '../Icons'

type DBConn = {
  name: string
  driver: string
  source: string
  status: string
  error?: string
}

const DRIVERS = ['postgres', 'mysql', 'sqlite', 'redis', 'mongo']

function statusDot(status: string) {
  if (status === 'connected') return <span className="inline-block w-1.5 h-1.5 rounded-full bg-green-500" />
  if (status === 'error' || status === 'unavailable') return <span className="inline-block w-1.5 h-1.5 rounded-full bg-red-500" />
  return <span className="inline-block w-1.5 h-1.5 rounded-full bg-yellow-500" />
}

function statusLabel(status: string) {
  if (status === 'connected') return <span className="text-green-400">connected</span>
  if (status === 'error') return <span className="text-red-400">error</span>
  if (status === 'unavailable') return <span className="text-red-400">unavailable</span>
  return <span className="text-yellow-400">available</span>
}

function ConnectionCard({ conn, onTest, testState }: {
  conn: DBConn
  onTest: () => void
  testState: { loading: boolean; result?: string; error?: string }
}) {
  return (
    <div
      className="rounded-lg px-4 py-3 w-full relative group/card"
      style={{ background: 'var(--bg-card)' }}
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5 shrink-0" style={{ color: 'var(--text-dim)' }}>
          <IconDatabase size={16} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-0.5">
            <span className="font-mono text-[14px] font-semibold text-[var(--text-primary)]">{conn.name}</span>
            <span
              className="text-[10px] px-1.5 py-0.5 rounded-sm font-medium"
              style={{
                background: 'var(--bg-input)',
                border: '1px solid var(--border)',
                color: 'var(--text-dim)',
              }}
            >
              {conn.driver}
            </span>
            <span className="inline-flex items-center gap-1 text-[10px]">
              {statusDot(conn.status)}
              {statusLabel(conn.status)}
            </span>
          </div>
          <div className="flex flex-wrap items-center gap-3 text-[11px] text-[var(--text-dim)]">
            <span className="font-mono text-[11px]">{conn.source}</span>
          </div>
          {conn.error && (
            <div className="mt-1 text-[10px] text-[var(--red)]">{conn.error}</div>
          )}
          {testState.result && (
            <div className="mt-1 text-[10px] text-green-400">{testState.result}</div>
          )}
          {testState.error && (
            <div className="mt-1 text-[10px] text-[var(--red)]">{testState.error}</div>
          )}
        </div>
        <div className="opacity-0 group-hover/card:opacity-100 transition-opacity flex gap-1 shrink-0">
          <button
            onClick={onTest}
            disabled={testState.loading}
            title="Test connection"
            className="inline-flex items-center text-[var(--text-dim)] hover:text-[var(--accent)] text-[11px] px-1.5 py-0.5 cursor-pointer bg-transparent border-none rounded transition-colors"
            aria-label={`Test ${conn.name}`}
          >
            {testState.loading ? <IconRefresh size={14} /> : <IconZap size={14} />}
          </button>
        </div>
      </div>
    </div>
  )
}

export default function DatabasesTab() {
  const [connections, setConnections] = useState<DBConn[]>([])
  const [loading, setLoading] = useState(false)
  const [testStates, setTestStates] = useState<Record<string, { loading: boolean; result?: string; error?: string }>>({})
  const [showAdd, setShowAdd] = useState(false)
  const [addName, setAddName] = useState('')
  const [addDriver, setAddDriver] = useState('postgres')
  const [addDSN, setAddDSN] = useState('')
  const [addError, setAddError] = useState('')

  const loadConnections = useCallback(async () => {
    try {
      const conns = await App.ListDatabaseConnections() as unknown as DBConn[]
      setConnections(conns || [])
    } catch {
      setConnections([])
    }
  }, [])

  useEffect(() => { loadConnections() }, [loadConnections])

  const handleRescan = useCallback(async () => {
    setLoading(true)
    try {
      const conns = await App.RescanDatabases() as unknown as DBConn[]
      setConnections(conns || [])
    } catch (e: any) {
      setConnections([])
    } finally {
      setLoading(false)
    }
  }, [])

  const handleTest = useCallback(async (name: string) => {
    setTestStates((s) => ({ ...s, [name]: { loading: true } }))
    try {
      await App.TestDatabaseConnection(JSON.stringify({ Name: name }))
      setTestStates((s) => ({ ...s, [name]: { loading: false, result: 'Connection OK' } }))
    } catch (e: any) {
      setTestStates((s) => ({ ...s, [name]: { loading: false, error: e?.message || 'Connection failed' } }))
    }
  }, [])

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h3 className="text-[15px] font-semibold m-0 mb-1">Databases</h3>
          <p className="text-[11px] text-[var(--text-dim)] m-0">Manage discovered database connections</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setShowAdd((v) => !v)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
          >
            {showAdd ? <IconChevronDown size={12} /> : <IconChevronRight size={12} />}
            Add
          </button>
          <button
            onClick={handleRescan}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors"
          >
            <IconRefresh size={12} />
            {loading ? 'Scanning...' : 'Rescan Project'}
          </button>
        </div>
      </div>

      {showAdd && (
        <div
          className="rounded-lg px-4 py-3 mb-4 space-y-3"
          style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
        >
          <div className="flex items-center gap-3">
            <input
              type="text"
              value={addName}
              onChange={(e) => { setAddName(e.target.value); setAddError('') }}
              placeholder="Connection name"
              className="flex-1 text-[12px] font-mono px-3 py-1.5 rounded-md outline-none focus:border-[var(--border-strong)] form-input-glow transition-colors duration-150"
              style={{
                background: 'var(--bg-console)',
                border: '1px solid var(--border)',
                color: 'var(--text-primary)',
              }}
            />
            <select
              value={addDriver}
              onChange={(e) => setAddDriver(e.target.value)}
              className="text-[12px] px-3 py-1.5 rounded-md outline-none cursor-pointer"
              style={{
                background: 'var(--bg-console)',
                border: '1px solid var(--border)',
                color: 'var(--text-primary)',
              }}
            >
              {DRIVERS.map((d) => (
                <option key={d} value={d}>{d}</option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-3">
            <input
              type="text"
              value={addDSN}
              onChange={(e) => { setAddDSN(e.target.value); setAddError('') }}
              placeholder="DSN (e.g. postgres://user:pass@localhost:5432/dbname)"
              className="flex-1 text-[12px] font-mono px-3 py-1.5 rounded-md outline-none focus:border-[var(--border-strong)] form-input-glow transition-colors duration-150"
              style={{
                background: 'var(--bg-console)',
                border: '1px solid var(--border)',
                color: 'var(--text-primary)',
              }}
            />
            <button
              onClick={handleRescan}
              disabled={!addName || !addDSN}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-[12px] font-medium rounded border border-[var(--border-strong)] bg-[var(--bg-elevated)] text-[var(--text-primary)] cursor-pointer hover:bg-[var(--bg-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <IconPlus size={12} />
              Add
            </button>
          </div>
          {addError && (
            <p className="text-[11px] text-[var(--red)] m-0">{addError}</p>
          )}
        </div>
      )}

      {connections.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-[var(--text-dim)]">
          <IconDatabase size={32} />
          <span className="text-[13px] mt-3">No databases discovered.</span>
          <span className="text-[11px] mt-1">Click "Rescan Project" to detect databases.</span>
        </div>
      ) : (
        <div className="space-y-3">
          {connections.map((conn) => (
            <ConnectionCard
              key={conn.name}
              conn={conn}
              onTest={() => handleTest(conn.name)}
              testState={testStates[conn.name] || { loading: false }}
            />
          ))}
        </div>
      )}
    </div>
  )
}
