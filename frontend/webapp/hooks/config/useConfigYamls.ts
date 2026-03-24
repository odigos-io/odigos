import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_CONFIG_YAMLS } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, FieldTypes } from '@odigos/ui-kit/types';

// TODO: once we released the kit with the types, we can remove these interfaces and import `FetchedConfigYamls` from the kit
interface ConfigYamlField {
  displayName: string;
  componentType: FieldTypes;
  isHelmOnly: boolean;
  description: string;
  helmValuePath: string;
  docsLink?: string | null;
  componentProps?: string | null;
}
interface ConfigYaml {
  name: string;
  displayName: string;
  fields: ConfigYamlField[];
}
interface FetchedConfigYamls {
  configYamls: ConfigYaml[];
}

export const useConfigYamls = () => {
  const { data, loading, error } = useQuery<FetchedConfigYamls>(GET_CONFIG_YAMLS);
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  return {
    configYamls: data?.configYamls || [],
    configYamlsLoading: loading,
  };
};
