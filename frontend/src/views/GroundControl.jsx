import { useState, useEffect, useContext } from 'react'
import { UserContext } from '../UserContext'
import TicketCard from '../components/TicketCard'
import ArticleCard from '../components/ArticleCard'
import CrewRoster from '../components/CrewRoster'
import { fetchTickets, fetchArticles } from '../api'

// ─── Ticket Panel ────────────────────────────────────────────
function TicketPanel() {
  const [tickets, setTickets] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [severity, setSeverity] = useState('')
  const [status, setStatus] = useState('')
  const [mission, setMission] = useState('')

  function load(filters = {}) {
    setLoading(true)
    setError(null)
    fetchTickets({ severity, status, mission, limit: 100, ...filters })
      .then((data) => {
        setTickets(data.tickets ?? [])
        setTotal(data.total ?? 0)
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function handleApply(e) {
    e.preventDefault()
    load()
  }

  function handleReset() {
    setSeverity('')
    setStatus('')
    setMission('')
    load({ severity: '', status: '', mission: '' })
  }

  return (
    <div className="gc-panel">
      <div className="gc-panel-header">
        <div className="gc-panel-title">Ticket Queue {total > 0 && `(${total})`}</div>
        <form className="filter-bar" onSubmit={handleApply}>
          <select value={severity} onChange={(e) => setSeverity(e.target.value)}>
            <option value="">All severities</option>
            <option value="P1">P1 — Critical</option>
            <option value="P2">P2 — Major</option>
            <option value="P3">P3 — Minor</option>
            <option value="P4">P4 — Info</option>
          </select>
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="">All statuses</option>
            <option value="open">Open</option>
            <option value="in-progress">In Progress</option>
            <option value="resolved">Resolved</option>
            <option value="closed">Closed</option>
          </select>
          <select value={mission} onChange={(e) => setMission(e.target.value)}>
            <option value="">All missions</option>
            <option value="artemis-ii">Artemis II</option>
            <option value="artemis-iii">Artemis III</option>
            <option value="artemis-iv">Artemis IV</option>
          </select>
          <button type="submit" className="filter-btn">Filter</button>
          <button type="button" className="filter-btn" onClick={handleReset}
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border)' }}>
            Clear
          </button>
        </form>
      </div>

      <div className="gc-panel-body">
        {loading && <div className="state-loading">Loading tickets...</div>}
        {error && <div className="state-error">Failed to load tickets: {error}</div>}
        {!loading && !error && tickets.length === 0 && (
          <div className="state-empty">No tickets match the current filters.</div>
        )}
        {tickets.map((t) => (
          <TicketCard key={t.id} ticket={t} />
        ))}
      </div>
    </div>
  )
}

// ─── KB Panel ────────────────────────────────────────────────
const CATEGORIES = [
  'life-support', 'navigation', 'comms', 'power', 'propulsion',
  'eva', 'medical', 'operations', 'thermal', 'structures',
]

function KBPanel() {
  const [articles, setArticles] = useState([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const [search, setSearch] = useState('')
  const [category, setCategory] = useState('')

  function load(overrides = {}) {
    setLoading(true)
    setError(null)
    fetchArticles({ search, category, limit: 100, ...overrides })
      .then((data) => {
        setArticles(data.articles ?? [])
        setTotal(data.total ?? 0)
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function handleSearch(e) {
    e.preventDefault()
    load()
  }

  function handleCategoryChange(e) {
    const cat = e.target.value
    setCategory(cat)
    load({ category: cat })
  }

  function handleClear() {
    setSearch('')
    setCategory('')
    load({ search: '', category: '' })
  }

  return (
    <div className="gc-panel">
      <div className="gc-panel-header">
        <div className="gc-panel-title">Knowledge Base {total > 0 && `(${total})`}</div>
        <form className="filter-bar" onSubmit={handleSearch}>
          <input
            type="text"
            placeholder="Search articles..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <select value={category} onChange={handleCategoryChange}>
            <option value="">All categories</option>
            {CATEGORIES.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
          <button type="submit" className="filter-btn">Search</button>
          <button type="button" className="filter-btn" onClick={handleClear}
            style={{ background: 'var(--bg-elevated)', border: '1px solid var(--border)' }}>
            Clear
          </button>
        </form>
      </div>

      <div className="gc-panel-body">
        {loading && <div className="state-loading">Loading articles...</div>}
        {error && <div className="state-error">Failed to load articles: {error}</div>}
        {!loading && !error && articles.length === 0 && (
          <div className="state-empty">No articles found.</div>
        )}
        {articles.map((a) => (
          <ArticleCard key={a.id} article={a} />
        ))}
      </div>
    </div>
  )
}

// ─── Crew Panel ──────────────────────────────────────────────
function CrewPanel() {
  const { crew } = useContext(UserContext)

  return (
    <div className="gc-panel">
      <div className="gc-panel-header">
        <div className="gc-panel-title">Crew Activity</div>
      </div>
      <div className="gc-panel-body">
        <CrewRoster crew={crew} />
      </div>
    </div>
  )
}

// ─── Ground Control view ─────────────────────────────────────
export default function GroundControl() {
  return (
    <div className="gc-layout">
      <TicketPanel />
      <KBPanel />
      <CrewPanel />
    </div>
  )
}
