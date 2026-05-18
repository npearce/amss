import { useState, useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { UserContext } from './UserContext'
import NavBar from './components/NavBar'
import LoginPage from './LoginPage'
import { fetchCrew } from './api'
import { isAuthenticated, logout, keycloakEnabled } from './auth'

export default function App() {
  const [authed, setAuthed] = useState(() => isAuthenticated())
  const [crew, setCrew] = useState([])
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    if (!authed) return
    fetchCrew()
      .then((data) => {
        const members = data.crew ?? []
        setCrew(members)
        const defaultUser =
          members.find((m) => m.persona === 'astronaut' && m.status === 'active') ??
          members[0] ??
          null
        setCurrentUser(defaultUser)
      })
      .catch(() => {})
  }, [authed])

  if (!authed) {
    return <LoginPage onLogin={() => setAuthed(true)} />
  }

  function handleLogout() {
    logout()
    setAuthed(false)
    setCrew([])
    setCurrentUser(null)
  }

  return (
    <UserContext.Provider
      value={{
        currentUser,
        setCurrentUser,
        crew,
        onLogout: keycloakEnabled ? handleLogout : null,
      }}
    >
      <div className="app">
        <NavBar />
        <main className="main">
          <Outlet />
        </main>
      </div>
    </UserContext.Provider>
  )
}
