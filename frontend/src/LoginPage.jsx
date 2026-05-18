import { useState } from 'react'
import { login } from './auth'

export default function LoginPage({ onLogin }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await login(username, password)
      onLogin()
    } catch {
      setError('Invalid credentials. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-logo">◈</div>
        <h1 className="login-title">AMSS Login</h1>
        <p className="login-subtitle">Artemis Mission Support System</p>

        <form className="login-form" onSubmit={handleSubmit}>
          <div className="login-field">
            <label className="login-label" htmlFor="username">Username</label>
            <input
              id="username"
              className="login-input"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
              disabled={loading}
              required
            />
          </div>
          <div className="login-field">
            <label className="login-label" htmlFor="password">Password</label>
            <input
              id="password"
              className="login-input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
              disabled={loading}
              required
            />
          </div>

          {error && <div className="login-error">{error}</div>}

          <button className="login-btn" type="submit" disabled={loading}>
            {loading ? 'Authenticating...' : 'Log In'}
          </button>
        </form>

        <div className="login-hints">
          <div className="login-hint-label">Demo credentials</div>
          <div className="login-hint">
            Crew: <span className="login-hint-val">wiseman / artemis</span>
          </div>
          <div className="login-hint">
            Ground Control: <span className="login-hint-val">gc-eclss / houston</span>
          </div>
        </div>
      </div>
    </div>
  )
}
