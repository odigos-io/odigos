import { useEffect } from 'react';
import { useConfig } from '../config';
import { useSourceCRUD } from '../sources';
import { useLazyQuery, useMutation } from '@apollo/client';
import { Crud, type DataStream, StatusType } from '@odigos/ui-kit/types';
import { useDataStreamStore, useNotificationStore } from '@odigos/ui-kit/store';
import { DELETE_DATA_STREAM, GET_DATA_STREAMS, UPDATE_DATA_STREAM } from '@/graphql';
import { DEFAULT_DATA_STREAM_NAME, DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';

interface UseDataStreamsCrud {
  dataStreamsLoading: boolean;
  dataStreams: DataStream[];
  selectedStreamName: string;
  fetchDataStreams: () => Promise<void>;
  updateDataStream: (dataStreamName: string, dataStream: DataStream) => Promise<void>;
  deleteDataStream: (dataStreamName: string) => Promise<void>;
}

export const useDataStreamsCRUD = (): UseDataStreamsCrud => {
  const { isReadonly } = useConfig();
  const { fetchSourcesPaginated } = useSourceCRUD();
  const { addNotification } = useNotificationStore();
  const { dataStreams, setDataStreams, addDataStreams, removeDataStreams, dataStreamsLoading, setDataStreamsLoading, selectedStreamName, setSelectedStreamName } = useDataStreamStore();

  const notifyUser = (type: StatusType, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const [fetchDataStreamsQuery] = useLazyQuery<{ computePlatform?: { dataStreams?: DataStream[] } }>(GET_DATA_STREAMS);

  const fetchDataStreams = async (overrideStreamNameSelection?: string) => {
    setDataStreamsLoading(true);
    const { error, data } = await fetchDataStreamsQuery();
    setDataStreamsLoading(false);

    if (error) {
      notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message);
    } else if (data?.computePlatform?.dataStreams) {
      setDataStreams(data.computePlatform.dataStreams);

      if (overrideStreamNameSelection) {
        setSelectedStreamName(overrideStreamNameSelection);
      } else {
        const streamNameFromStorage = sessionStorage.getItem('selectedStreamName');
        const storedSteamNameExistsInCP = data.computePlatform.dataStreams.some((stream) => stream.name === streamNameFromStorage);

        if (streamNameFromStorage && storedSteamNameExistsInCP) {
          setSelectedStreamName(streamNameFromStorage);
        } else {
          setSelectedStreamName('default');
        }
      }
    }
  };

  const [mutateUpdate] = useMutation<{ updateDataStream: { name: string } }, { id: string; dataStream: DataStream }>(UPDATE_DATA_STREAM, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message),
    onCompleted: async (res, req) => {
      const oldStream = dataStreams.find((x) => x.name === req?.variables?.id);
      if (oldStream) removeDataStreams([oldStream]);
      const newStream = res.updateDataStream;
      addDataStreams([newStream]);
      notifyUser(StatusType.Success, Crud.Update, `Successfully updated "${oldStream?.name}" data stream`);

      const switchToStream = selectedStreamName === oldStream?.name ? newStream.name : selectedStreamName;
      await fetchDataStreams(switchToStream);
      await fetchSourcesPaginated();
      // We don't need to refetch destinations, because it's refetched with SSE for modified events.
      // The reason we do refetch sources, is because the stream name is never written to instrumentation configs.
    },
  });

  const [mutateDelete] = useMutation<{ deleteDataStream: boolean }, { id: string }>(DELETE_DATA_STREAM, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Delete, error.cause?.message || error.message),
    onCompleted: async (res, req) => {
      const oldStream = dataStreams.find((x) => x.name === req?.variables?.id);
      if (oldStream) removeDataStreams([oldStream]);
      notifyUser(StatusType.Success, Crud.Delete, `Successfully deleted "${oldStream?.name}" data stream`);

      const switchToStream = selectedStreamName === oldStream?.name ? DEFAULT_DATA_STREAM_NAME : selectedStreamName;
      await fetchDataStreams(switchToStream);
      await fetchSourcesPaginated();
      // We don't need to refetch destinations, because it's refetched with SSE for modified events.
      // The reason we do refetch sources, is because the stream name is never written to instrumentation configs.
    },
  });

  const updateDataStream: UseDataStreamsCrud['updateDataStream'] = async (dataStreamName, dataStream) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Updating data stream...', undefined, true);
      await mutateUpdate({ variables: { id: dataStreamName, dataStream } });
    }
  };

  const deleteDataStream: UseDataStreamsCrud['deleteDataStream'] = async (dataStreamName) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
    } else {
      notifyUser(StatusType.Default, 'Pending', 'Deleting data stream...', undefined, true);
      await mutateDelete({ variables: { id: dataStreamName } });
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
    selectedStreamName,
    fetchDataStreams,
    updateDataStream,
    deleteDataStream,
  };
};
