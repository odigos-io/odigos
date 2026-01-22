import { DOWNLOAD_DIAGNOSE } from '@/graphql';
import { API } from '@/utils';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, type DiagnoseFormData } from '@odigos/ui-kit/types';

interface DiagnoseResponse {
  stats: {
    fileCount: number;
    totalSizeBytes: number;
    totalSizeHuman: string;
  };
}

export const useDiagnose = () => {
  const { addNotification } = useNotificationStore();

  const [downloadDiagnose] = useLazyQuery<{ diagnose: DiagnoseResponse }, { input: DiagnoseFormData; dryRun?: boolean }>(DOWNLOAD_DIAGNOSE, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const downloadFile = async () => {
    try {
      const response = await fetch(`${API.BACKEND_HTTP_ORIGIN}/diagnose/download`);
      if (!response.ok) throw new Error('Failed to download diagnose file');

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      window.open(url, '_blank');
    } catch (error) {
      console.error(error);
      addNotification({ type: StatusType.Error, title: 'Error', message: 'Failed to download diagnose file' });
    }
  };

  return {
    downloadDiagnose: async (payload: DiagnoseFormData) => {
      const { data } = await downloadDiagnose({ variables: { input: payload } });

      if (data?.diagnose?.stats?.fileCount) {
        downloadFile();
      } else {
        addNotification({ type: StatusType.Error, title: 'Error', message: 'No diagnose data available' });
      }
    },
  };
};
