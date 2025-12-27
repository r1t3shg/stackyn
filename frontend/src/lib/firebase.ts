// Firebase configuration and initialization
import { initializeApp, getApps, FirebaseApp } from 'firebase/app';
import { getAuth, Auth } from 'firebase/auth';
import { getAnalytics, Analytics } from 'firebase/analytics';

// Your web app's Firebase configuration
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY || "AIzaSyDvur-G712H9H-wtLxjNxgIoUIjdMMRUBg",
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN || "stackyn-4acc0.firebaseapp.com",
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID || "stackyn-4acc0",
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET || "stackyn-4acc0.firebasestorage.app",
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID || "356626460530",
  appId: import.meta.env.VITE_FIREBASE_APP_ID || "1:356626460530:web:ce470e607025d2ad54f7eb",
  measurementId: import.meta.env.VITE_FIREBASE_MEASUREMENT_ID || "G-XSPQXJ6EPB"
};

// Initialize Firebase
let app: FirebaseApp;
if (getApps().length === 0) {
  app = initializeApp(firebaseConfig);
} else {
  app = getApps()[0];
}

// Initialize Firebase Auth
export const auth: Auth = getAuth(app);

// Initialize Analytics (only in browser environment)
let analytics: Analytics | null = null;
if (typeof window !== 'undefined') {
  try {
    analytics = getAnalytics(app);
  } catch (error) {
    console.warn('Firebase Analytics initialization failed:', error);
  }
}

export { analytics };
export default app;

