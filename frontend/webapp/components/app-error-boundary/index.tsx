'use client';

import React from 'react';
import { Theme } from '@odigos/ui-theme';
import { Button } from '@odigos/ui-components';
import { ErrorBoundary } from 'react-error-boundary';

const ErrorFallback = ({ error }: { error: Error }) => {
  const theme = Theme.useTheme();

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        textAlign: 'center',
        color: theme.text.error,
        fontFamily: theme.font_family.primary,
      }}
    >
      <h2>Something went wrong!</h2>

      <pre
        style={{
          padding: 10,
          borderRadius: 5,
          whiteSpace: 'pre-wrap',
          background: theme.colors.error,
        }}
      >
        {error.message}
      </pre>

      <details style={{ textAlign: 'left', marginTop: 10 }}>
        <summary>Stack Trace</summary>
        <pre style={{ whiteSpace: 'pre-wrap', fontSize: '12px' }}>{error.stack}</pre>
      </details>

      <p style={{ marginTop: 100, color: theme.text.secondary }}>Try refreshing the page or contact support.</p>
      <Button variant='secondary' onClick={() => window.location.reload()} style={{ fontSize: '1rem' }}>
        Reload
      </Button>
    </div>
  );
};

export const AppErrorBoundary = ({ children }: { children: React.ReactNode }) => {
  return (
    <ErrorBoundary
      FallbackComponent={(props) => (
        <Theme.Provider darkMode={false}>
          <ErrorFallback {...props} />
        </Theme.Provider>
      )}
    >
      {children}
    </ErrorBoundary>
  );
};
