import { useState, useEffect, useCallback } from 'react';
import { IS_LOCAL } from '@/utils';

export interface CSRFTokenResponse {
  csrf_token: string;
}

export interface UseCSRF {
  token: string | null;
  isLoading: boolean;
  error: string | null;
}

/**
 * Get CSRF token from cookie
 */
export const getCSRFTokenFromCookie = (): {
  token: string | null;
  error: string | null;
} => {
  if (typeof document === 'undefined')
    return {
      token: null,
      error: 'document is undefined',
    };

  const cookieValue = document.cookie
    .split('; ')
    .find((row) => row.startsWith('csrf_token='))
    ?.split('=')[1];

  return {
    token: cookieValue || null,
    error: null,
  };
};

/**
 * Get CSRF token from server
 */
export const getCSRFTokenFromServer = async (): Promise<{
  token: string | null;
  error: string | null;
}> => {
  try {
    const response = await fetch('/auth/csrf-token', {
      method: 'GET',
      credentials: 'include', // Include cookies for session validation
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    const data: CSRFTokenResponse = await response.json();

    return {
      token: data.csrf_token,
      error: null,
    };
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : 'Failed to fetch CSRF token from server';
    return {
      token: null,
      error: errorMessage,
    };
  }
};

/**
 * Create headers object with CSRF token
 */
export const createCSRFHeaders = (token: string | null): Record<string, string> => {
  const headers: Record<string, string> = {};

  if (token) {
    headers['X-CSRF-Token'] = token;
  }

  return headers;
};

/**
 * Hook to manage CSRF tokens for secure requests
 */
export const useCSRF = (): UseCSRF => {
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCSRFToken = useCallback(async () => {
    setError(null);

    // const cookieToken = getCSRFTokenFromCookie();
    // if (cookieToken.token) {
    //   setToken(cookieToken.token);
    //   return;
    // }

    setIsLoading(true);

    const serverToken = await getCSRFTokenFromServer();
    setToken(serverToken.token);
    setError(serverToken.error);

    setIsLoading(false);
  }, []);

  useEffect(() => {
    if (IS_LOCAL) {
      return;
    }

    // Fetch token on mount
    if (!token) {
      fetchCSRFToken();
      return;
    }

    // Refresh token every 23 hours (before 24h expiry)
    const refreshInterval = setInterval(() => fetchCSRFToken(), 23 * 60 * 60 * 1000);
    return () => clearInterval(refreshInterval);
  }, [token, fetchCSRFToken]);

  return {
    token,
    isLoading,
    error,
  };
};
