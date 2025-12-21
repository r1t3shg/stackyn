import { Routes, Route } from 'react-router-dom';
import Home from './pages/Home';
import NewApp from './pages/NewApp';
import AppDetails from './pages/AppDetails';
import DeploymentDetails from './pages/DeploymentDetails';

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/apps/new" element={<NewApp />} />
      <Route path="/apps/:id" element={<AppDetails />} />
      <Route path="/apps/:id/deployments/:deploymentId" element={<DeploymentDetails />} />
    </Routes>
  );
}

export default App;

