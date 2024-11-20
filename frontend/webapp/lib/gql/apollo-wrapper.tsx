'use client';

import { ApolloLink, HttpLink } from '@apollo/client';
import { ApolloNextAppProvider, InMemoryCache, ApolloClient, SSRMultipartLink } from '@apollo/experimental-nextjs-app-support';
import { onError } from '@apollo/client/link/error';
import { API } from '@/utils';

function makeClient() {
  const httpLink = new HttpLink({
    uri: API.BASE_URL,
  });

  const errorLink = onError(({ graphQLErrors, networkError }) => {
    if (graphQLErrors) {
      graphQLErrors.forEach(({ message, locations, path }) => console.log(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`));
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
  return <ApolloNextAppProvider makeClient={makeClient}>{children}</ApolloNextAppProvider>;
}
