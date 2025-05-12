import { useEffect } from 'react';
import { GET_DATA_STREAMS } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, type DataStream, StatusType } from '@odigos/ui-kit/types';

interface UseDataStreamsCrud {
  dataStreamsLoading: boolean;
  dataStreams: DataStream[];
  fetchDataStreams: () => Promise<void>;
}

export const useDataStreamsCRUD = (): UseDataStreamsCrud => {
  const { addNotification } = useNotificationStore();

  const [fetchDataStreamsQuery, { loading, data, called }] = useLazyQuery<{ dataStreams?: DataStream[] }>(GET_DATA_STREAMS);

  const fetchDataStreams = async () => {
    const { error } = await fetchDataStreamsQuery();

    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  };

  useEffect(() => {
    if (!called) fetchDataStreams();
  }, []);

  return {
    dataStreamsLoading: loading,
    dataStreams: data?.dataStreams || [],
    fetchDataStreams,
  };
};
