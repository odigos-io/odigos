import { DOWNLOAD_DIAGNOSE } from '@/graphql';
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

  return {
    downloadDiagnose: async (payload: DiagnoseFormData) => {
      await downloadDiagnose({ variables: { input: payload } });
    },
  };
};
