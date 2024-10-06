import React, { useEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { Drawer, KeyvalLoader } from '@/design.system';
import { NOTIFICATION, QUERIES } from '@/utils';
import { getApplication, getNamespaces } from '@/services';
import { SourcesList, SourcesOptionMenu } from '@/components/setup';
import {
  LoaderWrapper,
  SectionContainerWrapper,
} from './sources.section.styled';
import {
  Namespace,
  SourceConfig,
  NamespaceConfiguration,
  SelectedSourcesConfiguration,
} from '@/types';
import { FastSourcesSelection } from './fast-sources-selection';

const DEFAULT = 'default';

export function SourcesSection({ sectionData, setSectionData }) {
  const [currentNamespace, setCurrentNamespace] = useState<Namespace>();
  const [searchFilter, setSearchFilter] = useState<string>('');
  const [isDrawerOpen, setDrawerOpen] = useState(false);

  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_NAMESPACES],
    getNamespaces
  );

  useEffect(() => {
    if (!currentNamespace && data) {
      const currentNamespace =
        data?.namespaces.find((item: Namespace) => item.name === DEFAULT) ??
        data?.namespaces[0];
      setCurrentNamespace(currentNamespace);
    }
  }, [data]);

  useEffect(() => {
    onNameSpaceChange();
  }, [currentNamespace]);

  const namespacesList = useMemo(
    () =>
      data?.namespaces?.map((item: Namespace, index: number) => ({
        id: index,
        label: item.name,
      })),
    [data]
  );

  const sourceData = useMemo(() => {
    if (!currentNamespace) return;

    let namespace = sectionData[currentNamespace?.name];

    //filter by search query
    namespace = searchFilter
      ? namespace?.objects.filter((item: SourceConfig) =>
          item.name.toLowerCase().includes(searchFilter.toLowerCase())
        )
      : namespace?.objects;
    //remove instrumented applications
    return namespace?.filter(
      (item: SourceConfig) => !item.instrumentation_effective
    );
  }, [searchFilter, currentNamespace, sectionData]);

  const toggleDrawer = () => setDrawerOpen(!isDrawerOpen);

  async function onNameSpaceChange() {
    if (!currentNamespace || sectionData[currentNamespace?.name]) return;
    const namespace = await getApplication(currentNamespace?.name);

    const { selected } = data.namespaces.find(
      (item: Namespace) => item.name === currentNamespace?.name
    );

    const newSelectedNamespace = {
      ...sectionData,
      [currentNamespace?.name]: {
        selected_all: selected,
        future_selected: selected,
        objects: [...namespace?.applications],
      },
    };
    setSectionData(newSelectedNamespace);
    if (selected) {
      onSelectAllChange(true, newSelectedNamespace);
    }
  }

  function handleSourceClick({ item }: { item: SourceConfig }) {
    if (!currentNamespace) return;

    const objIndex = sectionData[currentNamespace?.name].objects.findIndex(
      (app: SourceConfig) => app.name === item.name
    );

    const namespace = sectionData[currentNamespace?.name];
    let objects = [...namespace.objects];

    // Make a shallow copy of the object to ensure it's extensible
    let objectToUpdate = { ...objects[objIndex] };

    objectToUpdate.selected = !objectToUpdate.selected;
    objects[objIndex] = objectToUpdate;

    let currentNamespaceConfig = {
      ...namespace,
      objects,
    };

    if (!objectToUpdate.selected && namespace.selected_all) {
      currentNamespaceConfig = {
        ...currentNamespaceConfig,
        selected_all: false,
        future_selected: false,
      };
    }
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onSelectAllChange(
    value: boolean,
    data?: SelectedSourcesConfiguration
  ) {
    if (!currentNamespace) return;

    const currentData = data || sectionData;
    const namespace = currentData[currentNamespace?.name];
    let objects = namespace.objects.map((item: SourceConfig) => ({
      ...item,
      selected: value,
    }));

    const currentNamespaceConfig = {
      ...namespace,
      future_selected: value,
      selected_all: value,
      objects,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function onFutureApplyChange(value: boolean) {
    if (!currentNamespace) return;

    const currentNamespaceConfig = {
      ...sectionData[currentNamespace?.name],
      future_selected: value,
    };
    handleSetNewSelectedConfig(currentNamespaceConfig);
  }

  function handleSetNewSelectedConfig(config: NamespaceConfiguration) {
    if (!currentNamespace) return;

    const newSelectedNamespaceConfig: SelectedSourcesConfiguration = {
      ...sectionData,
      [currentNamespace?.name]: config,
    };

    setSectionData(newSelectedNamespaceConfig);
  }

  if (isLoading || currentNamespace === undefined) {
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );
  }

  return (
    <SectionContainerWrapper>
      {isDrawerOpen && (
        <Drawer
          isOpen={isDrawerOpen}
          onClose={toggleDrawer}
          position="right"
          width="500px"
        >
          <FastSourcesSelection {...{ sectionData, setSectionData }} />
        </Drawer>
      )}
      <SourcesOptionMenu
        currentNamespace={currentNamespace}
        setCurrentItem={setCurrentNamespace}
        data={namespacesList}
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
        onSelectAllChange={onSelectAllChange}
        selectedApplications={sectionData}
        onFutureApplyChange={onFutureApplyChange}
        toggleFastSourcesSelection={toggleDrawer}
      />
      <SourcesList
        data={sourceData}
        selectedData={sectionData[currentNamespace?.name]}
        onItemClick={handleSourceClick}
        onClearClick={() => onSelectAllChange(false)}
      />
    </SectionContainerWrapper>
  );
}
