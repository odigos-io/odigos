'use client';
import React, { useEffect, useRef, useState } from 'react';
import { ROUTES } from '@/utils';
import { OverviewDataFlowWrapper } from './styled';
import { useRouter, useSearchParams } from 'next/navigation';
import { KeyvalDataFlow, KeyvalLoader } from '@/design.system';
import { useActions, useDestinations, useSources } from '@/hooks';
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
  useEffect(() => {
    if (!sources || !destinationList || !actions) return;

    const filteredActions = actions.filter((action) => !action.spec.disabled);
    const mapSources = sources
      .sort((a, b) => a.name.localeCompare(b.name))
      .map((source) => {
        const languages =
          source?.instrumented_application_details?.languages || [];
        const conditions =
          source?.instrumented_application_details?.conditions || [];
        return {
          ...source,
          conditions,
          languages:
            languages.length > 0
              ? languages
              : [{ language: 'default', container_name: '' }],
        };
      });

    const { nodes, edges } = buildFlowNodesAndEdges(
      mapSources,
      destinationList,
      filteredActions
    );
    setNodes(nodes);
    setEdges(edges);
  }, [actions, destinationList, sources]);

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
