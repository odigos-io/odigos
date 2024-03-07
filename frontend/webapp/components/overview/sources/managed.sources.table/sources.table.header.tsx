import React, { useEffect, useMemo, useState } from 'react';
import theme from '@/styles/palette';
import styled from 'styled-components';
import { Namespace, SourceSortOptions } from '@/types';
import { OVERVIEW } from '@/utils';
import { UnFocusSources } from '@/assets/icons/side.menu';
import { ActionsGroup, KeyvalText } from '@/design.system';

const StyledThead = styled.thead`
  background-color: ${theme.colors.light_dark};
`;

const StyledTh = styled.th`
  padding: 10px 20px;
  text-align: left;
  border-bottom: 1px solid ${theme.colors.blue_grey};
`;

const StyledMainTh = styled(StyledTh)`
  padding: 10px 20px;
  display: flex;
  align-items: center;
  gap: 8px;
`;

const ActionGroupContainer = styled.div`
  display: flex;
  justify-content: flex-end;
  padding-right: 20px;
  gap: 24px;
  flex: 1;
`;

interface ActionsTableHeaderProps {
  data: any[];
  sortSources?: (condition: string) => void;
  filterSourcesByNamespace?: (namespaces: string[]) => void;
  toggleActionStatus?: (ids: string[], disabled: boolean) => void;
  namespaces?: Namespace[];
}

export function SourcesTableHeader({
  data,
  namespaces,
  sortSources,
  filterSourcesByNamespace,
}: ActionsTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');
  const [groupNamespaces, setGroupNamespaces] = useState<string[]>([]);

  useEffect(() => {
    if (namespaces) {
      console.log({ object: namespaces });
      setGroupNamespaces(
        namespaces.filter((item) => item.totalApps > 0).map((item) => item.name)
      );
    }
  }, [namespaces]);

  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortSources && sortSources(id);
  }

  function onGroupClick(id: string) {
    let newGroup: string[] = [];
    if (groupNamespaces.includes(id)) {
      setGroupNamespaces(groupNamespaces.filter((item) => item !== id));
      newGroup = groupNamespaces.filter((item) => item !== id);
    } else {
      setGroupNamespaces([...groupNamespaces, id]);
      newGroup = [...groupNamespaces, id];
    }

    filterSourcesByNamespace && filterSourcesByNamespace(newGroup);
  }

  const sourcesGroups = useMemo(() => {
    if (!namespaces) return [];

    const totalNamespacesWithApps = namespaces.filter(
      (item) => item.totalApps > 0
    ).length;

    const namespacesItems = namespaces
      .sort((a, b) => b.totalApps - a.totalApps)
      ?.map((item: Namespace, index: number) => ({
        label: `${item.name} (${item.totalApps} apps) `,
        onClick: () => onGroupClick(item.name),
        id: item.name,
        selected: groupNamespaces.includes(item.name) && item.totalApps > 0,
        disabled:
          (groupNamespaces.length === 1 &&
            groupNamespaces.includes(item.name)) ||
          item.totalApps === 0 ||
          totalNamespacesWithApps === 1,
      }));

    return [
      {
        label: 'Namespaces',
        subTitle: 'Display',
        items: namespacesItems,
        condition: true,
      },
      {
        label: 'Sort by',
        subTitle: 'Sort by',
        items: [
          {
            label: 'Kind',
            onClick: () => onSortClick(SourceSortOptions.KIND),
            id: SourceSortOptions.KIND,
            selected: currentSortId === SourceSortOptions.KIND,
          },
          {
            label: 'Language',
            onClick: () => onSortClick(SourceSortOptions.LANGUAGE),
            id: SourceSortOptions.LANGUAGE,
            selected: currentSortId === SourceSortOptions.LANGUAGE,
          },
          {
            label: 'Name',
            onClick: () => onSortClick(SourceSortOptions.NAME),
            id: SourceSortOptions.NAME,
            selected: currentSortId === SourceSortOptions.NAME,
          },
          {
            label: 'Namespace',
            onClick: () => onSortClick(SourceSortOptions.NAMESPACE),
            id: SourceSortOptions.NAMESPACE,
            selected: currentSortId === SourceSortOptions.NAMESPACE,
          },
        ],
        condition: true,
      },
    ];
  }, [namespaces, groupNamespaces, data]);

  return (
    <StyledThead>
      <StyledMainTh>
        <UnFocusSources style={{ width: 18, height: 18 }} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.ACTIONS}`}
        </KeyvalText>
        <ActionGroupContainer>
          <ActionsGroup actionGroups={sourcesGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
