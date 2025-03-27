import { useEffect } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import type { NamespaceInstrumentInput } from '@/types';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, Namespace, StatusType } from '@odigos/ui-kit/types';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { GET_NAMESPACE, GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';

const allNamespaces = {
  computePlatform: {
    k8sActualNamespaces: [
      {
        name: 'anthos-identity-service',
        selected: false,
      },
      {
        name: 'buabookkeeping-ca',
        selected: false,
      },
      {
        name: 'ca-catalog-ipf-kubie',
        selected: false,
      },
      {
        name: 'ca-comp-po-monitoring',
        selected: false,
      },
      {
        name: 'ca-corpcomplianceplus',
        selected: false,
      },
      {
        name: 'ca-gif-flags',
        selected: false,
      },
      {
        name: 'ca-operator',
        selected: false,
      },
      {
        name: 'ca-plan-pricing-br',
        selected: false,
      },
      {
        name: 'ca-pricing-pcst',
        selected: false,
      },
      {
        name: 'ca-pricing-portal-hub',
        selected: false,
      },
      {
        name: 'ce-orchestra-ca',
        selected: false,
      },
      {
        name: 'cert-manager',
        selected: false,
      },
      {
        name: 'conflux-ca',
        selected: false,
      },
      {
        name: 'cpc-ca',
        selected: false,
      },
      {
        name: 'csis-grs-s2dcm-ca',
        selected: false,
      },
      {
        name: 'csis-lmd-dispatcher-ui-ca',
        selected: false,
      },
      {
        name: 'ctinsights-ca',
        selected: false,
      },
      {
        name: 'dctech-canada',
        selected: false,
      },
      {
        name: 'default',
        selected: false,
      },
      {
        name: 'ds-ca',
        selected: false,
      },
      {
        name: 'ereceipt-tracer',
        selected: false,
      },
      {
        name: 'flagger',
        selected: false,
      },
      {
        name: 'fs-api',
        selected: false,
      },
      {
        name: 'gcrm-mp-event-bus',
        selected: false,
      },
      {
        name: 'gcrm-mp-sf-ark-stream',
        selected: false,
      },
      {
        name: 'gcrm-mp-sf-sink',
        selected: false,
      },
      {
        name: 'heptio-qm',
        selected: false,
      },
      {
        name: 'istio-ca',
        selected: false,
      },
      {
        name: 'kube-public',
        selected: false,
      },
      {
        name: 'logging',
        selected: false,
      },
      {
        name: 'marquis-system',
        selected: false,
      },
      {
        name: 'modauto',
        selected: false,
      },
      {
        name: 'monitoring',
        selected: false,
      },
      {
        name: 'notification-verification',
        selected: false,
      },
      {
        name: 'opa',
        selected: false,
      },
      {
        name: 'parse-and-save-service',
        selected: false,
      },
      {
        name: 'process-receipt-service',
        selected: false,
      },
      {
        name: 'raas-chaosmart-nginx',
        selected: false,
      },
      {
        name: 'receipt-service',
        selected: false,
      },
      {
        name: 'sec-squad',
        selected: false,
      },
      {
        name: 'smp-system',
        selected: false,
      },
      {
        name: 'spirl-system',
        selected: false,
      },
      {
        name: 'sr-ratelimit-system',
        selected: false,
      },
      {
        name: 'sr-system',
        selected: false,
      },
      {
        name: 'sre-agent-system',
        selected: false,
      },
      {
        name: 'sre-chaos',
        selected: false,
      },
      {
        name: 'sre-system',
        selected: false,
      },
      {
        name: 'sre-wishlenz',
        selected: false,
      },
      {
        name: 'tunr',
        selected: false,
      },
      {
        name: 'velero',
        selected: false,
      },
      {
        name: 'wce',
        selected: false,
      },
      {
        name: 'wce-system',
        selected: false,
      },
      {
        name: 'wcl-sso',
        selected: false,
      },
      {
        name: 'wcl-tpr',
        selected: false,
      },
      {
        name: 'wcnp-addons-dummy',
        selected: false,
      },
      {
        name: 'worker-ca',
        selected: false,
      },
    ],
  },
};

export const useNamespace = (namespaceName?: string) => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const {
    refetch: queryAll,
    // data: allNamespaces,
    loading: allLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespaces?: Namespace[] } }>(GET_NAMESPACES, {
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  // TODO: change query, to lazy query
  const {
    refetch: querySingle,
    data: singleNamespace,
    loading: singleLoading,
  } = useQuery<{ computePlatform?: { k8sActualNamespace?: Namespace } }>(GET_NAMESPACE, {
    skip: !namespaceName,
    variables: { namespaceName },
    onError: (error) => addNotification({ type: StatusType.Error, title: error.name || Crud.Read, message: error.cause?.message || error.message }),
  });

  const [mutatePersist] = useMutation<{ persistK8sNamespace: boolean }>(PERSIST_NAMESPACE, {
    onError: (error) => {
      // TODO: after estimating the number of instrumentationConfigs to create for future apps in "useSourceCRUD" hook, then uncomment the below
      // setInstrumentCount('sourcesToCreate', 0);
      // setInstrumentCount('sourcesCreated', 0);
      // setInstrumentAwait(false);
      addNotification({ type: StatusType.Error, title: error.name || Crud.Update, message: error.cause?.message || error.message });
    },
  });

  const persistNamespace = async (payload: NamespaceInstrumentInput) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutatePersist({ variables: { namespace: payload } });
    }
  };

  useEffect(() => {
    if (!allNamespaces?.computePlatform?.k8sActualNamespaces?.length) queryAll();
  }, []);

  useEffect(() => {
    if (namespaceName && !singleLoading) querySingle({ namespaceName });
  }, [namespaceName]);

  const namespaces = allNamespaces?.computePlatform?.k8sActualNamespaces || [];
  const namespace = singleNamespace?.computePlatform?.k8sActualNamespace;

  return {
    loading: allLoading || singleLoading,
    namespaces,
    namespace,
    persistNamespace,
  };
};
