import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { 
  createUserWithEmailAndPassword, 
  signInWithEmailAndPassword,
  signOut,
  sendEmailVerification,
  sendPasswordResetEmail,
  onAuthStateChanged,
  User as FirebaseUser
} from 'firebase/auth';
import { API_BASE_URL } from '@/lib/config';
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
    // Try Firebase login first (for new users)
    try {
      const userCredential = await signInWithEmailAndPassword(auth, email, password);
      const idToken = await userCredential.user.getIdToken();
      
      setToken(idToken);
      setFirebaseUser(userCredential.user);
      
      // Also try to get user from our backend
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
        }
      } catch (err) {
        console.error('Failed to verify token with backend:', err);
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
          const error = await response.json().catch(() => ({ error: 'Login failed' }));
          throw new Error(error.error || 'Invalid email or password');
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

  // Send password reset email
  const sendPasswordReset = async (email: string): Promise<void> => {
    try {
      // Firebase will use the default action URL configured in Firebase Console
      await sendPasswordResetEmail(auth, email);
    } catch (error: any) {
      // Don't reveal if email exists or not for security
      // Firebase will send email if account exists, otherwise silently fail
      if (error.code === 'auth/user-not-found') {
        // Silently succeed to prevent email enumeration
        return;
      }
      throw new Error(error.message || 'Failed to send password reset email');
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
