import ReactMarkdown from 'react-markdown'

export default function ChatMessage({ message }) {
  const { role, content, kbRefs, ticketCreated } = message

  if (role === 'user') {
    return (
      <div className="chat-message user">
        <div className="message-bubble user-bubble">{content}</div>
      </div>
    )
  }

  return (
    <div className="chat-message assistant">
      <div className="message-bubble assistant-bubble">
        <ReactMarkdown>{content}</ReactMarkdown>

        {kbRefs && kbRefs.length > 0 && (
          <div className="message-refs">
            {kbRefs.map((id) => (
              <span key={id} className="kb-chip">
                {id}
              </span>
            ))}
          </div>
        )}

        {ticketCreated && (
          <div>
            <span className="ticket-chip">Ticket created: {ticketCreated}</span>
          </div>
        )}
      </div>
    </div>
  )
}
