'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import Theme from '@odigos/ui-theme';
import { SLACK_LINK } from '@/utils';
import { MainContent } from '@/styles';
import { type SourceInstrumentInput } from '@/types';
import { usePaginatedStore, useStatusStore } from '@/store';
import { OdigosLogoText, SlackLogo } from '@odigos/ui-icons';
import { FORM_ALERTS, NOTIFICATION_TYPE, PLATFORM_TYPE } from '@odigos/ui-utils';
import { Header, IconButton, PlatformSelect, Status, Tooltip } from '@odigos/ui-components';
import { useActionCRUD, useConfig, useDescribeOdigos, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD, useSSE, useTokenCRUD, useTokenTracker } from '@/hooks';
import {
  ActionDrawer,
  ActionModal,
  CliDrawer,
  DataFlow,
  DataFlowActionsMenu,
  InstrumentationRuleDrawer,
  InstrumentationRuleModal,
  MultiSourceControl,
  NotificationManager,
  Source,
} from '@odigos/ui-containers';

const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const { sourcesFetching } = usePaginatedStore();
  const { status, title, message } = useStatusStore();

  const { metrics } = useMetrics();
  const { data: config } = useConfig();
  const { allNamespaces } = useNamespace();
  const { tokens, updateToken } = useTokenCRUD();
  const { data: describeOdigos } = useDescribeOdigos();
  const { sources, filteredSources, loading: srcLoad, persistSources } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { actions, filteredActions, loading: actLoad, createAction, updateAction, deleteAction } = useActionCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad, createInstrumentationRule, updateInstrumentationRule, deleteInstrumentationRule } = useInstrumentationRuleCRUD();

  return (
    <>
      <Header
        left={[
          <OdigosLogoText key='logo' size={80} />,
          <PlatformSelect key='platform' type={PLATFORM_TYPE.K8S} />,
          <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
          config?.readonly && (
            <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
              <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
            </Tooltip>
          ),
        ]}
        right={[
          <Theme.ToggleDarkMode key='toggle-theme' />,
          <NotificationManager key='notifs' />,
          <CliDrawer key='cli' tokens={tokens} saveToken={updateToken} describe={describeOdigos} />,
          <IconButton key='slack' onClick={() => window.open(SLACK_LINK, '_blank', 'noopener noreferrer')} tooltip='Join our Slack community'>
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

      <InstrumentationRuleModal isEnterprise={false} createInstrumentationRule={createInstrumentationRule} />
      <ActionModal createAction={createAction} />
      <AllModals />

      <InstrumentationRuleDrawer instrumentationRules={instrumentationRules} updateInstrumentationRule={updateInstrumentationRule} deleteInstrumentationRule={deleteInstrumentationRule} />
      <ActionDrawer actions={actions} updateAction={updateAction} deleteAction={deleteAction} />
      <AllDrawers />
    </>
  );
}
