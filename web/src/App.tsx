import { NavLink, Route, Routes } from 'react-router-dom';
import { TrackPage } from './pages/TrackPage';
import { OpsPage } from './pages/OpsPage';

export function App() {
  return (
    <div className="app">
      <header className="topbar">
        <span className="brand">📦 ParcelPigeon</span>
        <nav>
          <NavLink to="/" end>
            Track
          </NavLink>
          <NavLink to="/ops">Operations</NavLink>
        </nav>
      </header>
      <main>
        <Routes>
          <Route path="/" element={<TrackPage />} />
          <Route path="/ops" element={<OpsPage />} />
        </Routes>
      </main>
    </div>
  );
}
