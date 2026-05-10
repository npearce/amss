import { useState, useEffect } from 'react'
import { Outlet } from 'react-router-dom'
import { UserContext } from './UserContext'
import NavBar from './components/NavBar'
import { fetchCrew } from './api'

export default function App() {
  const [crew, setCrew] = useState([])
  const [currentUser, setCurrentUser] = useState(null)

  useEffect(() => {
    fetchCrew()
      .then((data) => {
        const members = data.crew ?? []
        setCrew(members)
        // Default to the first active astronaut
        const defaultUser =
          members.find((m) => m.persona === 'astronaut' && m.status === 'active') ??
          members[0] ??
          null
        setCurrentUser(defaultUser)
      })
      .catch(() => {
        // BFF not reachable — render empty UI so layout still shows
      })
  }, [])

  return (
    <UserContext.Provider value={{ currentUser, setCurrentUser, crew }}>
      <div className="app">
        <NavBar />
        <main className="main">
          <Outlet />
        </main>
      </div>
    </UserContext.Provider>
  )
}
