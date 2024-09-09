import React, { useEffect, useMemo, useState } from 'react';
import { OVERVIEW } from '@/utils';
import theme from '@/styles/palette';
import styled from 'styled-components';
import {
  ActionsGroup,
  KeyvalCheckbox,
  KeyvalLink,
  KeyvalSwitch,
  KeyvalText,
} from '@/design.system';
import { ManagedSource, Namespace } from '@/types';
import { UnFocusSourcesIcon } from '@keyval-dev/design-system';
import { useSources } from '@/hooks';

enum K8SSourceTypes {
  DEPLOYMENT = 'deployment',
  STATEFUL_SET = 'statefulset',
  DAEMON_SET = 'daemonset',
}
enum SourceSortOptions {
  NAME = 'name',
  KIND = 'kind',
  NAMESPACE = 'namespace',
  LANGUAGE = 'language',
}
const StyledThead = styled.div`
  background-color: ${theme.colors.light_dark};
  border-top-right-radius: 6px;
  border-top-left-radius: 6px;
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
const SELECT_ALL_CHECKBOX = 'select_all';
interface ActionsTableHeaderProps {
  data: ManagedSource[];
  namespaces?: Namespace[];
  sortSources?: (condition: string) => void;
  filterSourcesByKind?: (kinds: string[]) => void;
  filterSourcesByNamespace?: (namespaces: string[]) => void;
  selectedCheckbox: string[];
  onSelectedCheckboxChange: (id: string) => void;
  deleteSourcesHandler: () => void;
  filterSourcesByLanguage?: (languages: string[]) => void;
  filterByConditionStatus?: (status: 'All' | 'True' | 'False') => void;
  filterByConditionMessage: (message: string[]) => void;
}

export function SourcesTableHeader({
  data,
  namespaces,
  sortSources,
  filterSourcesByKind,
  filterSourcesByNamespace,
  filterSourcesByLanguage,
  deleteSourcesHandler,
  selectedCheckbox,
  onSelectedCheckboxChange,
  filterByConditionStatus,
  filterByConditionMessage,
}: ActionsTableHeaderProps) {
  const [currentSortId, setCurrentSortId] = useState('');
  const [groupNamespaces, setGroupNamespaces] = useState<string[]>([]);
  const [showSourcesWithIssues, setShowSourcesWithIssues] = useState(false);
  const [groupErrorMessage, setGroupErrorMessage] = useState<string[]>([]);
  const [groupLanguages, setGroupLanguages] = useState<string[]>([
    'javascript',
    'python',
    'java',
    'go',
    'dotnet',
  ]);
  const [groupKinds, setGroupKinds] = useState<string[]>([
    K8SSourceTypes.DEPLOYMENT,
    K8SSourceTypes.STATEFUL_SET,
    K8SSourceTypes.DAEMON_SET,
  ]);

  const { groupErrorMessages } = useSources();

  useEffect(() => {
    if (!filterByConditionStatus) {
      return;
    }

    setGroupErrorMessage(groupErrorMessages());

    showSourcesWithIssues
      ? filterByConditionStatus('False')
      : filterByConditionStatus('All');
  }, [showSourcesWithIssues, data]);

  useEffect(() => {
    if (namespaces) {
      setGroupNamespaces(
        namespaces.filter((item) => item.totalApps > 0).map((item) => item.name)
      );
    }
  }, [namespaces]);

  function onSortClick(id: string) {
    setCurrentSortId(id);
    sortSources && sortSources(id);
  }

  function onNamespaceClick(id: string) {
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

  function onKindClick(id: string) {
    let newGroup: string[] = [];
    if (groupKinds.includes(id)) {
      setGroupKinds(groupKinds.filter((item) => item !== id));
      newGroup = groupKinds.filter((item) => item !== id);
    } else {
      setGroupKinds([...groupKinds, id]);
      newGroup = [...groupKinds, id];
    }

    filterSourcesByKind && filterSourcesByKind(newGroup);
  }

  function onLanguageClick(id: string) {
    let newGroup: string[] = [];
    if (groupLanguages.includes(id)) {
      setGroupLanguages(groupLanguages.filter((item) => item !== id));
      newGroup = groupLanguages.filter((item) => item !== id);
    } else {
      setGroupLanguages([...groupLanguages, id]);
      newGroup = [...groupLanguages, id];
    }

    filterSourcesByLanguage && filterSourcesByLanguage(newGroup);
  }

  function onErrorClick(message: string) {
    let newGroup: string[] = [];
    if (groupErrorMessage.includes(message)) {
      setGroupErrorMessage(
        groupErrorMessage.filter((item) => item !== message)
      );
      newGroup = groupErrorMessage.filter((item) => item !== message);
    } else {
      setGroupErrorMessage([...groupErrorMessage, message]);
      newGroup = [...groupErrorMessage, message];
    }

    filterByConditionMessage(newGroup);
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
        onClick: () => onNamespaceClick(item.name),
        id: item.name,
        selected: groupNamespaces.includes(item.name) && item.totalApps > 0,
        disabled:
          (groupNamespaces.length === 1 &&
            groupNamespaces.includes(item.name)) ||
          item.totalApps === 0 ||
          totalNamespacesWithApps === 1,
      }));

    const actionsGroup = [
      {
        label: 'Language',
        subTitle: 'Filter',
        condition: true,
        items: [
          {
            label: 'Javascript',
            onClick: () => onLanguageClick('javascript'),
            id: 'javascript',
            selected: groupLanguages.includes('javascript'),
            disabled:
              (groupLanguages.length === 1 &&
                groupLanguages.includes('javascript')) ||
              (data.length === 1 &&
                data?.[0]?.instrumented_application_details?.languages?.[0]
                  .language === 'javascript'),
          },
          {
            label: 'Python',
            onClick: () => onLanguageClick('python'),
            id: 'python',
            selected: groupLanguages.includes('python'),
            disabled:
              (groupLanguages.length === 1 &&
                groupLanguages.includes('python')) ||
              (data.length === 1 &&
                data?.[0]?.instrumented_application_details?.languages?.[0]
                  .language === 'python'),
          },
          {
            label: 'Java',
            onClick: () => onLanguageClick('java'),
            id: 'java',
            selected: groupLanguages.includes('java'),
            disabled:
              (groupLanguages.length === 1 &&
                groupLanguages.includes('java')) ||
              (data.length === 1 &&
                data?.[0]?.instrumented_application_details?.languages?.[0]
                  .language === 'java'),
          },
          {
            label: 'Go',
            onClick: () => onLanguageClick('go'),
            id: 'go',
            selected: groupLanguages.includes('go'),
            disabled:
              (groupLanguages.length === 1 && groupLanguages.includes('go')) ||
              (data.length === 1 &&
                data?.[0]?.instrumented_application_details?.languages?.[0]
                  .language === 'go'),
          },
          {
            label: '.NET',
            onClick: () => onLanguageClick('dotnet'),
            id: 'dotnet',
            selected: groupLanguages.includes('dotnet'),
            disabled:
              (groupLanguages.length === 1 &&
                groupLanguages.includes('dotnet')) ||
              (data.length === 1 &&
                data?.[0]?.instrumented_application_details?.languages?.[0]
                  .language === 'dotnet'),
          },
        ],
      },
      {
        label: 'Kind',
        subTitle: 'Filter',
        condition: true,
        items: [
          {
            label: 'Deployment',
            onClick: () => onKindClick(K8SSourceTypes.DEPLOYMENT),
            id: K8SSourceTypes.DEPLOYMENT,
            selected:
              groupKinds.includes(K8SSourceTypes.DEPLOYMENT) &&
              data.some(
                (item) => item.kind.toLowerCase() === K8SSourceTypes.DEPLOYMENT
              ),
            disabled:
              groupKinds.length === 1 &&
              groupKinds.includes(K8SSourceTypes.DEPLOYMENT) &&
              data.some(
                (item) => item.kind.toLowerCase() === K8SSourceTypes.DEPLOYMENT
              ),
          },
          {
            label: 'StatefulSet',
            onClick: () => onKindClick(K8SSourceTypes.STATEFUL_SET),
            id: K8SSourceTypes.STATEFUL_SET,
            selected:
              groupKinds.includes(K8SSourceTypes.STATEFUL_SET) &&
              data.some(
                (item) =>
                  item.kind.toLowerCase() === K8SSourceTypes.STATEFUL_SET
              ),
            disabled:
              (groupKinds.length === 1 &&
                groupKinds.includes(K8SSourceTypes.STATEFUL_SET)) ||
              data.every(
                (item) =>
                  item.kind.toLowerCase() !== K8SSourceTypes.STATEFUL_SET
              ),
          },
          {
            label: 'DemonSet',
            onClick: () => onKindClick(K8SSourceTypes.DAEMON_SET),
            id: K8SSourceTypes.DAEMON_SET,
            selected:
              groupKinds.includes(K8SSourceTypes.DAEMON_SET) &&
              data.some(
                (item) => item.kind.toLowerCase() === K8SSourceTypes.DAEMON_SET
              ),
            disabled:
              (groupKinds.length === 1 &&
                groupKinds.includes(K8SSourceTypes.DAEMON_SET)) ||
              data.every(
                (item) => item.kind.toLowerCase() !== K8SSourceTypes.DAEMON_SET
              ),
          },
        ],
      },
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

    if (showSourcesWithIssues) {
      actionsGroup.unshift({
        label: 'Error',
        subTitle: 'Filter by error message',
        condition: true,
        items: groupErrorMessages().map((item) => ({
          label: item,
          onClick: () => onErrorClick(item),
          id: item,
          selected: groupErrorMessage.includes(item),
          disabled:
            groupErrorMessage.length === 1 && groupErrorMessage.includes(item),
        })),
      });
    }

    return actionsGroup;
  }, [namespaces, groupNamespaces, data]);

  return (
    <StyledThead>
      <StyledMainTh>
        <KeyvalCheckbox
          value={selectedCheckbox.length === data.length && data.length > 0}
          onChange={() => onSelectedCheckboxChange(SELECT_ALL_CHECKBOX)}
        />
        <UnFocusSourcesIcon size={18} />
        <KeyvalText size={14} weight={600} color={theme.text.white}>
          {`${data.length} ${OVERVIEW.MENU.SOURCES}`}
        </KeyvalText>

        {groupErrorMessage.length > 0 && (
          <KeyvalSwitch
            toggle={showSourcesWithIssues}
            handleToggleChange={() =>
              setShowSourcesWithIssues(!showSourcesWithIssues)
            }
            label={'Show only sources with issues'}
          />
        )}
        {selectedCheckbox.length > 0 && (
          <KeyvalLink
            onClick={deleteSourcesHandler}
            value={OVERVIEW.DELETE}
            fontSize={12}
          />
        )}
        <ActionGroupContainer>
          <ActionsGroup actionGroups={sourcesGroups} />
        </ActionGroupContainer>
      </StyledMainTh>
    </StyledThead>
  );
}
