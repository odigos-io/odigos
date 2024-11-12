'use client';

import { ApolloLink, HttpLink } from '@apollo/client';
<<<<<<< HEAD
import {
  ApolloNextAppProvider,
  InMemoryCache,
  ApolloClient,
  SSRMultipartLink,
} from '@apollo/experimental-nextjs-app-support';
import { onError } from '@apollo/client/link/error';

function makeClient() {
  const httpLink = new HttpLink({
    uri: 'http://localhost:8085/graphql',
=======
import { ApolloNextAppProvider, InMemoryCache, ApolloClient, SSRMultipartLink } from '@apollo/experimental-nextjs-app-support';
import { onError } from '@apollo/client/link/error';
import { API } from '@/utils';

function makeClient() {
  const httpLink = new HttpLink({
    uri: API.BASE_URL,
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) {
<<<<<<< HEAD
      graphQLErrors.forEach(({ message, locations, path }) =>
        console.log(
          `[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`
        )
      );
=======
      graphQLErrors.forEach(({ message, locations, path }) => console.log(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`));
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
    }
    if (networkError) console.log(`[Network error]: ${networkError}`);
  });

  return new ApolloClient({
    cache: new InMemoryCache({
      addTypename: false,
    }),
    devtools: {
      enabled: true,
    },
    link:
      typeof window === 'undefined'
        ? ApolloLink.from([
            new SSRMultipartLink({
              stripDefer: true,
            }),
            errorLink,
            httpLink,
          ])
        : ApolloLink.from([errorLink, httpLink]),
  });
}

export function ApolloWrapper({ children }: React.PropsWithChildren<{}>) {
<<<<<<< HEAD
  return (
    <ApolloNextAppProvider makeClient={makeClient}>
      {children}
    </ApolloNextAppProvider>
  );
=======
  return <ApolloNextAppProvider makeClient={makeClient}>{children}</ApolloNextAppProvider>;
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
}
