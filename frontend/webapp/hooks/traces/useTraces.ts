import type { Trace } from '@/types';
import { GET_TRACES } from '@/graphql';
import { useQuery } from '@apollo/client';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { useNotificationStore } from '@odigos/ui-kit/store';

interface UseTracesParams {
  serviceName: string;
}

export const useTraces = ({ serviceName }: UseTracesParams) => {
  const { addNotification } = useNotificationStore();

  const { data } = useQuery<{ getTraces: Trace[] }>(GET_TRACES, {
    variables: { serviceName },
    skip: !serviceName,
    onError: (error) =>
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      }),
  });

  return {
    traces: data?.getTraces ?? [],
  };
};
