import { NOTIFICATION, ROUTES, SETUP } from '@/utils';
import { useMutation } from 'react-query';
import { setDestination } from '@/services';
import { useSelector } from 'react-redux';
import { useRouter } from 'next/navigation';
import { useNotification } from '../useNotification';
import { useSources } from '../sources';

export function useConnect() {
  const router = useRouter();
  const { show } = useNotification();
  const { upsertSources } = useSources();
  const sectionData = useSelector(({ app }) => app.sources);

  // Extracted error handling function
  const handleError = (error, defaultErrorMessage = SETUP.ERROR) => {
    const message = error?.response?.data?.message || defaultErrorMessage;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  };

  const { mutateAsync } = useMutation(setDestination);

  const connect = async (body) => {
    try {
      await upsertSources({
        sectionData,
        onSuccess: () => {},
        onError: (error) => handleError(error),
      });

      await mutateAsync(body, {
        onSuccess: () => router.push(ROUTES.OVERVIEW),
        onError: (error) => handleError(error),
      });
    } catch (error) {
      handleError(error);
    }
  };

  return { connect };
}
