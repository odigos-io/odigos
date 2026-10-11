import { useState, useEffect } from 'react';
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
 *
 * In production we initialize `isLoading: true` so the very first
 * render of the consumer (typically `<OdigosApiAdapter>`'s ready
 * gate) reports the loading state. Without this, the synchronous
 * first render sees `isLoading: false` (the React `useState` initial)
 * before the mount-effect flips it to `true`, and any cache-first
 * `useApiQuery` inside the provider fires its initial fetch before
 * the CSRF header has a chance to populate. In `IS_LOCAL` mode the
 * hook is inert (no fetch, no header), so we initialize `false` and
 * the consumer gates skip the loader.
 */
export const useCSRF = (): UseCSRF => {
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(!IS_LOCAL);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (IS_LOCAL) {
      return;
    }

    let cancelled = false;

    // All setState calls happen after `await`, never synchronously inside the
    // effect body, so this doesn't trigger the cascading-render lint rule.
    const fetchAndApply = async () => {
      const serverToken = await getCSRFTokenFromServer();
      if (cancelled) return;
      setToken(serverToken.token);
      setError(serverToken.error);
      setIsLoading(false);
    };

    fetchAndApply();

    // Refresh token every 23 hours (before 24h expiry)
    const refreshInterval = setInterval(fetchAndApply, 23 * 60 * 60 * 1000);

    return () => {
      cancelled = true;
      clearInterval(refreshInterval);
    };
  }, []);

  return {
    token,
    isLoading,
    error,
  };
};
