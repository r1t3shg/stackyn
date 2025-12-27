import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { 
  createUserWithEmailAndPassword, 
  signInWithEmailAndPassword,
  signOut,
  sendEmailVerification,
  onAuthStateChanged,
  User as FirebaseUser
} from 'firebase/auth';
import { auth } from '@/lib/firebase';

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
  signupComplete: (idToken: string, fullName: string, companyName: string) => Promise<void>;
  logout: () => Promise<void>;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [firebaseUser, setFirebaseUser] = useState<FirebaseUser | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Listen to Firebase auth state changes
  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, async (firebaseUser) => {
      setFirebaseUser(firebaseUser);
      
      if (firebaseUser) {
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
      } else {
        setUser(null);
        setToken(null);
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
      }
      
      setIsLoading(false);
    });

    return () => unsubscribe();
  }, []);

  const login = async (email: string, password: string) => {
    try {
      const userCredential = await signInWithEmailAndPassword(auth, email, password);
      const idToken = await userCredential.user.getIdToken();
      
      setToken(idToken);
      setFirebaseUser(userCredential.user);
      
      // Also try to get user from our backend
      const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
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
          // Try to get full user details from our backend
          // For now, just store basic info
          const userData: User = {
            id: userCredential.user.uid,
            email: userCredential.user.email || '',
            email_verified: data.email_verified,
          };
          setUser(userData);
          localStorage.setItem('auth_user', JSON.stringify(userData));
        }
      } catch (err) {
        console.error('Failed to verify token with backend:', err);
      }
      
      localStorage.setItem('auth_token', idToken);
    } catch (error: any) {
      throw new Error(error.message || 'Login failed');
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
      await sendEmailVerification(userCredential.user);
      
      return userCredential.user;
    } catch (error: any) {
      throw new Error(error.message || 'Signup failed');
    }
  };

  const signupComplete = async (idToken: string, fullName: string, companyName: string) => {
    const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
    const response = await fetch(`${API_BASE_URL}/api/auth/signup/complete`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ 
        id_token: idToken,
        full_name: fullName,
        company_name: companyName,
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
