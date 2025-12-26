import { Link } from 'react-router-dom';

interface LogoProps {
  className?: string;
  height?: number;
}

export default function Logo({ className = '', height = 40 }: LogoProps) {
  return (
    <Link to="/" className={`flex items-center ${className}`}>
      <img
        src="/stackyn_logo.svg"
        alt="Stackyn"
        height={height}
        className="h-auto"
      />
    </Link>
  );
}

