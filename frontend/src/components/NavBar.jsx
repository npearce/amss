import { useContext } from 'react'
import { NavLink } from 'react-router-dom'
import { UserContext } from '../UserContext'
import UserSwitcher from './UserSwitcher'

export default function NavBar() {
  const { crew, currentUser, setCurrentUser, onLogout } = useContext(UserContext)

  return (
    <nav className="navbar">
      <div className="navbar-brand">
        <span className="navbar-logo">◈</span>
        <span className="navbar-title">AMSS</span>
        <span className="navbar-subtitle">Artemis Mission Support</span>
      </div>

      <div className="navbar-links">
        <NavLink
          to="/chat"
          className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
        >
          Astronaut Chat
        </NavLink>
        <NavLink
          to="/ground-control"
          className={({ isActive }) => 'nav-link' + (isActive ? ' active' : '')}
        >
          Ground Control
        </NavLink>
      </div>

      <UserSwitcher crew={crew} currentUser={currentUser} onChange={setCurrentUser} />
      {onLogout && (
        <button className="logout-btn" onClick={onLogout}>Log out</button>
      )}
    </nav>
  )
}
