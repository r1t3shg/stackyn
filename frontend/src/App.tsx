import { Routes, Route, Navigate } from 'react-router-dom';
import LandingPage from './pages/LandingPage';
import TermsOfService from './pages/TermsOfService';
import PrivacyPolicy from './pages/PrivacyPolicy';
import Pricing from './pages/Pricing';
import Home from './pages/Home';
import NewApp from './pages/NewApp';
import AppDetails from './pages/AppDetails';
import DeploymentDetails from './pages/DeploymentDetails';
import Login from './pages/Login';
import ProtectedRoute from './components/ProtectedRoute';

function App() {
  return (
    <Routes>
      <Route path="/" element={<LandingPage />} />
      <Route path="/terms" element={<TermsOfService />} />
      <Route path="/privacy" element={<PrivacyPolicy />} />
      <Route path="/pricing" element={<Pricing />} />
      <Route path="/login" element={<Login />} />
      <Route
        path="/apps"
        element={
          <ProtectedRoute>
            <Home />
          </ProtectedRoute>
        }
      />
      <Route
        path="/apps/new"
        element={
          <ProtectedRoute>
            <NewApp />
          </ProtectedRoute>
        }
      />
      <Route
        path="/apps/:id"
        element={
          <ProtectedRoute>
            <AppDetails />
          </ProtectedRoute>
        }
      />
      <Route
        path="/apps/:id/deployments/:deploymentId"
        element={
          <ProtectedRoute>
            <DeploymentDetails />
          </ProtectedRoute>
        }
      />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;


