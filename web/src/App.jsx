import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import LandingPage from './pages/LandingPage';
import LoginPage from './pages/LoginPage';
import AuthCallback from './pages/AuthCallback';
import Dashboard from './pages/Dashboard';
import UserDetails from './pages/UserDetails';
import AuthWrapper from './components/AuthWrapper';
import APIExplorer from './pages/APIExplorer';
import './App.css';

function App() {
  return (
    <Router>
      <Routes>
        <Route element={<AuthWrapper />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/user/:userId" element={<UserDetails />} />
          <Route path="/api-explorer" element={<APIExplorer />} />
        </Route>
        <Route path="/login" element={<LoginPage mode="login" />} />
        <Route path="/signup" element={<LoginPage mode="signup" />} />
        <Route path="/auth/x/callback" element={<AuthCallback />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
}

export default App;
