import { DropdownOption, K8sActualNamespace, K8sActualSource } from '@/types';
import { useEffect, useState } from 'react';
import { useComputePlatform, useNamespace } from '../compute-platform';

export const useConnectSourcesList = ({ stateMenu, setSourcesList }) => {
  const [namespacesList, setNamespacesList] = useState<DropdownOption[]>([]);

  const { error, data } = useComputePlatform();
  const { data: namespacesData } = useNamespace(
    stateMenu.selectedOption?.value,
    false
  );

  useEffect(() => {
    data && buildNamespacesList();
  }, [data, error]);

  useEffect(() => {
    if (namespacesData && namespacesData.k8sActualSources) {
      setSourcesList(namespacesData.k8sActualSources || []);
      stateMenu.setSelectAllCheckbox(
        namespacesData.k8sActualSources?.length ===
          stateMenu.selectedItems[stateMenu.selectedOption?.value || '']
            ?.length && namespacesData.k8sActualSources?.length > 0
      );
    }
  }, [namespacesData]);

  function buildNamespacesList() {
    const namespaces = data?.computePlatform?.k8sActualNamespaces || [];
    const namespacesList = namespaces.map((namespace: K8sActualNamespace) => ({
      id: namespace.name,
      value: namespace.name,
    }));

    stateMenu.setSelectedOption(namespacesList[0]);
    setNamespacesList(namespacesList);
  }
  return { namespacesList };
};
