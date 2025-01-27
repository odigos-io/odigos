'use client';

import { API } from '@/utils';
import { onError } from '@apollo/client/link/error';
import { ApolloLink, HttpLink } from '@apollo/client';
import { ApolloNextAppProvider, InMemoryCache, ApolloClient, SSRMultipartLink } from '@apollo/experimental-nextjs-app-support';

function makeClient() {
  const apolloLinks = [
    // This link is used to send requests to the GraphQL server.
    new HttpLink({ uri: API.GRAPHQL }),

    // This link is used to log errors to the console.
    onError(({ graphQLErrors, networkError }) => {
      if (graphQLErrors) graphQLErrors.forEach(({ message, locations, path }) => console.warn(`[GraphQL error]: Message: ${message}, Location: ${locations}, Path: ${path}`));
      if (networkError) console.warn(`[Network error]: ${networkError}`);
    }),
  ];

  if (typeof window === 'undefined') {
    apolloLinks.unshift(new SSRMultipartLink({ stripDefer: true }));
  }

  return new ApolloClient({
    devtools: { enabled: true },
    cache: new InMemoryCache({ addTypename: false }),
    link: ApolloLink.from(apolloLinks),
  });
}

export function ApolloWrapper({ children }: React.PropsWithChildren<{}>) {
  return <ApolloNextAppProvider makeClient={makeClient}>{children}</ApolloNextAppProvider>;
}
