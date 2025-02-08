'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import Theme from '@odigos/ui-theme';
import { MainContent } from '@/styles';
import { FORM_ALERTS, SLACK_LINK } from '@/utils';
import { type SourceInstrumentInput } from '@/types';
import { usePaginatedStore, useStatusStore } from '@/store';
import { NOTIFICATION_TYPE, PLATFORM_TYPE } from '@odigos/ui-utils';
import { OdigosLogoText, SlackLogo, TerminalIcon } from '@odigos/ui-icons';
import { Header, IconButton, PlatformSelect, Status, Tooltip } from '@odigos/ui-components';
import { DataFlow, DataFlowActionsMenu, DRAWER_OTHER_TYPES, MultiSourceControl, NotificationManager, Source, useDrawerStore } from '@odigos/ui-containers';
import { useActionCRUD, useConfig, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const theme = Theme.useTheme();

  const { setDrawerType } = useDrawerStore();
  const { sourcesFetching } = usePaginatedStore();
  const { status, title, message } = useStatusStore();

  const { metrics } = useMetrics();
  const { data: config } = useConfig();
  const { allNamespaces } = useNamespace();
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { sources, filteredSources, loading: srcLoad, persistSources } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  return (
    <>
      <Header
        left={[
          <OdigosLogoText size={80} />,
          <PlatformSelect type={PLATFORM_TYPE.K8S} />,
          <Status status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
          config?.readonly && (
            <Tooltip text={FORM_ALERTS.READONLY_WARNING}>
              <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
            </Tooltip>
          ),
        ]}
        right={[
          <Theme.ToggleDarkMode />,
          <NotificationManager />,
          <IconButton onClick={() => setDrawerType(DRAWER_OTHER_TYPES.ODIGOS_CLI)} tooltip='Odigos CLI' withPing pingColor={theme.colors.majestic_blue}>
            <TerminalIcon size={18} />
          </IconButton>,
          <IconButton onClick={() => window.open(SLACK_LINK, '_blank', 'noopener noreferrer')} tooltip='Join our Slack community'>
            <SlackLogo />
          </IconButton>,
        ]}
      />

      <MainContent>
        <DataFlowActionsMenu namespaces={allNamespaces} sources={filteredSources} destinations={filteredDestinations} actions={filteredActions} instrumentationRules={filteredInstrumentationRules} />
        <DataFlow
          heightToRemove='176px'
          sources={filteredSources}
          sourcesLoading={srcLoad || sourcesFetching}
          sourcesTotalCount={sources.length}
          destinations={filteredDestinations}
          destinationsLoading={destLoad}
          destinationsTotalCount={destinations.length}
          actions={filteredActions}
          actionsLoading={actLoad}
          actionsTotalCount={actions.length}
          instrumentationRules={filteredInstrumentationRules}
          instrumentationRulesLoading={ruleLoad}
          instrumentationRulesTotalCount={instrumentationRules.length}
          metrics={metrics}
        />
        <MultiSourceControl
          totalSourceCount={sources.length}
          uninstrumentSources={(payload) => {
            const inp: SourceInstrumentInput = {};

            Object.entries(payload).forEach(([namespace, sources]: [string, Source[]]) => {
              inp[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
            });

            persistSources(inp, {});
          }}
        />
      </MainContent>

      <AllModals />
      <AllDrawers />
    </>
  );
}
