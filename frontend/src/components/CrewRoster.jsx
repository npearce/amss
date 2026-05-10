import { useState } from 'react'
import { fetchCrewActivity } from '../api'

function formatTime(ts) {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return ts
  }
}

function MemberRow({ member }) {
  const [expanded, setExpanded] = useState(false)
  const [activity, setActivity] = useState(null)
  const [loading, setLoading] = useState(false)

  function toggle() {
    if (expanded) {
      setExpanded(false)
      return
    }
    setExpanded(true)
    if (activity) return
    setLoading(true)
    fetchCrewActivity(member.id, { limit: 10 })
      .then((data) => setActivity(data))
      .catch(() => setActivity({ conversations: [], total: 0 }))
      .finally(() => setLoading(false))
  }

  return (
    <div
      className={`crew-member ${expanded ? 'selected' : ''}`}
      onClick={toggle}
    >
      <div className="crew-name">{member.name}</div>
      <div className="crew-meta">
        <span>{member.role}</span>
        {member.mission !== 'all' && (
          <span className="mission-badge">{member.mission}</span>
        )}
        {member.status !== 'active' && (
          <span className="text-dim">{member.status}</span>
        )}
      </div>

      {expanded && (
        <div className="crew-activity">
          {loading ? (
            <div className="text-dim" style={{ fontSize: 12 }}>Loading...</div>
          ) : activity ? (
            <>
              <div className="activity-count">
                {activity.total ?? 0} conversation{activity.total !== 1 ? 's' : ''}
              </div>
              {(activity.conversations ?? []).slice(0, 5).map((conv, i) => (
                <div key={conv.id ?? i} className="activity-entry">
                  <div className="activity-query">{conv.query}</div>
                  <div className="activity-time">{formatTime(conv.timestamp)}</div>
                </div>
              ))}
              {(activity.conversations ?? []).length === 0 && (
                <div className="text-dim" style={{ fontSize: 12 }}>No conversations yet.</div>
              )}
            </>
          ) : (
            <div className="text-dim" style={{ fontSize: 12 }}>No activity available.</div>
          )}
        </div>
      )}
    </div>
  )
}

export default function CrewRoster({ crew }) {
  const astronauts = crew.filter((c) => c.persona === 'astronaut')
  const groundControl = crew.filter((c) => c.persona === 'ground-control')

  if (crew.length === 0) {
    return <div className="state-loading">Loading crew...</div>
  }

  return (
    <div>
      <div className="crew-group">
        <div className="crew-group-label">Astronauts</div>
        {astronauts.map((m) => (
          <MemberRow key={m.id} member={m} />
        ))}
      </div>
      <div className="crew-group">
        <div className="crew-group-label">Ground Control</div>
        {groundControl.map((m) => (
          <MemberRow key={m.id} member={m} />
        ))}
      </div>
    </div>
  )
}
