import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

export default function PricingRedirect() {
  const navigate = useNavigate();

  useEffect(() => {
    // Redirect to home page with pricing hash
    window.location.href = '/#pricing';
  }, []);

  return null;
}

