import { useQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import type { DescribeSource, WorkloadId } from '@/types';

export const useDescribeSource = ({ namespace, name, kind }: WorkloadId) => {
  const { data, loading, error } = useQuery<DescribeSource>(DESCRIBE_SOURCE, {
    variables: { namespace, name, kind },
    pollInterval: 5000,
  });

  // This function is used to restructure the data, so that it reflects the output given by "odigos describe" command in the CLI.
  // This is not really needed, but it's a nice-to-have feature to make the data more readable.
  const restructureForPrettyMode = (code?: DescribeSource['describeSource']) => {
    if (!code) return {};

    const payload: Record<string, any> = {};

    const mapObjects = (obj: any, category?: string, options?: { keyPrefix?: string }) => {
      if (typeof obj === 'object' && !!obj?.name) {
        let key = options?.keyPrefix ? `${options?.keyPrefix}${obj.name}` : obj.name;
        let val = obj.value;

        if (obj.explain) key += `@tooltip=${obj.explain}`;
        if (obj.status) val += `@status=${obj.status}`;
        else val += '@status=none';

        if (!!category && !payload[category]) payload[category] = {};
        if (!!category) payload[category][key] = val;
        else payload[key] = val;
      }
    };

    Object.values(code).forEach((val) => mapObjects(val));
    Object.values(code.labels).forEach((val) => mapObjects(val, 'Labels'));
    Object.values(code.instrumentationConfig).forEach((val) => mapObjects(val, 'Instrumentation Config'));
    code.runtimeInfo?.containers.forEach((obj, i) => Object.values(obj).forEach((val) => mapObjects(val, 'Runtime Info', { keyPrefix: `Container #${i + 1} - ` })));
    Object.values(code.instrumentationDevice).forEach((val) => mapObjects(val, 'Instrumentation Device'));
    code.instrumentationDevice?.containers.forEach((obj, i) => Object.values(obj).forEach((val) => mapObjects(val, 'Instrumentation Device', { keyPrefix: `Container #${i + 1} - ` })));

    payload['Pods'] = { 'Total Pods': `${code.totalPods}@status=none` };
    code.pods.forEach((obj) => {
      Object.values(obj).forEach((val) => mapObjects(val, 'Pods'));
      obj.containers.forEach((containers, i) => {
        Object.values(containers).forEach((val) => mapObjects(val, 'Pods', { keyPrefix: `Container #${i + 1} - ` }));
        containers.instrumentationInstances.forEach((obj, i) => Object.values(obj).forEach((val) => mapObjects(val, 'Pods', { keyPrefix: `Instrumentation Instance #${i + 1} - ` })));
      });
    });

    return payload;
  };

  return {
    loading,
    error,
    data: data?.describeSource,
    restructureForPrettyMode,
  };
};
