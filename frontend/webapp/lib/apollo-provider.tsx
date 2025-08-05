'use client';

import { type FC, type PropsWithChildren } from 'react';
import { API } from '@/utils';
import { useCSRF } from '@/hooks';
import { onError } from '@apollo/client/link/error';
import { ApolloLink, HttpLink } from '@apollo/client';
import type { OperationDefinitionNode } from 'graphql';
import { setContext } from '@apollo/client/link/context';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';
import { ApolloNextAppProvider, InMemoryCache, ApolloClient, SSRMultipartLink } from '@apollo/client-integration-nextjs';

const makeClient = (csrfToken: string) => {
  const httpLink = new HttpLink({
    uri: API.GRAPHQL,
    credentials: 'include', // Include cookies for CSRF token
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) graphQLErrors.forEach(({ message, locations, path }) => console.warn(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`));
    if (networkError) console.warn(`[Network error]: ${networkError}`);
  });

  // Add CSRF token to headers for mutations
  const authLink = setContext((_, { headers }) => {
    // TODO: block mutations for readonly operations and remove from all hooks
    // const operationType = (operation.query.definitions[0] as OperationDefinitionNode)?.operation;

    return {
      headers: {
        ...headers,
        ...(csrfToken ? { 'X-CSRF-Token': csrfToken } : {}),
      },
    };
  });

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
  const { token } = useCSRF();

  if (!token) {
    return (
      <CenterThis style={{ height: '100%' }}>
        <FadeLoader scale={2} />
      </CenterThis>
    );
  }

  return <ApolloNextAppProvider makeClient={() => makeClient(token)}>{children}</ApolloNextAppProvider>;
};

export default ApolloProvider;
