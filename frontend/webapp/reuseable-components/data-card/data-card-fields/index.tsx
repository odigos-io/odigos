import React from 'react';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { NOTIFICATION_TYPE } from '@/types';
import { capitalizeFirstLetter, getProgrammingLanguageIcon, parseJsonStringToPrettyString, safeJsonParse, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';
import { ActiveStatus, Code, DataTab, Divider, InstrumentStatus, InteractiveTable, MonitorsIcons, NotificationNote, Text, Tooltip } from '@/reuseable-components';

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

const renderValue = (type: DataCardRow['type'], value: DataCardRow['value']) => {
  // We need to maintain this with new components every time we add a new type to "DataCardFieldTypes"

  switch (type) {
    case DataCardFieldTypes.DIVIDER:
      return <Divider length='100%' margin='0' />;

    case DataCardFieldTypes.MONITORS:
      return <MonitorsIcons monitors={value?.split(', ') || []} withLabels color={theme.colors.secondary} />;

    case DataCardFieldTypes.ACTIVE_STATUS:
      return <ActiveStatus isActive={value == 'true'} size={10} withIcon withBorder />;

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
        language: WORKLOAD_PROGRAMMING_LANGUAGES.UNKNOWN,
        runtimeVersion: '-',
        otherAgent: null,
        hasPresenceOfOtherAgent: false,
      });

      // Determine if running concurrently is possible based on language and other_agent
      const canRunInParallel = !hasPresenceOfOtherAgent && (language === WORKLOAD_PROGRAMMING_LANGUAGES.PYTHON || language === WORKLOAD_PROGRAMMING_LANGUAGES.JAVA);

      return (
        <DataTab
          title={containerName}
          subTitle={`${language === WORKLOAD_PROGRAMMING_LANGUAGES.JAVASCRIPT ? 'Node.js' : capitalizeFirstLetter(language)}` + (!!runtimeVersion ? ` • Runtime Version: ${runtimeVersion}` : '')}
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
          renderActions={() => <InstrumentStatus language={language} />}
        />
      );
    }

    default: {
      return <PreWrap>{parseJsonStringToPrettyString(typeof value === 'string' ? value || '-' : '-')}</PreWrap>;
    }
  }
};
