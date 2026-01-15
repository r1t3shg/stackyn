import { Link } from 'react-router-dom';

interface LogoProps {
  className?: string;
  height?: number;
  showText?: boolean;
}

export default function Logo({ className = '', height = 24, showText = true }: LogoProps) {
  return (
    <Link to="/" className={`flex items-center gap-2 ${className}`}>
      <img
        src="/stackyn_logo.svg"
        alt="Stackyn"
        height={height}
        className="h-auto"
      />
      {showText && (
        <span className="text-xl font-bold text-[var(--text-primary)]">Stackyn</span>
      )}
    </Link>
  );
}

