import { useEffect } from 'react';

interface UpgradePlanModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export default function UpgradePlanModal({
  isOpen,
  onClose,
}: UpgradePlanModalProps) {
  // Close on Escape key
  useEffect(() => {
    if (!isOpen) return;
    
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };
    
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      />
      
      {/* Modal */}
      <div className="relative bg-[var(--app-bg)] rounded-lg shadow-xl border border-[var(--border-subtle)] max-w-md w-full mx-4 z-10">
        <div className="p-6">
          {/* Title */}
          <h3 className="text-xl font-semibold text-[var(--text-primary)] mb-3">
            Upgrade Plan
          </h3>
          
          {/* Message */}
          <p className="text-[var(--text-secondary)] mb-6">
            We're working hard to bring you a seamless billing experience. Billing features will be available soon!
          </p>
          
          {/* Actions */}
          <div className="flex gap-3 justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-[var(--border-subtle)] hover:bg-[var(--border)] text-[var(--text-primary)] font-medium rounded-lg transition-colors"
            >
              Close
            </button>
            <button
              disabled
              className="px-4 py-2 bg-[var(--border-subtle)] text-[var(--text-secondary)] font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              Billing launching soon
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

