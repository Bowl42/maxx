import { createContext, useContext, useState, useEffect, type ReactNode } from 'react';
import { useTransport } from '@/lib/transport';

const AUTH_TOKEN_KEY = 'maxx-admin-token';
const AUTH_INIT_TIMEOUT_MS = 8000;

interface AuthContextValue {
  isAuthenticated: boolean;
  isLoading: boolean;
  authEnabled: boolean;
  login: (token: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const { transport } = useTransport();
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [authEnabled, setAuthEnabled] = useState(false);

  useEffect(() => {
    let cancelled = false;
    let timedOut = false;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const shouldSkip = () => cancelled || timedOut;

    const checkAuth = async () => {
      try {
        const status = await transport.getAuthStatus();
        if (shouldSkip()) {
          return;
        }
        setAuthEnabled(status.authEnabled);

        if (!status.authEnabled) {
          setIsAuthenticated(true);
          return;
        }

        const savedToken = localStorage.getItem(AUTH_TOKEN_KEY);
        if (savedToken) {
          transport.setAuthToken(savedToken);
          try {
            await transport.getProxyStatus();
            if (shouldSkip()) {
              return;
            }
            setIsAuthenticated(true);
          } catch (error) {
            if (shouldSkip()) {
              return;
            }
            console.error('[AuthProvider] Saved token verification failed:', error);
            localStorage.removeItem(AUTH_TOKEN_KEY);
            transport.clearAuthToken();
          }
        }
      } catch (error) {
        if (shouldSkip()) {
          return;
        }
        console.error('[AuthProvider] Auth check failed, fallback to authenticated:', error);
        // Auth check failed, assume no auth required
        setIsAuthenticated(true);
      }
    };

    const runAuthBootstrap = async () => {
      console.log('[AuthProvider] Starting auth bootstrap...');

      try {
        await Promise.race([
          checkAuth(),
          new Promise<never>((_, reject) => {
            timeoutId = setTimeout(() => {
              reject(
                new Error(`[AuthProvider] Auth bootstrap timeout after ${AUTH_INIT_TIMEOUT_MS}ms`),
              );
            }, AUTH_INIT_TIMEOUT_MS);
          }),
        ]);
      } catch (error) {
        if (cancelled) {
          return;
        }

        timedOut = true;
        console.error(
          '[AuthProvider] Auth bootstrap failed or timed out, fallback to authenticated:',
          error,
        );
        setIsAuthenticated(true);
      } finally {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    runAuthBootstrap();

    return () => {
      cancelled = true;
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [transport]);

  const login = (token: string) => {
    localStorage.setItem(AUTH_TOKEN_KEY, token);
    transport.setAuthToken(token);
    setIsAuthenticated(true);
  };

  const logout = () => {
    localStorage.removeItem(AUTH_TOKEN_KEY);
    transport.clearAuthToken();
    setIsAuthenticated(false);
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, isLoading, authEnabled, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
}
