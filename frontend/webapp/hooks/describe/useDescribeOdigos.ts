import { useQuery } from '@apollo/client';
import { DESCRIBE_ODIGOS } from '@/graphql';
import type { DescribeOdigos } from '@/types';

export const useDescribeOdigos = () => {
  const { data, loading, error } = useQuery<DescribeOdigos>(DESCRIBE_ODIGOS, {
    pollInterval: 5000,
  });

  // This function is used to restructure the data, so that it reflects the output given by "odigos describe" command in the CLI.
  // This is not really needed, but it's a nice-to-have feature to make the data more readable.
  const restructureForPrettyMode = (code?: DescribeOdigos['describeOdigos']) => {
    if (!code) return {};

    const payload: Record<string, any> = {
      [code.odigosVersion.name]: code.odigosVersion.value,
      'Number Of Sources': code.numberOfSources,
      'Number Of Destinations': code.numberOfDestinations,
    };

    const mapObjects = (obj: any, objectName: string) => {
      if (typeof obj === 'object' && !!obj?.name) {
        let key = obj.name;
        let val = obj.value;

        if (obj.explain) key += `#tooltip=${obj.explain}`;
        if (obj.status) val += `#status=${obj.status}`;
        else val += '#status=none';

        if (!payload[objectName]) payload[objectName] = {};
        payload[objectName][key] = val;
      }
    };

    Object.values(code.clusterCollector).forEach((val) => mapObjects(val, 'Cluster Collector'));
    Object.values(code.nodeCollector).forEach((val) => mapObjects(val, 'Node Collector'));

    return payload;
  };

  return {
    loading,
    error,
    data: data?.describeOdigos,
    restructureForPrettyMode,
  };
};
