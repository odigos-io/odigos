import { useEffect } from 'react';
import { GET_DATA_STREAMS } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { Crud, type DataStream, StatusType } from '@odigos/ui-kit/types';
import { useDataStreamStore, useNotificationStore } from '@odigos/ui-kit/store';

interface UseDataStreamsCrud {
  dataStreamsLoading: boolean;
  dataStreams: DataStream[];
  fetchDataStreams: () => Promise<void>;
}

export const useDataStreamsCRUD = (): UseDataStreamsCrud => {
  const { addNotification } = useNotificationStore();
  const { dataStreams, setDataStreams, selectedStreamName, setSelectedStreamName } = useDataStreamStore();

  const [fetchDataStreamsQuery, { loading, data }] = useLazyQuery<{ dataStreams?: DataStream[] }>(GET_DATA_STREAMS);

  const fetchDataStreams = async () => {
    const { error, data } = await fetchDataStreamsQuery();

    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    } else if (data?.dataStreams) {
      setDataStreams(data.dataStreams);
      if (!selectedStreamName) setSelectedStreamName('default');
    }
  };

  useEffect(() => {
    if (!dataStreams.length && !loading) fetchDataStreams();
  }, []);

  return {
    dataStreamsLoading: loading,
    dataStreams: data?.dataStreams || [],
    fetchDataStreams,
  };
};
