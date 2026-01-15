import { Routes, Route, Navigate } from 'react-router-dom';
import LandingPage from './pages/LandingPage';
import TermsOfService from './pages/TermsOfService';
import PrivacyPolicy from './pages/PrivacyPolicy';
import Home from './pages/Home';
import NewApp from './pages/NewApp';
import AppDetails from './pages/AppDetails';
import DeploymentDetails from './pages/DeploymentDetails';
import Login from './pages/Login';
import SignUp from './pages/SignUp';
import ForgotPassword from './pages/ForgotPassword';
import ProtectedRoute from './components/ProtectedRoute';
import PricingRedirect from './components/PricingRedirect';

function App() {
  // Check if we're on the console subdomain
  const isConsoleSubdomain = typeof window !== 'undefined' && 
    window.location.hostname === 'console.dev.stackyn.com';

  return (
    <Routes>
      {/* On console subdomain, root shows apps list; otherwise show landing page */}
      <Route 
        path="/" 
        element={
          isConsoleSubdomain ? (
            <ProtectedRoute>
              <Home />
            </ProtectedRoute>
          ) : (
            <LandingPage />
          )
        } 
      />
      <Route path="/terms" element={<TermsOfService />} />
      <Route path="/privacy" element={<PrivacyPolicy />} />
      <Route path="/pricing" element={<PricingRedirect />} />
      <Route path="/login" element={<Login />} />
      <Route path="/signup" element={<SignUp />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
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


