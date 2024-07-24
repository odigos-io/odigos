// lib/client.js
import { HttpLink, InMemoryCache, ApolloClient } from '@apollo/client';
import { registerApolloClient } from '@apollo/experimental-nextjs-app-support';

export const { getClient } = registerApolloClient(() => {
  return new ApolloClient({
    cache: new InMemoryCache(),
    link: new HttpLink({
      uri: 'https://main--time-pav6zq.apollographos.net/graphql',
    }),
  });
});
