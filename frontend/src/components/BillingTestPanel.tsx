import { useState } from 'react';
import { userApi } from '@/lib/api';

interface BillingTestPanelProps {
  onUpdate?: () => void;
}

/**
 * Test panel for simulating billing states
 * Only visible in development/test environments
 */
export default function BillingTestPanel({ onUpdate }: BillingTestPanelProps) {
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [billingStatus, setBillingStatus] = useState<'trial' | 'active' | 'expired'>('trial');
  const [plan, setPlan] = useState<'free_trial' | 'starter' | 'pro'>('free_trial');

  // Only show in development
  if (process.env.NODE_ENV === 'production') {
    return null;
  }

  const handleTestBilling = async () => {
    setLoading(true);
    setMessage(null);

    try {
      const token = localStorage.getItem('auth_token');
      if (!token) {
        setMessage('Error: Not authenticated');
        return;
      }

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'}/api/v1/test/billing`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          billing_status: billingStatus,
          plan: plan,
          // Optionally set trial_ends_at to past date to test expired trial
          ...(billingStatus === 'expired' && {
            trial_ends_at: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(), // Yesterday
          }),
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || 'Failed to update billing state');
      }

      setMessage(`✅ Billing state updated: ${billingStatus} / ${plan}`);
      
      // Reload user profile
      if (onUpdate) {
        setTimeout(() => {
          onUpdate();
        }, 500);
      }
    } catch (err) {
      setMessage(`❌ Error: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed bottom-4 right-4 bg-yellow-50 border-2 border-yellow-400 rounded-lg p-4 shadow-lg max-w-sm z-50">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-lg">🧪</span>
        <h3 className="font-bold text-gray-900">Billing Test Panel</h3>
      </div>
      
      <div className="space-y-3">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Billing Status
          </label>
          <select
            value={billingStatus}
            onChange={(e) => setBillingStatus(e.target.value as typeof billingStatus)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
          >
            <option value="trial">Trial</option>
            <option value="active">Active</option>
            <option value="expired">Expired</option>
          </select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Plan
          </label>
          <select
            value={plan}
            onChange={(e) => setPlan(e.target.value as typeof plan)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
          >
            <option value="free_trial">Free Trial</option>
            <option value="starter">Starter</option>
            <option value="pro">Pro</option>
          </select>
        </div>

        <button
          onClick={handleTestBilling}
          disabled={loading}
          className="w-full bg-yellow-500 hover:bg-yellow-600 text-white font-semibold py-2 px-4 rounded-md text-sm transition-colors disabled:opacity-50"
        >
          {loading ? 'Updating...' : 'Update Billing State'}
        </button>

        {message && (
          <div className={`text-sm p-2 rounded ${message.includes('✅') ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
            {message}
          </div>
        )}

        <p className="text-xs text-gray-600">
          ⚠️ This updates your actual billing status in the database. Use with caution!
        </p>
      </div>
    </div>
  );
}

