import { AxiosError } from 'axios';
import { useMutation } from 'react-query';
import { checkConnection } from '@/services';

export function useCheckConnection() {
  const { mutateAsync, isLoading } = useMutation(checkConnection);

  const checkDestinationConnection = async (body, callback) => {
    try {
      await mutateAsync(body, {
        onSuccess: (res) =>
          callback({
            enabled: true,
            message: res.message,
          }),
        onError: (error: AxiosError) =>
          callback({
            enabled: false,
            message: 'Please check your input and try again.',
          }),
      });
    } catch (error) {}
  };

  return {
    isLoading,
    checkDestinationConnection,
  };
}
