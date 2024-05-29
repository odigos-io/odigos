import { NOTIFICATION, ROUTES, SETUP } from '@/utils';
import { useMutation } from 'react-query';
import { setDestination } from '@/services';
import { useSelector } from 'react-redux';
import { useRouter } from 'next/navigation';
import { useSources } from '../sources';

export function useConnect() {
  const router = useRouter();

  const { upsertSources } = useSources();
  const { mutateAsync } = useMutation(setDestination);
  const sectionData = useSelector(({ app }) => app.sources);

  const connect = async (body) => {
    try {
      await upsertSources({
        sectionData,
        onSuccess: () => {},
        onError: null,
      });

      await mutateAsync(body, {
        onSuccess: () => router.push(`${ROUTES.OVERVIEW}?poll=true`),
        onError: () => {},
      });
    } catch (error) {}
  };

  return { connect };
}
