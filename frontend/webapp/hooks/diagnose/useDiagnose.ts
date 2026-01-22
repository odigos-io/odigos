import { DOWNLOAD_DIAGNOSE } from '@/graphql';
import { API, downloadFileFromURL } from '@/utils';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, type DiagnoseFormData } from '@odigos/ui-kit/types';
import { useCallback } from 'react';

interface DiagnoseResponse {
  stats: {
    fileCount: number;
    totalSizeBytes: number;
    totalSizeHuman: string;
  };
}

export const useDiagnose = () => {
  const { addNotification } = useNotificationStore();

  const [prepareFile] = useLazyQuery<{ diagnose: DiagnoseResponse }, { input: DiagnoseFormData; dryRun?: boolean }>(DOWNLOAD_DIAGNOSE, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const downloadDiagnose = useCallback(async (payload: DiagnoseFormData) => {
    // 1st we prepare the file, this will create a temporary directory with the diagnose data
    const { data } = await prepareFile({ variables: { input: payload } });

    if (!data?.diagnose?.stats?.fileCount) {
      addNotification({ type: StatusType.Error, title: 'Error', message: 'No diagnose data available' });
    } else {
      try {
        // then we get the file from the backend
        const response = await fetch(`${API.BACKEND_HTTP_ORIGIN}/diagnose/download`);
        if (!response.ok) throw new Error('Failed to download diagnose file');

        // then we create a blob URL from the file
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);

        // finally we download the file
        downloadFileFromURL(url);
      } catch (error) {
        console.error(error);
        addNotification({ type: StatusType.Error, title: 'Error', message: 'Failed to download diagnose file' });
      }
    }
  }, []);

  return {
    downloadDiagnose,
  };
};
