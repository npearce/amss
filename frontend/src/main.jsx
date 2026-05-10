import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import App from './App'
import AstronautChat from './views/AstronautChat'
import GroundControl from './views/GroundControl'
import './styles/global.css'

createRoot(document.getElementById('root')).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<App />}>
          <Route index element={<Navigate to="/chat" replace />} />
          <Route path="chat" element={<AstronautChat />} />
          <Route path="ground-control" element={<GroundControl />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
