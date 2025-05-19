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
  const { dataStreams, setDataStreams, dataStreamsLoading, setDataStreamsLoading, selectedStreamName, setSelectedStreamName } = useDataStreamStore();

  const [fetchDataStreamsQuery] = useLazyQuery<{ computePlatform?: { dataStreams?: DataStream[] } }>(GET_DATA_STREAMS);

  const fetchDataStreams = async () => {
    setDataStreamsLoading(true);
    const { error, data } = await fetchDataStreamsQuery();
    setDataStreamsLoading(false);

    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    } else if (data?.computePlatform?.dataStreams) {
      setDataStreams(data.computePlatform.dataStreams);

      const streamNameFromStorage = sessionStorage.getItem('selectedStreamName');
      const storedSteamNameExistsInCP = data.computePlatform.dataStreams.some((stream) => stream.name === streamNameFromStorage);

      if (streamNameFromStorage && storedSteamNameExistsInCP) {
        setSelectedStreamName(streamNameFromStorage);
      } else {
        setSelectedStreamName('default')
      }
    }
  };

  useEffect(() => {
    if (selectedStreamName) sessionStorage.setItem('selectedStreamName', selectedStreamName);
  }, [selectedStreamName]);

  useEffect(() => {
    if (!dataStreams.length && !dataStreamsLoading) fetchDataStreams();
  }, []);

  return {
    dataStreamsLoading,
    dataStreams,
    fetchDataStreams,
  };
};
