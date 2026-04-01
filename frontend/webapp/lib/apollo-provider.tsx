'use client';

import { type FC, type PropsWithChildren } from 'react';
import { API, IS_LOCAL } from '@/utils';
import { useCSRF } from '@/hooks';
import { onError } from '@apollo/client/link/error';
import { ApolloLink, HttpLink } from '@apollo/client';
import { setContext } from '@apollo/client/link/context';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';
import { ApolloNextAppProvider, InMemoryCache, ApolloClient, SSRMultipartLink } from '@apollo/client-integration-nextjs';
import { getCSRFTokenFromCookie } from '@/hooks/tokens/useCSRF';

const makeClient = (csrfToken: string | null) => {
  const httpLink = new HttpLink({
    uri: API.GRAPHQL,
    credentials: IS_LOCAL ? 'same-origin' : 'include', // Include cookies for CSRF token
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) graphQLErrors.forEach(({ message, locations, path }) => console.warn(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`));
    if (networkError) console.warn(`[Network error]: ${networkError}`);
  });

  // Prefer token from document.cookie so X-CSRF-Token always matches the cookie the browser sends
  // (avoids drift from React state; cookie values can include '=' padding — never parse with split('=')[1]).
  const authLink = setContext((_, ctx) => {
    const headers = {
      ...ctx.headers,
    };
    const fromCookie = typeof document !== 'undefined' ? getCSRFTokenFromCookie().token : null;
    const t = fromCookie ?? csrfToken;
    if (t) {
      headers['X-CSRF-Token'] = t;
    }

    return { headers };
  });

  // TODO: block mutations for readonly operations and remove from all hooks
  // const readonlyLink = setContext((operation) => {
  //   const operationType = (operation.query.definitions[0] as OperationDefinitionNode)?.operation;
  //   return {};
  // });

  return new ApolloClient({
    devtools: {
      enabled: true,
    },
    defaultOptions: {
      watchQuery: { fetchPolicy: 'cache-and-network' },
      query: { fetchPolicy: 'cache-first' },
      mutate: { fetchPolicy: 'network-only' },
    },
    cache: new InMemoryCache({
      addTypename: false,
    }),
    link:
      typeof window === 'undefined'
        ? ApolloLink.from([
            new SSRMultipartLink({
              stripDefer: true,
            }),
            errorLink,
            httpLink,
          ])
        : ApolloLink.from([authLink, errorLink, httpLink]),
  });
};

const ApolloProvider: FC<PropsWithChildren> = ({ children }) => {
  const { token, isLoading, error } = useCSRF();

  if (isLoading) {
    return (
      <CenterThis style={{ height: '100%' }}>
        <FadeLoader scale={2} />
      </CenterThis>
    );
  }

  if (!token && error) {
    return (
      <CenterThis style={{ height: '100%', padding: 24, textAlign: 'center' }}>
        Could not load security token (CSRF). Try refreshing the page or clearing cookies for this host.
        {IS_LOCAL
          ? ' Dev: run kubectl port-forward svc/ui -n odigos-system 3000:3000 when using Next on :3001.'
          : ''}
      </CenterThis>
    );
  }

  return (
    <ApolloNextAppProvider key={token ?? ''} makeClient={() => makeClient(token)}>
      {children}
    </ApolloNextAppProvider>
  );
};

export default ApolloProvider;
