'use client';
import React, { useEffect, useRef, useState } from 'react';
import { OverviewDataFlowWrapper } from './styled';
import { ROUTES, getMainContainerLanguage } from '@/utils';
import { useRouter, useSearchParams } from 'next/navigation';
import { KeyvalDataFlow, KeyvalLoader } from '@/design.system';
import {
  useActions,
  useDestinations,
  useOverviewMetrics,
  useSources,
} from '@/hooks';
import { buildFlowNodesAndEdges } from '@keyval-dev/design-system';

interface FlowNode {
  id: string;
  type: string;
  position: {
    x: number;
    y: number;
  };
  data: any;
}
interface FlowEdge {
  id: string;
  source: string;
  target: string;
  animated: boolean;
  label?: string;
  style?: Record<string, any>;
  data?: any;
}

const POLL_DATA = 'poll';
const POLL_INTERVAL = 2000; // Interval in milliseconds between polls
const MAX_ATTEMPTS = 5; // Maximum number of polling attempts

export function DataFlowContainer() {
  const [nodes, setNodes] = useState<FlowNode[]>();
  const [edges, setEdges] = useState<FlowEdge[]>();
  const [pollingAttempts, setPollingAttempts] = useState(0);

  const router = useRouter();
  const { actions } = useActions();
  const { sources, refetchSources } = useSources();
  const { destinationList, refetchDestinations } = useDestinations();

  const useSearch = useSearchParams();
  const intervalId = useRef<NodeJS.Timer>();

  const { metrics } = useOverviewMetrics();

  useEffect(() => {
    if (!sources || !destinationList || !actions) return;

    const filteredActions = actions.filter((action) => !action.spec.disabled);
    const mapSources = sources
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((source) => {
        let languages =
          source?.instrumented_application_details?.languages || [];

        languages.map((language) => {
          if (language.language === 'ignored') {
            language.language = getMainContainerLanguage(source);
          }
        });

        const conditions =
          source?.instrumented_application_details?.conditions || [];

        const currentSourceMetrics = metrics?.sources?.find(
          (metric) =>
            metric.name === source.name &&
            metric.namespace === source.namespace &&
            metric.kind === source.kind
        );

        if (!currentSourceMetrics) {
          return {
            ...source,
            conditions,
            languages:
              languages.length > 0
                ? languages
                : [{ language: 'default', container_name: '' }],
          };
        }

        const data_transfer = formatBytes(currentSourceMetrics?.throughput);

        return {
          ...source,
          conditions,
          metrics: {
            data_transfer,
          },
          languages:
            languages.length > 0
              ? languages
              : [{ language: 'default', container_name: '' }],
        };
      });

    const mapDestinations = destinationList.map((destination) => {
      const currentDestinationMetrics = metrics?.destinations?.find(
        (metric) => metric.id === destination.id
      );

      if (!currentDestinationMetrics) {
        return destination;
      }

      const data_transfer = formatBytes(
        currentDestinationMetrics?.throughput ||
          currentDestinationMetrics?.throughput === 0
          ? currentDestinationMetrics?.throughput
          : 0
      );

      return {
        ...destination,
        metrics: {
          data_transfer,
        },
      };
    });

    const { nodes, edges } = buildFlowNodesAndEdges(
      mapSources,
      mapDestinations,
      filteredActions
    );
    setNodes(nodes);
    setEdges(edges);
  }, [actions, destinationList, sources, metrics]);

  useEffect(() => {
    const pullData = useSearch.get(POLL_DATA);
    if (pullData) {
      intervalId.current = setInterval(() => {
        Promise.all([refetchSources(), refetchDestinations()])
          .then(() => {})
          .catch(console.error);

        setPollingAttempts((prev) => prev + 1);
      }, POLL_INTERVAL);

      return () => clearInterval(intervalId.current);
    }
  }, [refetchDestinations, refetchSources, pollingAttempts, useSearch]);

  useEffect(() => {
    if (pollingAttempts >= MAX_ATTEMPTS) {
      clearInterval(intervalId.current);
      return;
    }
  }, [pollingAttempts]);

  function onNodeClick(node: FlowNode, object: any) {
    if (object?.type === 'destination') {
      router.push(`${ROUTES.UPDATE_DESTINATION}${object.data.id}`);
    }
    if (object?.type === 'action') {
      router.push(`${ROUTES.EDIT_ACTION}?id=${object.data.id}`);
    }
    if (object?.data?.kind) {
      router.push(
        `${ROUTES.MANAGE_SOURCE}?name=${object.data.name}&namespace=${object.data.namespace}&kind=${object.data.kind}`
      );
    }
  }

  if (!nodes || !edges) {
    return <KeyvalLoader />;
  }

  return (
    <OverviewDataFlowWrapper>
      <KeyvalDataFlow nodes={nodes} edges={edges} onNodeClick={onNodeClick} />
    </OverviewDataFlowWrapper>
  );
}

function formatBytes(bytes: number): string {
  const sizes = ['Bytes', 'KB/s', 'MB/s', 'GB/s', 'TB/s'];
  if (bytes === 0) return '0 KB/s';
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(2)} ${sizes[i]}`;
}
