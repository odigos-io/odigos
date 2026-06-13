'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled, { keyframes } from 'styled-components';
import { useTraceCorrelations, type TraceCorrelationsInputGroup, type TraceCorrelationsOutputSeries, type TraceCorrelationsWorkload } from '@/hooks/metrics/useTraceCorrelations';

const fadeIn = keyframes`
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
`;

const Page = styled.div`
  min-height: 100vh;
  background:
    radial-gradient(circle at top left, rgba(56, 189, 248, 0.12), transparent 28%),
    radial-gradient(circle at top right, rgba(167, 139, 250, 0.12), transparent 24%),
    linear-gradient(180deg, #0b1020 0%, #111827 45%, #0f172a 100%);
  color: #e2e8f0;
  font-family: 'SF Pro Display', 'Segoe UI', system-ui, sans-serif;
`;

const Shell = styled.div`
  max-width: 1400px;
  margin: 0 auto;
  padding: 32px 24px 64px;
`;

const Hero = styled.header`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 28px;
`;

const Eyebrow = styled.div`
  font-size: 12px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: #67e8f9;
  margin-bottom: 8px;
`;

const Title = styled.h1`
  margin: 0;
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 1.05;
  font-weight: 700;
  color: #f8fafc;
`;

const Subtitle = styled.p`
  margin: 10px 0 0;
  max-width: 720px;
  color: #94a3b8;
  font-size: 1rem;
  line-height: 1.6;
`;

const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
`;

const Input = styled.input`
  min-width: 240px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.72);
  color: #f8fafc;
  outline: none;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);

  &:focus {
    border-color: rgba(34, 211, 238, 0.55);
    box-shadow: 0 0 0 3px rgba(34, 211, 238, 0.12);
  }
`;

const Button = styled.button`
  padding: 12px 16px;
  border: none;
  border-radius: 12px;
  background: linear-gradient(135deg, #06b6d4, #6366f1);
  color: white;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s ease, opacity 0.15s ease;

  &:hover {
    transform: translateY(-1px);
  }

  &:disabled {
    opacity: 0.6;
    cursor: wait;
  }
`;

const StatsGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 14px;
  margin-bottom: 24px;
`;

const StatCard = styled.div`
  padding: 18px 20px;
  border-radius: 18px;
  background: rgba(15, 23, 42, 0.72);
  border: 1px solid rgba(148, 163, 184, 0.12);
  backdrop-filter: blur(12px);
  animation: ${fadeIn} 0.35s ease;
`;

const StatLabel = styled.div`
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  color: #94a3b8;
  margin-bottom: 8px;
`;

const StatValue = styled.div`
  font-size: 1.8rem;
  font-weight: 700;
  color: #f8fafc;
`;

const WorkloadGrid = styled.div`
  display: grid;
  gap: 18px;
`;

const WorkloadCard = styled.section`
  border-radius: 22px;
  overflow: hidden;
  border: 1px solid rgba(148, 163, 184, 0.12);
  background: rgba(15, 23, 42, 0.78);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.22);
  animation: ${fadeIn} 0.4s ease;
`;

const WorkloadHeader = styled.button<{ $expanded?: boolean }>`
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  padding: 22px 24px;
  border: none;
  text-align: left;
  cursor: pointer;
  background: linear-gradient(135deg, rgba(14, 165, 233, 0.12), rgba(99, 102, 241, 0.08));
  border-bottom: ${({ $expanded }) => ($expanded ? '1px solid rgba(148, 163, 184, 0.1)' : 'none')};
  transition: background 0.15s ease;

  &:hover {
    background: linear-gradient(135deg, rgba(14, 165, 233, 0.16), rgba(99, 102, 241, 0.12));
  }
`;

const HeaderMain = styled.div`
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
`;

const CollapseIcon = styled.span<{ $expanded: boolean }>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-top: 2px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.55);
  color: #67e8f9;
  font-size: 14px;
  line-height: 1;
  flex-shrink: 0;
  transform: rotate(${({ $expanded }) => ($expanded ? '90deg' : '0deg')});
  transition: transform 0.2s ease;
`;

const HeaderContent = styled.div`
  min-width: 0;
`;

const WorkloadTitle = styled.h2`
  margin: 0;
  font-size: 1.25rem;
  color: #f8fafc;
`;

const WorkloadMeta = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
`;

const Pill = styled.span`
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.8);
  border: 1px solid rgba(148, 163, 184, 0.14);
  color: #cbd5e1;
  font-size: 12px;
`;

const WorkloadStats = styled.div`
  display: flex;
  gap: 18px;
  align-items: center;
`;

const MiniStat = styled.div`
  text-align: right;
`;

const MiniStatLabel = styled.div`
  font-size: 11px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.08em;
`;

const MiniStatValue = styled.div`
  font-size: 1.4rem;
  font-weight: 700;
  color: #5eead4;
`;

const InputGroupList = styled.div`
  display: grid;
  gap: 14px;
  padding: 20px 24px 24px;
`;

const InputGroupCard = styled.div`
  border-radius: 16px;
  border: 1px solid rgba(148, 163, 184, 0.1);
  background: rgba(2, 6, 23, 0.45);
  overflow: hidden;
`;

const InputHeader = styled.button<{ $expanded?: boolean }>`
  display: block;
  width: 100%;
  padding: 14px 16px;
  border: none;
  text-align: left;
  cursor: pointer;
  background: rgba(30, 41, 59, 0.35);
  border-bottom: ${({ $expanded }) => ($expanded ? '1px solid rgba(148, 163, 184, 0.08)' : 'none')};
  transition: background 0.15s ease;

  &:hover {
    background: rgba(30, 41, 59, 0.5);
  }
`;

const InputHeaderRow = styled.div`
  display: flex;
  align-items: flex-start;
  gap: 10px;
`;

const SectionLabel = styled.div<{ $variant?: 'inbound' | 'outbound' }>`
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: ${({ $variant }) => ($variant === 'outbound' ? '#c4b5fd' : '#67e8f9')};
  margin-bottom: 8px;
`;

const InboundPanel = styled.div`
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(34, 211, 238, 0.22);
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.1), rgba(14, 165, 233, 0.04));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
`;

const OutboundPanel = styled.div`
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(167, 139, 250, 0.22);
  background: linear-gradient(135deg, rgba(167, 139, 250, 0.1), rgba(99, 102, 241, 0.04));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
`;

const FlowBody = styled.div`
  padding: 16px;
  display: grid;
  gap: 0;
`;

const FlowDivider = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 0 10px 8px;
  color: #94a3b8;
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
`;

const FlowDividerLine = styled.span`
  flex: 1;
  height: 1px;
  background: linear-gradient(90deg, rgba(34, 211, 238, 0.45), rgba(167, 139, 250, 0.45));
`;

const FlowArrowBadge = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.85);
  border: 1px solid rgba(148, 163, 184, 0.14);
  color: #e2e8f0;
  white-space: nowrap;
`;

type RowClassification = 'none' | 'baseline' | 'suspicious';

const OutputFlowList = styled.div`
  position: relative;
  margin-left: 18px;
  padding-left: 28px;
  border-left: 2px solid rgba(34, 211, 238, 0.22);
  display: grid;
  gap: 14px;
`;

const OutputFlowRow = styled.div<{ $classification?: RowClassification }>`
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto minmax(148px, auto);
  gap: 14px;
  align-items: start;
  padding: 12px;
  border-radius: 14px;
  border: 1px solid transparent;
  transition: background 0.2s ease, border-color 0.2s ease;
  background: ${({ $classification }) => {
    if ($classification === 'baseline') return 'rgba(16, 185, 129, 0.14)';
    if ($classification === 'suspicious') return 'rgba(239, 68, 68, 0.14)';
    return 'transparent';
  }};
  border-color: ${({ $classification }) => {
    if ($classification === 'baseline') return 'rgba(52, 211, 153, 0.28)';
    if ($classification === 'suspicious') return 'rgba(248, 113, 113, 0.32)';
    return 'transparent';
  }};

  &::before {
    content: '';
    position: absolute;
    left: -30px;
    top: 50%;
    width: 24px;
    height: 2px;
    transform: translateY(-50%);
    background: linear-gradient(90deg, rgba(34, 211, 238, 0.35), rgba(167, 139, 250, 0.55));
  }

  &::after {
    content: '▶';
    position: absolute;
    left: -8px;
    top: 50%;
    transform: translateY(-50%);
    color: #a78bfa;
    font-size: 11px;
  }

  @media (max-width: 820px) {
    grid-template-columns: 1fr;
  }
`;

const RowActions = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-self: start;
`;

const RowActionButton = styled.button<{ $active?: boolean; $variant: 'baseline' | 'suspicious' }>`
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid
    ${({ $active, $variant }) =>
      $active
        ? $variant === 'baseline'
          ? 'rgba(52, 211, 153, 0.55)'
          : 'rgba(248, 113, 113, 0.55)'
        : 'rgba(148, 163, 184, 0.18)'};
  background: ${({ $active, $variant }) =>
    $active
      ? $variant === 'baseline'
        ? 'rgba(16, 185, 129, 0.22)'
        : 'rgba(239, 68, 68, 0.22)'
      : 'rgba(15, 23, 42, 0.72)'};
  color: ${({ $active, $variant }) =>
    $active ? ($variant === 'baseline' ? '#6ee7b7' : '#fca5a5') : '#cbd5e1'};
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s ease, border-color 0.15s ease, color 0.15s ease;

  &:hover {
    border-color: ${({ $variant }) =>
      $variant === 'baseline' ? 'rgba(52, 211, 153, 0.45)' : 'rgba(248, 113, 113, 0.45)'};
  }
`;

const HeaderSummary = styled.div`
  color: #cbd5e1;
  font-size: 13px;
  line-height: 1.5;
`;

const AttributeList = styled.ul`
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
`;

const AttributeItem = styled.li`
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
`;

const AttributeKey = styled.span`
  color: #94a3b8;

  &::after {
    content: ': ';
  }
`;

const AttributeValue = styled.span`
  color: #f8fafc;
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
`;

const HttpSummary = styled.li`
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.45;
  color: #e2e8f0;
  word-break: break-word;
`;

const HttpMethod = styled.span<{ $method: string }>`
  font-weight: 700;
  color: ${({ $method }) => getHttpMethodColor($method)};
`;

const HttpStatus = styled.span`
  color: #cbd5e1;
`;

const HttpTarget = styled.span`
  color: #94a3b8;
`;

const HTTP_SUMMARY_KEYS = new Set([
  'http.method',
  'http.request.method',
  'http.route',
  'http.target',
  'url.path',
  'http.url',
  'http.status_code',
  'http.response.status_code',
  'server.address',
  'net.peer.name',
  'http.host',
  'url.domain',
  'peer.service',
]);

function getHttpMethodColor(method: string) {
  switch (method.toUpperCase()) {
    case 'GET':
      return '#6ee7b7';
    case 'POST':
      return '#60a5fa';
    case 'PUT':
      return '#fbbf24';
    case 'PATCH':
      return '#c084fc';
    case 'DELETE':
      return '#f87171';
    case 'HEAD':
    case 'OPTIONS':
      return '#94a3b8';
    default:
      return '#e2e8f0';
  }
}

function attributesToMap(attributes: { key: string; value: string }[]) {
  return Object.fromEntries(attributes.map((attr) => [attr.key, attr.value]));
}

function pathFromUrl(url: string) {
  try {
    return new URL(url).pathname || undefined;
  } catch {
    return undefined;
  }
}

function isHttpAttributes(attributes: { key: string; value: string }[]) {
  return attributes.some(
    (attr) =>
      attr.key.startsWith('http.') ||
      attr.key.startsWith('url.') ||
      attr.key === 'server.address' ||
      attr.key === 'net.peer.name' ||
      attr.key === 'peer.service',
  );
}

function buildHttpSummary(attributes: { key: string; value: string }[]) {
  const map = attributesToMap(attributes);
  const method = map['http.method'] || map['http.request.method'];
  const path =
    map['http.route'] ||
    map['http.target'] ||
    map['url.path'] ||
    (map['http.url'] ? pathFromUrl(map['http.url']) : undefined);
  const status = map['http.status_code'] || map['http.response.status_code'];
  const target =
    map['server.address'] || map['net.peer.name'] || map['http.host'] || map['url.domain'] || map['peer.service'];

  if (!method && !path && !status && !target) {
    return null;
  }

  return { method, path, status, target };
}

function partitionHttpAttributes(attributes: { key: string; value: string }[]) {
  if (!isHttpAttributes(attributes)) {
    return { summary: null, remaining: attributes };
  }

  const summary = buildHttpSummary(attributes);
  if (!summary) {
    return { summary: null, remaining: attributes };
  }

  const remaining = attributes.filter((attr) => !HTTP_SUMMARY_KEYS.has(attr.key));
  return { summary, remaining };
}

function HttpSummaryLine({ summary }: { summary: NonNullable<ReturnType<typeof buildHttpSummary>> }) {
  return (
    <HttpSummary>
      {summary.method ? (
        <>
          <HttpMethod $method={summary.method}>{summary.method.toUpperCase()}</HttpMethod>{' '}
        </>
      ) : null}
      {summary.path ? <span>{summary.path}</span> : null}
      {summary.status ? (
        <>
          {summary.path || summary.method ? ' ' : null}
          <HttpStatus>{summary.status}</HttpStatus>
        </>
      ) : null}
      {summary.target ? (
        <>
          {' '}
          <HttpTarget>
            to {summary.target}
          </HttpTarget>
        </>
      ) : null}
    </HttpSummary>
  );
}

function formatHttpSummaryText(summary: NonNullable<ReturnType<typeof buildHttpSummary>>) {
  const parts: string[] = [];
  if (summary.method) parts.push(summary.method.toUpperCase());
  if (summary.path) parts.push(summary.path);
  if (summary.status) parts.push(summary.status);
  if (summary.target) parts.push(`to ${summary.target}`);
  return parts.join(' ');
}

const CountBadge = styled.div`
  justify-self: start;
  align-self: start;
  padding: 8px 12px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(52, 211, 153, 0.25);
  color: #6ee7b7;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
`;

const Timestamp = styled.div`
  color: #cbd5e1;
  font-size: 13px;
  align-self: start;
`;

const RelativeTime = styled.span`
  display: block;
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
  font-weight: 500;
`;

const EmptyState = styled.div`
  padding: 48px 24px;
  text-align: center;
  border-radius: 22px;
  border: 1px dashed rgba(148, 163, 184, 0.18);
  color: #94a3b8;
  background: rgba(15, 23, 42, 0.55);
`;

const ErrorState = styled(EmptyState)`
  color: #fca5a5;
  border-color: rgba(248, 113, 113, 0.25);
`;

const LoadingState = styled(EmptyState)`
  color: #67e8f9;
`;

const ToggleButton = styled.button<{ $active?: boolean }>`
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid ${({ $active }) => ($active ? 'rgba(34, 211, 238, 0.55)' : 'rgba(148, 163, 184, 0.18)')};
  background: ${({ $active }) => ($active ? 'rgba(34, 211, 238, 0.12)' : 'rgba(15, 23, 42, 0.72)')};
  color: ${({ $active }) => ($active ? '#67e8f9' : '#cbd5e1')};
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;

  &:hover {
    border-color: rgba(34, 211, 238, 0.45);
  }
`;

const HiddenAttributesNote = styled.span`
  color: #64748b;
  font-size: 12px;
  font-style: italic;
`;

const HIDDEN_ATTRIBUTE_KEYS = new Set(['otel.scope.name', 'otel.scope.version', 'span.kind']);

function filterVisibleAttributes(attributes: { key: string; value: string }[], showFullData: boolean) {
  if (showFullData) return attributes;
  return attributes.filter((attr) => !HIDDEN_ATTRIBUTE_KEYS.has(attr.key));
}

function formatAttributes(attributes: { key: string; value: string }[]) {
  if (!attributes.length) return 'No attributes';
  return attributes.map((attr) => `${attr.key}=${attr.value}`).join(', ');
}

function formatTimestamp(value: string) {
  if (!value) return 'Unknown';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

function formatTimeAgo(value: string, now = Date.now()) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';

  const diffMs = Math.max(0, now - date.getTime());
  const totalMinutes = Math.floor(diffMs / 60_000);
  const days = Math.floor(totalMinutes / (60 * 24));
  const hours = Math.floor((totalMinutes % (60 * 24)) / 60);
  const minutes = totalMinutes % 60;

  if (days > 0) {
    if (hours > 0 && minutes > 0) return `${days}d${hours}h${minutes}m`;
    if (hours > 0) return `${days}d${hours}h`;
    if (minutes > 0) return `${days}d${minutes}m`;
    return `${days}d`;
  }
  if (hours > 0) {
    return minutes > 0 ? `${hours}h${minutes}m` : `${hours}h`;
  }
  if (minutes > 0) return `${minutes}m`;
  return 'just now';
}

function FirstSeenTimestamp({ value }: { value: string }) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(interval);
  }, []);

  const timeAgo = useMemo(() => formatTimeAgo(value, now), [value, now]);

  return (
    <Timestamp>
      First seen
      <br />
      <strong>{formatTimestamp(value)}</strong>
      {timeAgo ? <RelativeTime>{timeAgo} ago</RelativeTime> : null}
    </Timestamp>
  );
}

function workloadStats(workload: TraceCorrelationsWorkload) {
  const inputGroups = workload.inputs.length;
  const outputSeries = workload.inputs.reduce((sum, input) => sum + input.outputs.length, 0);
  const connections = workload.inputs.reduce(
    (sum, input) => sum + input.outputs.reduce((inner, output) => inner + output.connectionCount, 0),
    0,
  );
  return { inputGroups, outputSeries, connections };
}

function AttributeChips({ attributes, showFullData }: { attributes: { key: string; value: string }[]; showFullData: boolean }) {
  const visibleAttributes = filterVisibleAttributes(attributes, showFullData);
  const hiddenCount = attributes.length - visibleAttributes.length;
  const { summary, remaining } = partitionHttpAttributes(visibleAttributes);

  if (!summary && !remaining.length) {
    return (
      <HiddenAttributesNote>
        {hiddenCount > 0 ? `${hiddenCount} metadata attribute${hiddenCount === 1 ? '' : 's'} hidden` : 'No attributes'}
      </HiddenAttributesNote>
    );
  }

  return (
    <AttributeList>
      {summary ? <HttpSummaryLine summary={summary} /> : null}
      {remaining.map((attr) => (
        <AttributeItem key={`${attr.key}:${attr.value}`}>
          <AttributeKey>{attr.key}</AttributeKey>
          <AttributeValue>{attr.value}</AttributeValue>
        </AttributeItem>
      ))}
    </AttributeList>
  );
}

function summarizeAttributes(attributes: { key: string; value: string }[], showFullData: boolean) {
  const visible = filterVisibleAttributes(attributes, showFullData);
  if (!visible.length) return 'Metadata hidden';

  const { summary } = partitionHttpAttributes(visible);
  if (summary) return formatHttpSummaryText(summary);

  if (visible.length === 1) return `${visible[0].key}=${visible[0].value}`;
  return `${visible[0].key}=${visible[0].value} + ${visible.length - 1} more`;
}

function connectionRowKey(inputAttributes: { key: string; value: string }[], output: TraceCorrelationsOutputSeries) {
  return `${formatAttributes(inputAttributes)}::${formatAttributes(output.attributes)}::${output.firstDetectedAt}`;
}

function ConnectionRow({
  output,
  showFullData,
}: {
  output: TraceCorrelationsOutputSeries;
  showFullData: boolean;
}) {
  const [classification, setClassification] = useState<RowClassification>('none');

  const setBaseline = () => {
    setClassification((current) => (current === 'baseline' ? 'none' : 'baseline'));
  };

  const setSuspicious = () => {
    setClassification((current) => (current === 'suspicious' ? 'none' : 'suspicious'));
  };

  return (
    <OutputFlowRow $classification={classification === 'none' ? undefined : classification}>
      <OutboundPanel>
        <SectionLabel $variant='outbound'>Outbound</SectionLabel>
        <AttributeChips attributes={output.attributes} showFullData={showFullData} />
      </OutboundPanel>
      <CountBadge>{output.connectionCount.toLocaleString()} connections</CountBadge>
      <FirstSeenTimestamp value={output.firstDetectedAt} />
      <RowActions>
        <RowActionButton type='button' $variant='baseline' $active={classification === 'baseline'} onClick={setBaseline}>
          Add to baseline
        </RowActionButton>
        <RowActionButton type='button' $variant='suspicious' $active={classification === 'suspicious'} onClick={setSuspicious}>
          Suspicious
        </RowActionButton>
      </RowActions>
    </OutputFlowRow>
  );
}

function InputGroupView({ group, showFullData }: { group: TraceCorrelationsInputGroup; showFullData: boolean }) {
  const [expanded, setExpanded] = useState(true);

  return (
    <InputGroupCard>
      <InputHeader
        type='button'
        $expanded={expanded}
        aria-expanded={expanded}
        onClick={() => setExpanded((current) => !current)}
      >
        <InputHeaderRow>
          <CollapseIcon $expanded={expanded}>›</CollapseIcon>
          <div>
            <SectionLabel $variant='inbound'>Inbound pattern</SectionLabel>
            {expanded ? (
              <HeaderSummary>{group.outputs.length} outbound pattern{group.outputs.length === 1 ? '' : 's'}</HeaderSummary>
            ) : (
              <HeaderSummary>{summarizeAttributes(group.attributes, showFullData)}</HeaderSummary>
            )}
          </div>
        </InputHeaderRow>
      </InputHeader>

      {expanded ? (
        <FlowBody>
          <InboundPanel>
            <SectionLabel $variant='inbound'>Inbound</SectionLabel>
            <AttributeChips attributes={group.attributes} showFullData={showFullData} />
          </InboundPanel>

          <FlowDivider>
            <FlowDividerLine />
            <FlowArrowBadge>→ triggers</FlowArrowBadge>
            <FlowDividerLine />
          </FlowDivider>

          <OutputFlowList>
            {group.outputs.map((output) => (
              <ConnectionRow
                key={connectionRowKey(group.attributes, output)}
                output={output}
                showFullData={showFullData}
              />
            ))}
          </OutputFlowList>
        </FlowBody>
      ) : null}
    </InputGroupCard>
  );
}

function WorkloadCardView({ workload, showFullData }: { workload: TraceCorrelationsWorkload; showFullData: boolean }) {
  const [expanded, setExpanded] = useState(true);
  const stats = workloadStats(workload);

  return (
    <WorkloadCard>
      <WorkloadHeader
        type='button'
        $expanded={expanded}
        aria-expanded={expanded}
        onClick={() => setExpanded((current) => !current)}
      >
        <HeaderMain>
          <CollapseIcon $expanded={expanded}>›</CollapseIcon>
          <HeaderContent>
            <WorkloadTitle>{workload.name}</WorkloadTitle>
            <WorkloadMeta>
              <Pill>{workload.namespace}</Pill>
              <Pill>{workload.kind}</Pill>
              <Pill>{workload.containerName}</Pill>
            </WorkloadMeta>
          </HeaderContent>
        </HeaderMain>

        <WorkloadStats>
          <MiniStat>
            <MiniStatLabel>Inputs</MiniStatLabel>
            <MiniStatValue>{stats.inputGroups}</MiniStatValue>
          </MiniStat>
          <MiniStat>
            <MiniStatLabel>Outputs</MiniStatLabel>
            <MiniStatValue>{stats.outputSeries}</MiniStatValue>
          </MiniStat>
          <MiniStat>
            <MiniStatLabel>Connections</MiniStatLabel>
            <MiniStatValue>{stats.connections.toLocaleString()}</MiniStatValue>
          </MiniStat>
        </WorkloadStats>
      </WorkloadHeader>

      {expanded ? (
        <InputGroupList>
          {workload.inputs.map((input) => (
            <InputGroupView key={formatAttributes(input.attributes)} group={input} showFullData={showFullData} />
          ))}
        </InputGroupList>
      ) : null}
    </WorkloadCard>
  );
}

export default function TraceCorrelationsPage() {
  const [namespaceFilter, setNamespaceFilter] = useState('');
  const [showFullData, setShowFullData] = useState(false);
  const filter = namespaceFilter.trim() ? { namespace: namespaceFilter.trim() } : undefined;
  const { workloads, loading, error, refetch } = useTraceCorrelations(filter);

  const totals = useMemo(() => {
    const connections = workloads.reduce((sum, workload) => sum + workloadStats(workload).connections, 0);
    const outputSeries = workloads.reduce((sum, workload) => sum + workloadStats(workload).outputSeries, 0);
    return {
      workloads: workloads.length,
      connections,
      outputSeries,
    };
  }, [workloads]);

  return (
    <Page>
      <Shell>
        <Hero>
          <div>
            <Eyebrow>Trace Correlations</Eyebrow>
            <Title>Service I/O Connections</Title>
            <Subtitle>
              Live view of inbound-to-outbound span correlations emitted by the serviceio connector. Each card groups a
              workload, its inbound attribute patterns, and the outbound patterns they connect to.
            </Subtitle>
          </div>

          <Toolbar>
            <Input
              placeholder='Filter by namespace'
              value={namespaceFilter}
              onChange={(event) => setNamespaceFilter(event.target.value)}
            />
            <ToggleButton $active={showFullData} onClick={() => setShowFullData((current) => !current)}>
              {showFullData ? 'Hide metadata' : 'View full data'}
            </ToggleButton>
            <Button onClick={() => refetch()} disabled={loading}>
              {loading ? 'Refreshing…' : 'Refresh'}
            </Button>
          </Toolbar>
        </Hero>

        <StatsGrid>
          <StatCard>
            <StatLabel>Workloads</StatLabel>
            <StatValue>{totals.workloads}</StatValue>
          </StatCard>
          <StatCard>
            <StatLabel>Output series</StatLabel>
            <StatValue>{totals.outputSeries}</StatValue>
          </StatCard>
          <StatCard>
            <StatLabel>Total connections</StatLabel>
            <StatValue>{totals.connections.toLocaleString()}</StatValue>
          </StatCard>
        </StatsGrid>

        {error ? (
          <ErrorState>
            Failed to load trace correlations.
            <br />
            {error.message}
          </ErrorState>
        ) : null}

        {loading && !workloads.length ? <LoadingState>Loading trace correlations…</LoadingState> : null}

        {!loading && !error && !workloads.length ? (
          <EmptyState>No trace correlation data found yet. Make sure trace correlations are enabled and traffic is flowing.</EmptyState>
        ) : null}

        <WorkloadGrid>
          {workloads.map((workload) => (
            <WorkloadCardView
              key={`${workload.namespace}/${workload.kind}/${workload.name}/${workload.containerName}`}
              workload={workload}
              showFullData={showFullData}
            />
          ))}
        </WorkloadGrid>
      </Shell>
    </Page>
  );
}
