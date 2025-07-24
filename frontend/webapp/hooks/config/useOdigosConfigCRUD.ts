import { useConfig } from './useConfig';
import { useLazyQuery, useMutation } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { FetchedOdigosConfig, OdigosConfigInput } from '@/types';
import { GET_ODIGOS_CONFIG, UPDATE_ODIGOS_CONFIG } from '@/graphql';
import { Crud, OdigosConfig, StatusType } from '@odigos/ui-kit/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { mapFetchedOdigosConfig, mapOdigosConfigToInput } from '@/utils';

export const useOdigosConfigCRUD = () => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const [queryOdigosConfig] = useLazyQuery<{ odigosConfig: FetchedOdigosConfig }, {}>(GET_ODIGOS_CONFIG, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const [mutateUpdateOdigosConfig] = useMutation<{ updateOdigosConfig: boolean }, { odigosConfig: OdigosConfigInput }>(UPDATE_ODIGOS_CONFIG, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Update, message: error.cause?.message || error.message }),
  });

  const fetchOdigosConfig = async () => {
    const { error, data } = await queryOdigosConfig();

    if (error) {
      addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message });
    } else if (data?.odigosConfig) {
      return mapFetchedOdigosConfig(data.odigosConfig);
    }
  };

  const updateOdigosConfig = async (payload: Partial<OdigosConfig>) => {
    if (isReadonly) {
      addNotification({ type: StatusType.Warning, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
    } else {
      await mutateUpdateOdigosConfig({ variables: { odigosConfig: mapOdigosConfigToInput(payload) } });
      addNotification({ type: StatusType.Success, title: Crud.Update, message: 'Odigos configuration updated successfully' });
    }
  };

  return {
    fetchOdigosConfig,
    updateOdigosConfig,
  };
};
