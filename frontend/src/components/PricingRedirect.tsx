import { useEffect } from 'react';

export default function PricingRedirect() {
  useEffect(() => {
    // Redirect to home page with pricing hash
    window.location.href = '/#pricing';
  }, []);

  return null;
}

