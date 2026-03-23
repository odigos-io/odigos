import { useQuery } from '@apollo/client';
import { GET_CONFIG_YAMLS } from '@/graphql';

// TODO: once we released the kit with the types, we can remove these interfaces and import `FetchedConfigYamls` from the kit
interface ConfigYamlField {
  displayName: string;
  componentType: string;
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
  configYamls: {
    configs: ConfigYaml[];
  };
}

export const useConfigYamls = () => {
  const { data } = useQuery<FetchedConfigYamls>(GET_CONFIG_YAMLS);

  return {
    configYamls: data?.configYamls?.configs || [],
  };
};
