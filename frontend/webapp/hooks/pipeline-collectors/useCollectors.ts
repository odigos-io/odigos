import { useNotificationStore } from '@odigos/ui-kit/store';
import { type ApolloError, useLazyQuery } from '@apollo/client';
import { GET_GATEWAY_INFO, GET_GATEWAY_PODS, GET_NODE_COLLECTOR_INFO, GET_NODE_COLLECTOR_PODS, GET_COLLECTOR_POD_INFO } from '@/graphql';
import {
  Crud,
  type GatewayInfo,
  StatusType,
  type NodeCollectoInfo,
  type PodInfo,
  type ExtendedPodInfo,
  type GetGatewayInfo,
  type GetGatewayPods,
  type GetNodeCollectorInfo,
  type GetNodeCollectorPods,
  type GetExtendedPodInfo,
} from '@odigos/ui-kit/types';

interface UseCollectorsResult {
  getGatewayInfo: GetGatewayInfo;
  getGatewayPods: GetGatewayPods;
  getNodeCollectorInfo: GetNodeCollectorInfo;
  getNodeCollectorPods: GetNodeCollectorPods;
  getExtendedPodInfo: GetExtendedPodInfo;
}

export const useCollectors = (): UseCollectorsResult => {
  const { addNotification } = useNotificationStore();
  const onError = (error: ApolloError) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message });

  const [getGatewayInfo] = useLazyQuery<{ gatewayDeploymentInfo?: GatewayInfo }, {}>(GET_GATEWAY_INFO, { onError });
  const [getGatewayPods] = useLazyQuery<{ gatewayPods?: PodInfo[] }, {}>(GET_GATEWAY_PODS, { onError });
  const [getNodeCollectorInfo] = useLazyQuery<{ odigletDaemonSetInfo?: NodeCollectoInfo }, {}>(GET_NODE_COLLECTOR_INFO, { onError });
  const [getNodeCollectorPods] = useLazyQuery<{ odigletPods?: PodInfo[] }, {}>(GET_NODE_COLLECTOR_PODS, { onError });
  const [getExtendedPodInfo] = useLazyQuery<{ collectorPod?: ExtendedPodInfo }, { namespace: string; name: string }>(GET_COLLECTOR_POD_INFO, { onError });

  return {
    getGatewayInfo: () => getGatewayInfo().then((result) => result.data?.gatewayDeploymentInfo),
    getGatewayPods: () => getGatewayPods().then((result) => result.data?.gatewayPods),
    getNodeCollectorInfo: () => getNodeCollectorInfo().then((result) => result.data?.odigletDaemonSetInfo),
    getNodeCollectorPods: () => getNodeCollectorPods().then((result) => result.data?.odigletPods),
    getExtendedPodInfo: (namespace: string, name: string) => getExtendedPodInfo({ variables: { namespace, name } }).then((result) => result.data?.collectorPod),
  };
};
