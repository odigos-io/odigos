import { useQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import type { DescribeSource, WorkloadId } from '@/types';

const data = {
  describeSource: {
    name: {
      name: 'Name',
      value: 'coupon',
      status: null,
      explain: 'the name of the k8s workload object that this source describes',
    },
    kind: {
      name: 'Kind',
      value: 'deployment',
      status: null,
      explain: 'the kind of the k8s workload object that this source describes (deployment/daemonset/statefulset)',
    },
    namespace: {
      name: 'Namespace',
      value: 'default',
      status: null,
      explain: 'the namespace of the k8s workload object that this source describes',
    },
    labels: {
      instrumented: {
        name: 'Instrumented',
        value: 'false',
        status: null,
        explain: 'whether this workload is considered for instrumentation based on the presence of the odigos-instrumentation label',
      },
      workload: {
        name: 'Workload',
        value: 'unset',
        status: null,
        explain: 'the value of the odigos-instrumentation label on the workload object in k8s',
      },
      namespace: {
        name: 'Namespace',
        value: 'unset',
        status: null,
        explain: 'the value of the odigos-instrumentation label on the namespace object in k8s',
      },
      instrumentedText: {
        name: 'DecisionText',
        value: "Workload is NOT instrumented because neither the workload nor the namespace has the 'odigos-instrumentation' label set",
        status: null,
        explain: 'a human readable explanation of the decision to instrument or not instrument this workload',
      },
    },
    runtimeInfo: {
      containers: [
        {
          containerName: {
            name: 'Container Name',
            value: 'coupon',
            status: null,
            explain: 'the unique name of the container in the k8s pod',
          },
          language: {
            name: 'Programming Language',
            value: 'javascript',
            status: 'success',
            explain: 'the programming language detected by odigos to be running in this container',
          },
          runtimeVersion: {
            name: 'Runtime Version',
            value: '18.3.0',
            status: null,
            explain: 'the version of the runtime detected by odigos to be running in this container',
          },
          envVars: [],
        },
      ],
    },
    instrumentationConfig: {
      created: {
        name: 'Created',
        value: 'created',
        status: 'transitioning',
        explain: 'whether the instrumentation config object exists in the cluster. When a workload is labeled for instrumentation, an instrumentation config object is created',
      },
      createTime: {
        name: 'create time',
        value: '2025-01-21 14:04:00 +0200 IST',
        status: null,
        explain: 'the time when the instrumentation config object was created',
      },
      containers: [
        {
          containerName: {
            name: 'Container Name',
            value: 'coupon',
            status: null,
            explain: 'the unique name of the container in the k8s pod',
          },
          language: {
            name: 'Programming Language',
            value: 'javascript',
            status: 'success',
            explain: 'the programming language detected by odigos to be running in this container',
          },
          runtimeVersion: {
            name: 'Runtime Version',
            value: '18.3.0',
            status: null,
            explain: 'the version of the runtime detected by odigos to be running in this container',
          },
          envVars: [],
        },
      ],
    },
    instrumentationDevice: {
      statusText: {
        name: 'Status',
        value: 'Instrumentation device applied successfully',
        status: 'success',
        explain: 'the result of applying the instrumentation device to the workload manifest',
      },
      containers: [
        {
          containerName: {
            name: 'Container Name',
            value: 'coupon',
            status: null,
            explain: 'the unique name of the container in the k8s pod',
          },
          devices: {
            name: 'Devices',
            value: '[javascript-native-community]',
            status: null,
            explain: 'the odigos instrumentation devices that were added to the workload manifest',
          },
          originalEnv: [],
        },
      ],
    },
    totalPods: 1,
    podsPhasesCount: 'Running 1',
    pods: [
      {
        podName: {
          name: 'Pod Name',
          value: 'coupon-8497bfbc5f-qk9sp',
          status: null,
          explain: 'the name of the k8s pod object that is part of the source workload',
        },
        nodeName: {
          name: 'Node Name',
          value: 'kind-control-plane',
          status: null,
          explain: 'the name of the k8s node where the current pod being described is scheduled',
        },
        phase: {
          name: 'Phase',
          value: 'Running',
          status: 'success',
          explain: 'the current pod phase for the pod being described',
        },
        containers: [
          {
            containerName: {
              name: 'Container Name',
              value: 'coupon',
              status: null,
              explain: 'the unique name of a container being described in the pod',
            },
            actualDevices: {
              name: 'Actual Devices',
              value: '[javascript-native-community]',
              status: 'success',
              explain: 'the odigos instrumentation devices that were found on this pod container instance',
            },
            instrumentationInstances: [
              {
                healthy: {
                  name: 'Healthy',
                  value: 'true',
                  status: 'success',
                  explain: 'health indication for the instrumentation running for this process',
                },
                message: null,
                identifyingAttributes: [
                  {
                    name: 'service.instance.id',
                    value: '019488bf-bc72-788c-9813-5da651152b91',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'telemetry.sdk.language',
                    value: 'nodejs',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'process.runtime.version',
                    value: '18.3.0',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'telemetry.distro.version',
                    value: 'v1.0.143',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'process.pid',
                    value: '1',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'k8s.namespace.name',
                    value: 'default',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'k8s.container.name',
                    value: 'coupon',
                    status: null,
                    explain: null,
                  },
                  {
                    name: 'k8s.pod.name',
                    value: 'coupon-8497bfbc5f-qk9sp',
                    status: null,
                    explain: null,
                  },
                ],
              },
            ],
          },
        ],
      },
    ],
  },
};

export const useDescribeSource = ({ namespace, name, kind }: WorkloadId) => {
  // const { data, loading, error } = useQuery<DescribeSource>(DESCRIBE_SOURCE, {
  //   variables: { namespace, name, kind },
  //   pollInterval: 5000,
  // });

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
    loading: false,
    error: undefined,
    data: data?.describeSource,
    restructureForPrettyMode,
  };
};
