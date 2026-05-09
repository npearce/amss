import { useState, useEffect, useRef, useContext } from 'react'
import { UserContext } from '../UserContext'
import ChatMessage from '../components/ChatMessage'
import { fetchConversations, sendChat } from '../api'

function makeId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function convToMessages(conversations) {
  const msgs = []
  // conversations arrive newest-first; reverse to show oldest first
  const sorted = [...conversations].reverse()
  for (const conv of sorted) {
    msgs.push({ role: 'user', content: conv.query, id: conv.id + '-q' })
    msgs.push({
      role: 'assistant',
      content: conv.response,
      kbRefs: conv.kb_articles_referenced ?? [],
      ticketCreated: conv.ticket_created ?? null,
      id: conv.id + '-r',
    })
  }
  return msgs
}

export default function AstronautChat() {
  const { currentUser } = useContext(UserContext)
  const [messages, setMessages] = useState([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [historyLoaded, setHistoryLoaded] = useState(false)
  // One session ID per page load
  const sessionId = useRef(crypto.randomUUID()).current
  const bottomRef = useRef(null)
  const prevUserId = useRef(null)

  // Reload history when the selected user changes
  useEffect(() => {
    if (!currentUser) return
    if (prevUserId.current === currentUser.id) return
    prevUserId.current = currentUser.id
    setMessages([])
    setHistoryLoaded(false)

    fetchConversations({ crew_id: currentUser.id, limit: 20 })
      .then((data) => {
        const convs = data.conversations ?? []
        setMessages(convToMessages(convs))
      })
      .catch(() => {
        // BFF not running — start with empty chat
      })
      .finally(() => setHistoryLoaded(true))
  }, [currentUser])

  // Scroll to bottom when messages change
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  async function handleSend(e) {
    e.preventDefault()
    const text = input.trim()
    if (!text || !currentUser || loading) return

    setInput('')
    const userMsgId = makeId()
    setMessages((prev) => [...prev, { role: 'user', content: text, id: userMsgId }])
    setLoading(true)

    try {
      const data = await sendChat({
        crew_id: currentUser.id,
        session_id: sessionId,
        mission: currentUser.mission,
        message: text,
      })
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: data.response,
          kbRefs: data.kb_articles_referenced ?? [],
          ticketCreated: data.ticket_created ?? null,
          id: makeId(),
        },
      ])
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: `**Error:** ${err.message}`,
          kbRefs: [],
          ticketCreated: null,
          id: makeId(),
        },
      ])
    } finally {
      setLoading(false)
    }
  }

  function handleKeyDown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      handleSend(e)
    }
  }

  if (!currentUser) {
    return <div className="state-loading">Loading crew...</div>
  }

  return (
    <div className="chat-view">
      <div className="chat-header">
        <span className="chat-user">{currentUser.name}</span>
        <span className="chat-role">{currentUser.role}</span>
        <span className="chat-mission">
          {currentUser.mission === 'all' ? 'Ground Control' : currentUser.mission}
        </span>
      </div>

      <div className="chat-messages">
        {historyLoaded && messages.length === 0 && !loading && (
          <div className="chat-empty">
            <p>No conversation history.</p>
            <p style={{ marginTop: 8, fontSize: 12, color: 'var(--text-dim)' }}>
              Ask about spacecraft systems, procedures, or request support.
            </p>
          </div>
        )}

        {messages.map((msg) => (
          <ChatMessage key={msg.id} message={msg} />
        ))}

        {loading && (
          <div className="chat-message assistant">
            <div className="message-bubble assistant-bubble loading-bubble">
              <div className="dot-pulse">
                <span />
              </div>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      <form className="chat-input-bar" onSubmit={handleSend}>
        <input
          type="text"
          className="chat-input"
          placeholder="Ask about spacecraft systems, procedures, or report an issue..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={loading}
          autoFocus
        />
        <button
          type="submit"
          className="chat-send-btn"
          disabled={!input.trim() || loading}
        >
          Send
        </button>
      </form>
    </div>
  )
}
