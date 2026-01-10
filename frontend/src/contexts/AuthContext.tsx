import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { 
  createUserWithEmailAndPassword, 
  signInWithEmailAndPassword,
  signOut,
  sendEmailVerification,
  onAuthStateChanged,
  User as FirebaseUser
} from 'firebase/auth';
import { API_BASE_URL } from '@/lib/config';
import { auth } from '@/lib/firebase';
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
  firebaseUser: FirebaseUser | null;
  token: string | null;
  login: (email: string, password: string) => Promise<void>;
  signup: (email: string, password: string) => Promise<void>;
  signupFirebase: (email: string, password: string) => Promise<FirebaseUser>;
  signupComplete: (idToken: string, fullName: string, companyName: string, plan?: string) => Promise<void>;
  resendEmailVerification: () => Promise<void>;
  sendPasswordReset: (email: string) => Promise<void>;
  logout: () => Promise<void>;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [firebaseUser, setFirebaseUser] = useState<FirebaseUser | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Initialize auth state and listen to Firebase auth changes
  useEffect(() => {
    // First, check localStorage immediately for legacy JWT tokens (for fast initial render)
    const storedToken = localStorage.getItem('auth_token');
    const storedUser = localStorage.getItem('auth_user');
    
    if (storedToken && storedUser) {
      try {
        const parsedUser = JSON.parse(storedUser);
        setToken(storedToken);
        setUser(parsedUser);
        // Set isLoading to false immediately if we have a valid token in localStorage
        // This prevents redirect to login page while waiting for Firebase
        setIsLoading(false);
      } catch (e) {
        console.error('Failed to parse stored user:', e);
        // Clear invalid data
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
      }
    }

    // Then set up Firebase auth state listener
    const unsubscribe = onAuthStateChanged(auth, async (firebaseUser) => {
      setFirebaseUser(firebaseUser);
      
      if (firebaseUser) {
        // Firebase user is authenticated - prioritize Firebase auth
        try {
          // Get ID token
          const idToken = await firebaseUser.getIdToken();
          setToken(idToken);
          
          // Load user from our backend if available
          const storedUser = localStorage.getItem('auth_user');
          if (storedUser) {
            try {
              setUser(JSON.parse(storedUser));
            } catch (e) {
              console.error('Failed to parse stored user:', e);
            }
          }
          
          // Store Firebase token
          localStorage.setItem('auth_token', idToken);
          setIsLoading(false);
        } catch (error) {
          console.error('Failed to get Firebase ID token:', error);
          // If Firebase token fetch fails, check if we have a legacy token
          const storedToken = localStorage.getItem('auth_token');
          const storedUser = localStorage.getItem('auth_user');
          if (storedToken && storedUser) {
            try {
              setToken(storedToken);
              setUser(JSON.parse(storedUser));
              setIsLoading(false);
            } catch (e) {
              console.error('Failed to restore legacy auth:', e);
              setIsLoading(false);
            }
          } else {
            setIsLoading(false);
          }
        }
      } else {
        // Firebase user is null - check if we have a legacy JWT token
        const storedToken = localStorage.getItem('auth_token');
        const storedUser = localStorage.getItem('auth_user');
        
        if (storedToken && storedUser) {
          // We have a legacy JWT token - keep the user authenticated
          try {
            const parsedUser = JSON.parse(storedUser);
            setToken(storedToken);
            setUser(parsedUser);
            // Keep firebaseUser as null (legacy user)
            setIsLoading(false);
          } catch (e) {
            console.error('Failed to parse stored user:', e);
            // Clear invalid data
            setUser(null);
            setToken(null);
            localStorage.removeItem('auth_token');
            localStorage.removeItem('auth_user');
            setIsLoading(false);
          }
        } else {
          // No Firebase user and no legacy token - user is logged out
          setUser(null);
          setToken(null);
          setIsLoading(false);
        }
      }
    });

    // Listen for unauthorized events from API calls
    const handleUnauthorized = () => {
      setUser(null);
      setToken(null);
      setFirebaseUser(null);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    };

    window.addEventListener('auth:unauthorized', handleUnauthorized);

    return () => {
      unsubscribe();
      window.removeEventListener('auth:unauthorized', handleUnauthorized);
    };
  }, []);

  const login = async (email: string, password: string) => {
    // Try Firebase login first (for new users)
    try {
      const userCredential = await signInWithEmailAndPassword(auth, email, password);
      const idToken = await userCredential.user.getIdToken();
      
      setToken(idToken);
      setFirebaseUser(userCredential.user);
      
      // Also try to get user from our backend (non-blocking - don't fail login if this fails)
      try {
        const response = await fetch(`${API_BASE_URL}/api/auth/verify-token`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ id_token: idToken }),
        });
        
        if (response.ok) {
          const data = await response.json();
          const userData: User = {
            id: userCredential.user.uid,
            email: userCredential.user.email || '',
            email_verified: data.email_verified,
          };
          setUser(userData);
          localStorage.setItem('auth_user', JSON.stringify(userData));
        } else {
          // If verify-token fails, still allow login but log the error
          console.warn('Failed to verify token with backend (non-critical):', response.status);
        }
      } catch (err) {
        // Don't throw - this is a non-critical operation
        // Login should still succeed even if backend verification fails
        if (err instanceof Error) {
          console.warn(`Failed to verify token with backend at ${API_BASE_URL}/api/auth/verify-token (non-critical):`, err.message);
          if (err.message.includes('Failed to fetch') || err.message.includes('ERR_CONNECTION_REFUSED')) {
            console.warn('Make sure the backend server is running on port 8080');
          }
        } else {
          console.warn('Failed to verify token with backend (non-critical):', err);
        }
      }
      
      localStorage.setItem('auth_token', idToken);
      return; // Success with Firebase
    } catch (firebaseError: any) {
      // If Firebase login fails, try legacy backend login
      // This handles users created before Firebase integration
      console.log('Firebase login failed, trying legacy login:', firebaseError.code);
      
      // Only fallback to legacy login for specific Firebase errors
      const fallbackErrors = ['auth/user-not-found', 'auth/wrong-password', 'auth/invalid-credential'];
      if (!fallbackErrors.includes(firebaseError.code)) {
        // For other Firebase errors, throw the original error
        throw new Error(firebaseError.message || 'Login failed');
      }
      
      // Try legacy backend login
      try {
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
        
        // Store legacy JWT token
        setToken(data.token);
        setUser(data.user);
        localStorage.setItem('auth_token', data.token);
        localStorage.setItem('auth_user', JSON.stringify(data.user));
        
        // Note: firebaseUser will remain null for legacy users
        // This is fine - they can still use the app with JWT tokens
        console.log('Legacy login successful');
      } catch (legacyError: any) {
        // If legacy login also fails, throw a clear error
        // Make sure to preserve the error message
        if (legacyError instanceof Error) {
          throw legacyError;
        }
        throw new Error(legacyError.message || 'Invalid email or password');
      }
    }
  };

  const signup = async (email: string, password: string): Promise<void> => {
    // Legacy signup - redirect to Firebase signup
    await signupFirebase(email, password);
  };

  const signupFirebase = async (email: string, password: string): Promise<FirebaseUser> => {
    try {
      // Create Firebase user
      const userCredential = await createUserWithEmailAndPassword(auth, email, password);
      
      // Send email verification
      // Note: Firebase will use the default action URL configured in Firebase Console
      // Make sure your domain is authorized in Firebase Console > Authentication > Settings > Authorized domains
      try {
        await sendEmailVerification(userCredential.user);
        console.log('Email verification sent successfully');
      } catch (verifyError: any) {
        console.error('Failed to send email verification:', verifyError);
        // Don't fail the signup if email sending fails - user can resend later
        // The error might be due to Firebase configuration issues
      }
      
      return userCredential.user;
    } catch (error: any) {
      throw new Error(error.message || 'Signup failed');
    }
  };

  // Resend email verification
  const resendEmailVerification = async (): Promise<void> => {
    if (!firebaseUser) {
      throw new Error('No user logged in');
    }
    
    try {
      // Firebase will use the default action URL configured in Firebase Console
      await sendEmailVerification(firebaseUser);
    } catch (error: any) {
      throw new Error(error.message || 'Failed to resend verification email');
    }
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

  const signupComplete = async (idToken: string, fullName: string, companyName: string, plan: string = 'free') => {
    if (!firebaseUser) {
      throw new Error('No user logged in');
    }

    const response = await fetch(`${API_BASE_URL}/api/auth/signup/complete`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        id_token: idToken,
        full_name: fullName,
        company_name: companyName,
        email: firebaseUser.email || '', // Include email for verification
        plan: plan, // Include selected plan
      }),
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Signup completion failed' }));
      throw new Error(error.error || 'Signup completion failed');
    }

    const data = await response.json();
    setToken(data.token || idToken);
    setUser(data.user);
    localStorage.setItem('auth_token', data.token || idToken);
    localStorage.setItem('auth_user', JSON.stringify(data.user));
  };

  const logout = async () => {
    try {
      await signOut(auth);
      setToken(null);
      setUser(null);
      setFirebaseUser(null);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    } catch (error: any) {
      console.error('Logout error:', error);
      // Clear state anyway
      setToken(null);
      setUser(null);
      setFirebaseUser(null);
      localStorage.removeItem('auth_token');
      localStorage.removeItem('auth_user');
    }
  };

  return (
    <AuthContext.Provider value={{ 
      user, 
      firebaseUser, 
      token, 
      login, 
      signup, 
      signupFirebase, 
      signupComplete, 
      resendEmailVerification,
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
