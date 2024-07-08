import { useMutation, useQuery } from 'react-query';
import { checkConnection } from '@/services';

export function useCheckConnection() {
  const { mutateAsync, isLoading } = useMutation(checkConnection);

  const checkDestinationConnection = async (body) => {
    console.log('checkDestinationConnection', body);

    await mutateAsync(body, {
      onSuccess: (res) => console.log({ res }),
      onError: (error) => console.log({ error }),
    });
  };

  return {
    isLoading,
    checkDestinationConnection,
  };
}
