import { useState } from 'react'

function severityClass(sev) {
  return `severity-badge sev-${sev?.toLowerCase()}`
}

function statusClass(status) {
  return `status-badge status-${status?.replace('-', '-')}`
}

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

export default function TicketCard({ ticket }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="ticket-card" onClick={() => setExpanded((e) => !e)}>
      <div className="ticket-card-header">
        <span className={severityClass(ticket.severity)}>{ticket.severity}</span>
        <span className="ticket-id mono">{ticket.id}</span>
        <span className={statusClass(ticket.status)}>{ticket.status}</span>
      </div>

      <div className="ticket-title">{ticket.title}</div>

      <div className="ticket-meta">
        <span>{ticket.mission}</span>
        <span>{ticket.category}</span>
        {ticket.reported_by && <span>{ticket.reported_by}</span>}
        {ticket.assigned_to && ticket.assigned_to !== 'ground-control' && (
          <span>→ {ticket.assigned_to}</span>
        )}
      </div>

      {expanded && (
        <div className="ticket-body" onClick={(e) => e.stopPropagation()}>
          {ticket.description && (
            <p className="ticket-description">{ticket.description}</p>
          )}

          {ticket.resolution && (
            <div className="ticket-resolution">
              <strong>Resolution:</strong> {ticket.resolution}
            </div>
          )}

          {ticket.kb_articles_referenced?.length > 0 && (
            <div className="tags" style={{ marginBottom: 8 }}>
              {ticket.kb_articles_referenced.map((id) => (
                <span key={id} className="kb-chip">
                  {id}
                </span>
              ))}
            </div>
          )}

          {ticket.comments?.length > 0 && (
            <div className="ticket-comments">
              <div className="ticket-comments-label">
                Comments ({ticket.comments.length})
              </div>
              {ticket.comments.map((c, i) => (
                <div key={i} className="comment">
                  <div className="comment-meta">
                    <span className="comment-author">{c.author}</span>
                    <span>{formatTime(c.timestamp)}</span>
                  </div>
                  <div className="comment-text">{c.text}</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
