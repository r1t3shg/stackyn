import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { API_BASE_URL } from '@/lib/config';
import { authApi } from '@/lib/api';

interface User {
  id: string;
  email: string;
  full_name?: string;
  company_name?: string;
  email_verified?: boolean;
}

interface AuthContextType {
  user: User | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  loginWithToken: (token: string, user: User) => void;
  logout: () => Promise<void>;
  sendPasswordReset: (email: string) => Promise<void>;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Initialize auth state from localStorage
  useEffect(() => {
    // First, check for token in URL parameter (for cross-subdomain authentication)
    const urlParams = new URLSearchParams(window.location.search);
    const tokenParam = urlParams.get('token');
    
    if (tokenParam) {
      // Store token from URL parameter and remove it from URL
      try {
        localStorage.setItem('auth_token', tokenParam);
        // Fetch user info with the token
        fetch(`${API_BASE_URL}/api/user/me`, {
          headers: {
            'Authorization': `Bearer ${tokenParam}`,
          },
        })
          .then(response => {
            if (response.ok) {
              return response.json();
            }
            throw new Error('Failed to fetch user info');
          })
          .then(userData => {
            localStorage.setItem('auth_user', JSON.stringify(userData));
            setToken(tokenParam);
            setUser(userData);
            // Clean up URL
            window.history.replaceState({}, '', window.location.pathname);
          })
          .catch(err => {
            console.error('Failed to fetch user info:', err);
            localStorage.removeItem('auth_token');
            // Clean up URL
            window.history.replaceState({}, '', window.location.pathname);
          })
          .finally(() => {
            setIsLoading(false);
          });
        return; // Early return to prevent checking localStorage
      } catch (e) {
        console.error('Failed to handle token parameter:', e);
        localStorage.removeItem('auth_token');
        // Clean up URL
        window.history.replaceState({}, '', window.location.pathname);
        setIsLoading(false);
      }
    }
    
    // Check localStorage for stored JWT tokens
    const storedToken = localStorage.getItem('auth_token');
    const storedUser = localStorage.getItem('auth_user');
    
    if (storedToken && storedUser) {
      try {
        const parsedUser = JSON.parse(storedUser);
        setToken(storedToken);
        setUser(parsedUser);
      } catch (e) {
        console.error('Failed to parse stored user:', e);
        // Clear invalid data
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
      }
    }
    
    setIsLoading(false);

    // Listen for unauthorized events from API calls
    const handleUnauthorized = () => {
      setUser(null);
      setToken(null);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    };

    window.addEventListener('auth:unauthorized', handleUnauthorized);

    return () => {
      window.removeEventListener('auth:unauthorized', handleUnauthorized);
    };
  }, []);

  const login = async (email: string, password: string) => {
    // Backend login with password
    const response = await fetch(`${API_BASE_URL}/api/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password }),
    });
    
    if (!response.ok) {
      let errorMessage = 'Invalid email or password';
      try {
        const error = await response.json();
        errorMessage = error.error || errorMessage;
      } catch {
        // If response is not JSON, use status text
        if (response.status === 401) {
          errorMessage = 'Invalid email or password';
        } else if (response.status === 500) {
          errorMessage = 'Server error. Please try again later.';
        } else if (response.status === 0 || response.status >= 500) {
          errorMessage = 'Cannot connect to server. Please check your connection.';
        } else {
          errorMessage = `Login failed (${response.status})`;
        }
      }
      throw new Error(errorMessage);
    }
    
    const data = await response.json();
    
    // Store JWT token
    loginWithToken(data.token, data.user);
  };

  const loginWithToken = (token: string, userData: User) => {
    setToken(token);
    setUser(userData);
    localStorage.setItem('auth_token', token);
    localStorage.setItem('auth_user', JSON.stringify(userData));
  };

  // Send password reset OTP via Resend
  const sendPasswordReset = async (email: string): Promise<void> => {
    try {
      await authApi.forgotPassword(email);
      // Always succeed to prevent email enumeration
      // The backend will return success even if email doesn't exist
    } catch (error: any) {
      // Provide user-friendly error message
      throw new Error(error.message || 'Failed to send password reset code. Please try again later.');
    }
  };

  const logout = async () => {
    // Clear state
    setToken(null);
    setUser(null);
    localStorage.removeItem('auth_token');
    localStorage.removeItem('auth_user');
  };

  return (
    <AuthContext.Provider value={{ 
      user, 
      token, 
      login,
      loginWithToken,
      sendPasswordReset,
      logout, 
      isLoading 
    }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
