import { Link } from 'react-router-dom';
import Layout from '../components/Layout';

export default function Dashboard() {
  return (
    <Layout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Admin Dashboard</h1>
          <p className="mt-2 text-sm text-gray-600">
            Manage users, apps, and system resources
          </p>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          <Link
            to="/users"
            className="block p-6 bg-white rounded-lg shadow hover:shadow-lg transition-shadow"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Users</h3>
            <p className="text-sm text-gray-600">
              View and manage user accounts, plans, and quotas
            </p>
          </Link>

          <Link
            to="/apps"
            className="block p-6 bg-white rounded-lg shadow hover:shadow-lg transition-shadow"
          >
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Apps</h3>
            <p className="text-sm text-gray-600">
              Monitor and manage deployed applications
            </p>
          </Link>

          <div className="block p-6 bg-white rounded-lg shadow">
            <h3 className="text-lg font-semibold text-gray-900 mb-2">Plans & Quotas</h3>
            <p className="text-sm text-gray-600">
              View plan limits and usage statistics
            </p>
          </div>
        </div>
      </div>
    </Layout>
  );
}

