import React from 'react';
import { NOTIFICATION_TYPE } from '@/types';
import styled, { useTheme } from 'styled-components';
import { Code, DataTab } from '@/reuseable-components';
import { capitalizeFirstLetter, INSTUMENTATION_STATUS, parseJsonStringToPrettyString, safeJsonParse } from '@/utils';
import { Divider, getProgrammingLanguageIcon, InteractiveTable, MonitorsIcons, NotificationNote, Status, Text, Tooltip, Types } from '@odigos/ui-components';

export enum DataCardFieldTypes {
  DIVIDER = 'divider',
  MONITORS = 'monitors',
  ACTIVE_STATUS = 'active-status',
  SOURCE_CONTAINER = 'source-container',
  CODE = 'code',
  TABLE = 'table',
}

export interface DataCardRow {
  type?: DataCardFieldTypes;
  title?: string;
  tooltip?: string;
  value?: string | Record<string, any>;
  width?: string;
}

interface Props {
  data: DataCardRow[];
}

const ListContainer = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 16px 32px;
  width: 100%;
`;

const ListItem = styled.div<{ $width: string }>`
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: ${({ $width }) => $width};
`;

const ItemTitle = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 16px;
`;

export const DataCardFields: React.FC<Props> = ({ data }) => {
  return (
    <ListContainer>
      {data.map(({ type, title, tooltip, value, width = 'unset' }) => (
        <ListItem key={`data-field-${title || value}`} $width={width}>
          <Tooltip text={tooltip} withIcon>
            {!!title && <ItemTitle>{title}</ItemTitle>}
          </Tooltip>
          {renderValue(type, value)}
        </ListItem>
      ))}
    </ListContainer>
  );
};

const PreWrap = styled(Text)`
  font-size: 12px;
  white-space: pre-wrap;
`;

// We need to maintain this with new components every time we add a new type to "DataCardFieldTypes"
const renderValue = (type: DataCardRow['type'], value: DataCardRow['value']) => {
  const theme = useTheme();

  switch (type) {
    case DataCardFieldTypes.DIVIDER:
      return <Divider length='100%' margin='0' />;

    case DataCardFieldTypes.MONITORS:
      return <MonitorsIcons monitors={value?.split(', ') || []} withLabels color={theme.colors.secondary} />;

    case DataCardFieldTypes.ACTIVE_STATUS:
      return <Status status={value == 'true' ? NOTIFICATION_TYPE.SUCCESS : NOTIFICATION_TYPE.ERROR} title={value == 'true' ? 'Active' : 'Inactive'} size={10} withIcon withBorder />;

    case DataCardFieldTypes.CODE: {
      const params = safeJsonParse(value, { language: '', code: '' });

      return <Code {...params} />;
    }

    case DataCardFieldTypes.TABLE: {
      const params = safeJsonParse(value, { columns: [], rows: [] });

      return <InteractiveTable {...params} />;
    }

    case DataCardFieldTypes.SOURCE_CONTAINER: {
      const { containerName, language, runtimeVersion, otherAgent, hasPresenceOfOtherAgent } = safeJsonParse(value, {
        containerName: '-',
        language: Types.PROGRAMMING_LANGUAGES.UNKNOWN,
        runtimeVersion: '-',
        otherAgent: null,
        hasPresenceOfOtherAgent: false,
      });

      // Determine if running concurrently is possible based on language and other_agent
      const canRunInParallel = !hasPresenceOfOtherAgent && (language === Types.PROGRAMMING_LANGUAGES.PYTHON || language === Types.PROGRAMMING_LANGUAGES.JAVA);

      return (
        <DataTab
          title={containerName}
          subTitle={`${language === Types.PROGRAMMING_LANGUAGES.JAVASCRIPT ? 'Node.js' : capitalizeFirstLetter(language)}` + (!!runtimeVersion ? ` • Runtime Version: ${runtimeVersion}` : '')}
          iconSrc={getProgrammingLanguageIcon(language)}
          isExtended={!!otherAgent}
          renderExtended={() => (
            <NotificationNote
              type={NOTIFICATION_TYPE.INFO}
              message={
                hasPresenceOfOtherAgent
                  ? `By default, we do not operate alongside the ${otherAgent}. Please contact the Odigos team for guidance on enabling this configuration.`
                  : canRunInParallel
                  ? `We are operating alongside the ${otherAgent}, which is not the recommended configuration. We suggest disabling the ${otherAgent} for optimal performance.`
                  : `Concurrent execution with the ${otherAgent} is not supported. Please disable one of the agents to enable proper instrumentation.`
              }
            />
          )}
          renderActions={() => {
            const isActive = ![
              Types.PROGRAMMING_LANGUAGES.IGNORED,
              Types.PROGRAMMING_LANGUAGES.UNKNOWN,
              Types.PROGRAMMING_LANGUAGES.PROCESSING,
              Types.PROGRAMMING_LANGUAGES.NO_CONTAINERS,
              Types.PROGRAMMING_LANGUAGES.NO_RUNNING_PODS,
            ].includes(language);

            return (
              <Status
                status={isActive ? NOTIFICATION_TYPE.SUCCESS : NOTIFICATION_TYPE.ERROR}
                title={isActive ? INSTUMENTATION_STATUS.INSTRUMENTED : INSTUMENTATION_STATUS.UNINSTRUMENTED}
                withIcon
                withBorder
              />
            );
          }}
        />
      );
    }

    default: {
      return <PreWrap>{parseJsonStringToPrettyString(typeof value === 'string' ? value || '-' : '-')}</PreWrap>;
    }
  }
};
