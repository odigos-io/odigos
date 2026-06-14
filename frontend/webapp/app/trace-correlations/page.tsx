'use client';

import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import styled, { keyframes } from 'styled-components';
import { useTraceCorrelations, TRACE_CORRELATIONS_TIME_PRESETS, formatTraceCorrelationsTimeRangeLabel, resolveTraceCorrelationsTimeRange, toDatetimeLocalValue, type TraceCorrelationsInputGroup, type TraceCorrelationsOutputSeries, type TraceCorrelationsTimePreset, type TraceCorrelationsWorkload } from '@/hooks/metrics/useTraceCorrelations';

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

const TimeRangeBar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 20px;
  padding: 14px 16px;
  border-radius: 16px;
  background: rgba(15, 23, 42, 0.72);
  border: 1px solid rgba(148, 163, 184, 0.12);
`;

const TimeRangeLabel = styled.div`
  font-size: 12px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #94a3b8;
  margin-right: 4px;
`;

const TimeRangePresets = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
`;

const TimeRangeHint = styled.div`
  font-size: 12px;
  color: #64748b;
`;

const DataCoverageBanner = styled.div`
  margin-bottom: 20px;
  padding: 12px 16px;
  border-radius: 14px;
  border: 1px solid rgba(34, 211, 238, 0.22);
  background: rgba(14, 165, 233, 0.08);
  color: #cbd5e1;
  font-size: 14px;
  line-height: 1.5;
`;

const DataCoverageLabel = styled.strong`
  color: #f8fafc;
  font-weight: 600;
`;

const TimeRangeCustomFields = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: 10px;
  width: 100%;
`;

const TimeRangeField = styled.label`
  display: grid;
  gap: 6px;
  font-size: 11px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.08em;
`;

const DateTimeInput = styled.input`
  min-width: 210px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(15, 23, 42, 0.72);
  color: #f8fafc;
  outline: none;
  color-scheme: dark;
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);

  &:focus {
    border-color: rgba(34, 211, 238, 0.55);
    box-shadow: 0 0 0 3px rgba(34, 211, 238, 0.12);
  }
`;

const TimeRangeError = styled.div`
  width: 100%;
  font-size: 12px;
  color: #fca5a5;
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

const MetaPillWrap = styled.span`
  position: relative;
  display: inline-flex;
`;

const MetaPillPopup = styled.div`
  position: absolute;
  left: 0;
  top: calc(100% + 8px);
  z-index: 20;
  min-width: 220px;
  max-width: 300px;
  padding: 10px 12px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: rgba(15, 23, 42, 0.98);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.4);
  pointer-events: none;
  display: grid;
  gap: 4px;
`;

const MetaPillPopupTitle = styled.div`
  font-size: 12px;
  font-weight: 700;
  color: #f8fafc;
`;

const MetaPillPopupDescription = styled.div`
  font-size: 12px;
  line-height: 1.45;
  color: #94a3b8;
`;

const WorkloadTitleWrap = styled.span`
  position: relative;
  display: inline-block;
  max-width: 100%;
`;

const WorkloadTitleButton = styled.span`
  display: inline-block;
  margin: 0;
  font-size: 1.25rem;
  color: #f8fafc;
  font-weight: 700;
  cursor: help;
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
  padding: 0 24px 24px;
`;

const ViewModeBar = styled.div`
  display: flex;
  gap: 10px;
  padding: 16px 24px 0;
`;

const FlowDiagramWrapper = styled.div`
  padding: 20px 24px 24px;
`;

const FlowDiagramContainer = styled.div`
  position: relative;
  min-height: 280px;
`;

const FlowDiagramGrid = styled.div`
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  align-items: start;
  gap: 0;
  width: 100%;
  z-index: 1;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
    gap: 24px;
  }
`;

const FlowColumn = styled.div<{ $side: 'inbound' | 'outbound' }>`
  display: flex;
  flex-direction: column;
  gap: 14px;
  align-items: ${({ $side }) => ($side === 'inbound' ? 'flex-end' : 'flex-start')};
  width: 100%;
  min-width: 0;
`;

const FlowConnectionLane = styled.div`
  width: 100%;
  min-width: 0;
  min-height: 100%;

  @media (max-width: 900px) {
    display: none;
  }
`;

const FlowColumnTitle = styled.div<{ $variant: 'inbound' | 'outbound'; $align?: 'start' | 'end' }>`
  font-size: 11px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: ${({ $variant }) => ($variant === 'outbound' ? '#c4b5fd' : '#67e8f9')};
  margin-bottom: 4px;
  align-self: stretch;
  width: 100%;
  text-align: ${({ $align = 'start' }) => ($align === 'end' ? 'right' : 'left')};
`;

const FlowNodeCard = styled.div<{ $variant: 'inbound' | 'outbound'; $nested?: boolean; $highlighted?: boolean; $dimmed?: boolean }>`
  width: 100%;
  padding: ${({ $nested }) => ($nested ? '10px 12px' : '12px 14px')};
  border-radius: 14px;
  border: 1px solid
    ${({ $variant, $highlighted }) => {
      if ($highlighted) {
        return $variant === 'outbound' ? 'rgba(167, 139, 250, 0.75)' : 'rgba(34, 211, 238, 0.75)';
      }
      return $variant === 'outbound' ? 'rgba(167, 139, 250, 0.28)' : 'rgba(34, 211, 238, 0.28)';
    }};
  background: ${({ $variant, $highlighted }) => {
    if ($highlighted) {
      return $variant === 'outbound'
        ? 'linear-gradient(135deg, rgba(167, 139, 250, 0.22), rgba(99, 102, 241, 0.12))'
        : 'linear-gradient(135deg, rgba(34, 211, 238, 0.22), rgba(14, 165, 233, 0.12))';
    }
    return $variant === 'outbound'
      ? 'linear-gradient(135deg, rgba(167, 139, 250, 0.12), rgba(99, 102, 241, 0.06))'
      : 'linear-gradient(135deg, rgba(34, 211, 238, 0.12), rgba(14, 165, 233, 0.06))';
  }};
  box-shadow: ${({ $highlighted, $variant }) =>
    $highlighted
      ? $variant === 'outbound'
        ? '0 0 16px rgba(167, 139, 250, 0.28)'
        : '0 0 16px rgba(34, 211, 238, 0.28)'
      : 'inset 0 1px 0 rgba(255, 255, 255, 0.04)'};
  opacity: ${({ $dimmed }) => ($dimmed ? 0.38 : 1)};
  display: grid;
  gap: 8px;
  transition: opacity 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease, background 0.15s ease;
  cursor: pointer;
`;

const FlowHttpGroup = styled.div<{ $variant: 'inbound' | 'outbound' }>`
  width: 100%;
  padding: 12px;
  border-radius: 16px;
  border: 1px solid
    ${({ $variant }) => ($variant === 'outbound' ? 'rgba(167, 139, 250, 0.34)' : 'rgba(34, 211, 238, 0.34)')};
  background: ${({ $variant }) =>
    $variant === 'outbound' ? 'rgba(99, 102, 241, 0.08)' : 'rgba(14, 165, 233, 0.08)'};
  display: grid;
  gap: 10px;
`;

const FlowHttpGroupHeader = styled.div`
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  font-size: 13px;
  font-weight: 700;
  color: #f8fafc;
  word-break: break-word;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(148, 163, 184, 0.14);
`;

const FlowHttpGroupMeta = styled.div`
  font-size: 11px;
  color: #94a3b8;
  margin-top: -4px;
`;

const FlowNodeDetails = styled.div`
  margin-top: 2px;
`;

const FlowSvg = styled.svg`
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
  overflow: visible;

  @media (max-width: 900px) {
    display: none;
  }
`;

const FlowEdgeLabelBox = styled.div<{ $fresh?: boolean; $highlighted?: boolean; $dimmed?: boolean; $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  transform: translate(-50%, -50%);
  padding: 5px 8px;
  border-radius: 999px;
  border: 1px solid
    ${({ $fresh, $highlighted }) => {
      if ($highlighted) {
        return $fresh ? 'rgba(34, 211, 238, 0.85)' : 'rgba(148, 163, 184, 0.55)';
      }
      return $fresh ? 'rgba(34, 211, 238, 0.45)' : 'rgba(148, 163, 184, 0.22)';
    }};
  background: ${({ $fresh, $highlighted }) => {
    if ($highlighted) {
      return $fresh ? 'rgba(34, 211, 238, 0.28)' : 'rgba(15, 23, 42, 0.96)';
    }
    return $fresh ? 'rgba(34, 211, 238, 0.16)' : 'rgba(15, 23, 42, 0.92)';
  }};
  color: ${({ $fresh, $highlighted }) => {
    if ($highlighted) {
      return $fresh ? '#a5f3fc' : '#f8fafc';
    }
    return $fresh ? '#67e8f9' : '#cbd5e1';
  }};
  font-size: 11px;
  font-weight: ${({ $highlighted }) => ($highlighted ? 700 : 600)};
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  z-index: ${({ $highlighted }) => ($highlighted ? 3 : 2)};
  pointer-events: none;
  opacity: ${({ $dimmed }) => ($dimmed ? 0.2 : 1)};
  box-shadow: ${({ $fresh, $highlighted }) => {
    if ($highlighted) {
      return $fresh ? '0 0 14px rgba(34, 211, 238, 0.45)' : '0 0 10px rgba(148, 163, 184, 0.25)';
    }
    return $fresh ? '0 0 10px rgba(34, 211, 238, 0.2)' : 'none';
  }};
  transition: opacity 0.15s ease, border-color 0.15s ease, background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;

  @media (max-width: 900px) {
    display: none;
  }
`;

const FlowMobileEdgeList = styled.div`
  display: none;
  flex-direction: column;
  gap: 8px;
  padding: 4px 0;
  min-width: 0;

  @media (max-width: 900px) {
    display: flex;
  }
`;

const FlowMobileEdge = styled.div<{ $fresh?: boolean }>`
  padding: 8px 10px;
  border-radius: 10px;
  border: 1px solid ${({ $fresh }) => ($fresh ? 'rgba(34, 211, 238, 0.35)' : 'rgba(148, 163, 184, 0.16)')};
  background: rgba(15, 23, 42, 0.55);
  color: ${({ $fresh }) => ($fresh ? '#67e8f9' : '#94a3b8')};
  font-size: 11px;
  line-height: 1.45;
`;

const FlowHoverPopup = styled.div<{ $x: number; $y: number; $side: 'inbound' | 'outbound' }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  transform: ${({ $side }) =>
    $side === 'inbound' ? 'translate(14px, -50%)' : 'translate(calc(-100% - 14px), -50%)'};
  min-width: 240px;
  max-width: 320px;
  padding: 12px 14px;
  border-radius: 14px;
  border: 1px solid rgba(148, 163, 184, 0.24);
  background: rgba(15, 23, 42, 0.96);
  box-shadow: 0 16px 40px rgba(0, 0, 0, 0.45);
  z-index: 12;
  pointer-events: none;
  display: grid;
  gap: 10px;

  @media (max-width: 900px) {
    display: none;
  }
`;

const FlowHoverPopupTitle = styled.div`
  font-size: 14px;
  font-weight: 700;
  color: #f8fafc;
  line-height: 1.35;
`;

const FlowHoverPopupScope = styled.div`
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  font-size: 11px;
  color: #94a3b8;
  word-break: break-word;
`;

const FlowHoverPopupSummary = styled.div`
  font-size: 12px;
  line-height: 1.5;
  color: #cbd5e1;
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  word-break: break-word;
`;

const FlowHoverPopupStats = styled.div`
  display: grid;
  gap: 4px;
  padding-top: 4px;
  border-top: 1px solid rgba(148, 163, 184, 0.14);
  font-size: 12px;
  color: #94a3b8;
`;

const FlowHoverPopupStat = styled.div<{ $emphasis?: boolean }>`
  color: ${({ $emphasis }) => ($emphasis ? '#67e8f9' : '#94a3b8')};
  font-weight: ${({ $emphasis }) => ($emphasis ? 600 : 400)};
`;

const FlowHoverPopupAttributes = styled.ul`
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  gap: 4px;
  max-height: 160px;
  overflow: auto;
`;

const FlowHoverPopupAttribute = styled.li`
  font-size: 11px;
  line-height: 1.4;
  word-break: break-word;
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
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

const ScopeSection = styled.div`
  display: grid;
  gap: 12px;
`;

const ScopeGroupHeader = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const ScopeGroupTitle = styled.div`
  font-size: 14px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: 0.01em;
`;

const ScopeGroupSubtitle = styled.div`
  font-size: 12px;
  color: #94a3b8;
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  word-break: break-word;
`;

const ScopeGroupDirection = styled.span`
  color: #64748b;
  font-weight: 500;
`;

const ScopeGroupList = styled.div`
  display: grid;
  gap: 18px;
`;

const InboundPanel = styled.div`
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(34, 211, 238, 0.22);
  background: linear-gradient(135deg, rgba(34, 211, 238, 0.1), rgba(14, 165, 233, 0.04));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
  display: grid;
  gap: 10px;
`;

const OutboundPanel = styled.div`
  padding: 14px 16px;
  border-radius: 14px;
  border: 1px solid rgba(167, 139, 250, 0.22);
  background: linear-gradient(135deg, rgba(167, 139, 250, 0.1), rgba(99, 102, 241, 0.04));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
  min-width: 0;
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
  grid-template-columns: minmax(0, 1fr) auto minmax(148px, auto);
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

const HeaderHttpSummary = styled(HeaderSummary)`
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  color: #e2e8f0;
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
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12px;
  line-height: 1.45;
  word-break: break-word;
`;

const AttributeContent = styled.div`
  flex: 1;
  min-width: 0;
`;

const CopyButton = styled.button<{ $copied?: boolean }>`
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  margin-top: 1px;
  padding: 0;
  border-radius: 6px;
  border: 1px solid ${({ $copied }) => ($copied ? 'rgba(52, 211, 153, 0.45)' : 'rgba(148, 163, 184, 0.18)')};
  background: ${({ $copied }) => ($copied ? 'rgba(16, 185, 129, 0.14)' : 'rgba(15, 23, 42, 0.72)')};
  color: ${({ $copied }) => ($copied ? '#6ee7b7' : '#94a3b8')};
  cursor: pointer;
  opacity: 0.75;
  transition: opacity 0.15s ease, border-color 0.15s ease, background 0.15s ease, color 0.15s ease;

  &:hover {
    opacity: 1;
    border-color: rgba(34, 211, 238, 0.45);
    color: #67e8f9;
  }
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
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-family: 'SF Mono', 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.45;
  color: #e2e8f0;
  word-break: break-word;
`;

const HttpSummaryContent = styled.span`
  flex: 1;
  min-width: 0;
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

const SqlStatement = styled.span`
  display: inline;
  white-space: pre-wrap;
  word-break: break-word;
`;

const SqlLabel = styled.span`
  color: #94a3b8;
  font-weight: 600;
  letter-spacing: 0.04em;
`;

const SqlToken = styled.span<{ $type: SqlTokenType }>`
  color: ${({ $type }) => {
    switch ($type) {
      case 'keyword':
        return '#c084fc';
      case 'string':
        return '#6ee7b7';
      case 'param':
        return '#fbbf24';
      case 'number':
        return '#60a5fa';
      case 'identifier':
        return '#e2e8f0';
      case 'punct':
        return '#64748b';
      default:
        return '#f8fafc';
    }
  }};
`;

const SQL_KEYWORDS = new Set([
  'SELECT',
  'FROM',
  'WHERE',
  'AND',
  'OR',
  'NOT',
  'IN',
  'LIKE',
  'IS',
  'NULL',
  'INSERT',
  'INTO',
  'VALUES',
  'UPDATE',
  'SET',
  'DELETE',
  'JOIN',
  'INNER',
  'LEFT',
  'RIGHT',
  'OUTER',
  'FULL',
  'CROSS',
  'ON',
  'AS',
  'ORDER',
  'BY',
  'GROUP',
  'HAVING',
  'LIMIT',
  'OFFSET',
  'DISTINCT',
  'ALL',
  'UNION',
  'EXISTS',
  'BETWEEN',
  'CASE',
  'WHEN',
  'THEN',
  'ELSE',
  'END',
  'ASC',
  'DESC',
  'TRUE',
  'FALSE',
]);

type SqlTokenType = 'keyword' | 'string' | 'param' | 'number' | 'identifier' | 'punct' | 'ws';

type SqlTokenPart = {
  type: SqlTokenType;
  text: string;
};

function looksLikeSqlStatement(value: string) {
  const trimmed = value.trim();
  if (/^SQL:\s*/i.test(trimmed)) {
    return true;
  }
  return /^(SELECT|INSERT|UPDATE|DELETE|WITH|CREATE|ALTER|DROP)\b/i.test(trimmed);
}

function tokenizeSql(sql: string): SqlTokenPart[] {
  const tokens: SqlTokenPart[] = [];
  let index = 0;

  while (index < sql.length) {
    const rest = sql.slice(index);

    const whitespace = rest.match(/^\s+/);
    if (whitespace) {
      tokens.push({ type: 'ws', text: whitespace[0] });
      index += whitespace[0].length;
      continue;
    }

    const stringLiteral = rest.match(/^'(?:[^'\\]|\\.)*'/);
    if (stringLiteral) {
      tokens.push({ type: 'string', text: stringLiteral[0] });
      index += stringLiteral[0].length;
      continue;
    }

    const parameter = rest.match(/^(\$\d+|\?)/);
    if (parameter) {
      tokens.push({ type: 'param', text: parameter[0] });
      index += parameter[0].length;
      continue;
    }

    const number = rest.match(/^\d+(?:\.\d+)?/);
    if (number) {
      tokens.push({ type: 'number', text: number[0] });
      index += number[0].length;
      continue;
    }

    const word = rest.match(/^[a-zA-Z_][a-zA-Z0-9_]*/);
    if (word) {
      const upper = word[0].toUpperCase();
      tokens.push({
        type: SQL_KEYWORDS.has(upper) ? 'keyword' : 'identifier',
        text: word[0],
      });
      index += word[0].length;
      continue;
    }

    tokens.push({ type: 'punct', text: rest[0] });
    index += 1;
  }

  return tokens;
}

function SqlHighlightedStatement({ value }: { value: string }) {
  const trimmed = value.trim();
  const prefixMatch = trimmed.match(/^(SQL:)(\s*)/i);
  const sqlBody = prefixMatch ? trimmed.slice(prefixMatch[0].length) : trimmed;

  if (!looksLikeSqlStatement(value)) {
    return <>{value}</>;
  }

  const tokens = tokenizeSql(sqlBody);

  return (
    <SqlStatement>
      {prefixMatch ? (
        <>
          <SqlLabel>{prefixMatch[1]}</SqlLabel>
          {prefixMatch[2] || null}
        </>
      ) : null}
      {tokens.map((token, tokenIndex) => (
        <SqlToken key={`${tokenIndex}-${token.text}`} $type={token.type}>
          {token.text}
        </SqlToken>
      ))}
    </SqlStatement>
  );
}

function renderAttributeValue(key: string, value: string) {
  if (key === 'db.statement' && looksLikeSqlStatement(value)) {
    return <SqlHighlightedStatement value={value} />;
  }
  return value;
}

const HTTP_SUMMARY_KEYS = new Set([
  'http.method',
  'http.request.method',
  'url.template',
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
    map['url.template'] ||
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

function CopyAttributeButton({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = useState(false);

  const copy = async (event: React.MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  return (
    <CopyButton
      type='button'
      $copied={copied}
      onClick={copy}
      title={copied ? 'Copied' : label}
      aria-label={copied ? 'Copied' : label}
    >
      {copied ? (
        '✓'
      ) : (
        <svg width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='currentColor' strokeWidth='2' aria-hidden='true'>
          <rect x='9' y='9' width='13' height='13' rx='2' />
          <path d='M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1' />
        </svg>
      )}
    </CopyButton>
  );
}

function getHttpRouteKey(attributes: { key: string; value: string }[]) {
  if (!isHttpAttributes(attributes)) {
    return null;
  }

  const map = attributesToMap(attributes);
  return map['url.template'] || map['http.route'] || null;
}

function getDbGroupKey(attributes: { key: string; value: string }[]) {
  if (!isDbAttributes(attributes)) {
    return null;
  }

  const map = attributesToMap(attributes);
  const dbSystem = map['db.system'];
  if (!dbSystem) {
    return null;
  }

  const dbName = map['db.name'] ?? '';
  return `${dbSystem}\x00${dbName}`;
}

function formatDbGroupTitle(groupKey: string) {
  const [dbSystem, dbName] = groupKey.split('\x00');
  if (dbName) {
    return `${formatDbSystem(dbSystem)} · ${dbName}`;
  }
  return formatDbSystem(dbSystem);
}

function HttpSummaryParts({
  summary,
  hidePath,
}: {
  summary: NonNullable<ReturnType<typeof buildHttpSummary>>;
  hidePath?: boolean;
}) {
  return (
    <>
      {summary.method ? (
        <>
          <HttpMethod $method={summary.method}>{summary.method.toUpperCase()}</HttpMethod>{' '}
        </>
      ) : null}
      {!hidePath && summary.path ? <span>{summary.path}</span> : null}
      {summary.status ? (
        <>
          {(!hidePath && summary.path) || summary.method ? ' ' : null}
          <HttpStatus>{summary.status}</HttpStatus>
        </>
      ) : null}
      {summary.target ? (
        <>
          {' '}
          <HttpTarget>to {summary.target}</HttpTarget>
        </>
      ) : null}
    </>
  );
}

function HttpSummaryLine({
  summary,
  hidePath,
}: {
  summary: NonNullable<ReturnType<typeof buildHttpSummary>>;
  hidePath?: boolean;
}) {
  const summaryText = formatHttpSummaryText(summary, hidePath);

  return (
    <HttpSummary>
      <HttpSummaryContent>
        <HttpSummaryParts summary={summary} hidePath={hidePath} />
      </HttpSummaryContent>
      <CopyAttributeButton text={summaryText} label='Copy HTTP summary' />
    </HttpSummary>
  );
}

function CollapsedAttributeSummary({
  attributes,
  showFullData,
}: {
  attributes: { key: string; value: string }[];
  showFullData: boolean;
}) {
  const visible = filterVisibleAttributes(attributes, showFullData);
  if (!visible.length) {
    return <HeaderSummary>Metadata hidden</HeaderSummary>;
  }

  const { summary } = partitionHttpAttributes(visible);
  if (summary) {
    return (
      <HeaderHttpSummary>
        <HttpSummaryParts summary={summary} />
      </HeaderHttpSummary>
    );
  }

  return <HeaderSummary>{summarizeAttributes(attributes, showFullData)}</HeaderSummary>;
}

function formatHttpSummaryText(summary: NonNullable<ReturnType<typeof buildHttpSummary>>, hidePath = false) {
  const parts: string[] = [];
  if (summary.method) parts.push(summary.method.toUpperCase());
  if (!hidePath && summary.path) parts.push(summary.path);
  if (summary.status) parts.push(summary.status);
  if (summary.target) parts.push(`to ${summary.target}`);
  return parts.join(' ');
}

const RowMeta = styled.div`
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-self: start;
`;

const CountBadge = styled.div`
  padding: 8px 12px;
  border-radius: 12px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(52, 211, 153, 0.25);
  color: #6ee7b7;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
`;

const Timestamp = styled.div<{ $fresh?: boolean }>`
  font-size: 13px;
  padding: ${({ $fresh }) => ($fresh ? '8px 10px' : '0')};
  border-radius: ${({ $fresh }) => ($fresh ? '10px' : '0')};
  background: ${({ $fresh }) => ($fresh ? 'rgba(34, 211, 238, 0.12)' : 'transparent')};
  border: ${({ $fresh }) => ($fresh ? '1px solid rgba(34, 211, 238, 0.38)' : 'none')};
  color: ${({ $fresh }) => ($fresh ? '#67e8f9' : '#cbd5e1')};
  box-shadow: ${({ $fresh }) => ($fresh ? '0 0 14px rgba(34, 211, 238, 0.16)' : 'none')};
  transition: background 0.2s ease, border-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;

  strong {
    color: ${({ $fresh }) => ($fresh ? '#f0fdff' : 'inherit')};
  }
`;

const RelativeTime = styled.span<{ $fresh?: boolean }>`
  display: block;
  margin-top: 4px;
  color: ${({ $fresh }) => ($fresh ? '#22d3ee' : '#64748b')};
  font-size: 12px;
  font-weight: ${({ $fresh }) => ($fresh ? 700 : 500)};
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

const SCOPE_NAME_KEY = 'otel.scope.name';

function getAttributeValue(attributes: { key: string; value: string }[], key: string) {
  return attributes.find((attr) => attr.key === key)?.value;
}

function getScopeName(attributes: { key: string; value: string }[]) {
  return getAttributeValue(attributes, SCOPE_NAME_KEY);
}

function getVisibleScopeName(attributes: { key: string; value: string }[], showFullData: boolean) {
  if (!showFullData) {
    return undefined;
  }
  return getScopeName(attributes);
}

function isDbAttributes(attributes: { key: string; value: string }[]) {
  return attributes.some((attr) => attr.key.startsWith('db.') || attr.key === 'db.system');
}

function formatDbSystem(value: string) {
  const normalized = value.trim().toLowerCase();
  const labels: Record<string, string> = {
    postgresql: 'PostgreSQL',
    postgres: 'PostgreSQL',
    redis: 'Redis',
    mysql: 'MySQL',
    mongodb: 'MongoDB',
    mariadb: 'MariaDB',
    mssql: 'SQL Server',
    sqlite: 'SQLite',
    memcached: 'Memcached',
    elasticsearch: 'Elasticsearch',
    cassandra: 'Cassandra',
    dynamodb: 'DynamoDB',
  };

  return labels[normalized] ?? value.charAt(0).toUpperCase() + value.slice(1);
}

function getScopeGroupTitle(attributes: { key: string; value: string }[], showFullData = true) {
  const map = attributesToMap(attributes);

  if (isDbAttributes(attributes)) {
    const dbSystem = map['db.system'];
    return dbSystem ? `Database · ${formatDbSystem(dbSystem)}` : 'Database';
  }

  if (isHttpAttributes(attributes)) {
    return 'Http';
  }

  if (!showFullData) {
    return 'Connection';
  }

  const scopeName = getScopeName(attributes);
  return scopeName || 'Connection';
}

function ScopeHeader({
  attributes,
  direction,
  showFullData,
}: {
  attributes: { key: string; value: string }[];
  direction: 'inbound' | 'outbound';
  showFullData: boolean;
}) {
  const scopeName = getVisibleScopeName(attributes, showFullData);
  const title = getScopeGroupTitle(attributes, showFullData);

  return (
    <ScopeGroupHeader>
      <ScopeGroupTitle>
        {title}
        <ScopeGroupDirection>{direction === 'inbound' ? ' · Inbound' : ' · Outbound'}</ScopeGroupDirection>
      </ScopeGroupTitle>
      {scopeName ? <ScopeGroupSubtitle>{scopeName}</ScopeGroupSubtitle> : null}
    </ScopeGroupHeader>
  );
}

function groupOutputsByScope(outputs: TraceCorrelationsOutputSeries[], showFullData: boolean) {
  const groups = new Map<string, TraceCorrelationsOutputSeries[]>();

  for (const output of outputs) {
    const groupKey = showFullData
      ? (getScopeName(output.attributes) ?? 'Unknown scope')
      : getScopeGroupTitle(output.attributes, showFullData);
    const existing = groups.get(groupKey) ?? [];
    existing.push(output);
    groups.set(groupKey, existing);
  }

  return Array.from(groups.entries()).sort(([left], [right]) => left.localeCompare(right));
}

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

const FRESH_CONNECTION_MS = 10 * 60 * 1000;

function getFirstSeenAgeMs(value: string, now = Date.now()) {
  if (!value) return Number.POSITIVE_INFINITY;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return Number.POSITIVE_INFINITY;
  return Math.max(0, now - date.getTime());
}

function isFreshConnection(value: string, now = Date.now()) {
  return getFirstSeenAgeMs(value, now) < FRESH_CONNECTION_MS;
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
  const isFresh = useMemo(() => isFreshConnection(value, now), [value, now]);

  return (
    <Timestamp $fresh={isFresh}>
      {isFresh ? 'New connection' : 'First seen'}
      <br />
      <strong>{formatTimestamp(value)}</strong>
      {timeAgo ? <RelativeTime $fresh={isFresh}>{timeAgo} ago</RelativeTime> : null}
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

function AttributeChips({
  attributes,
  showFullData,
}: {
  attributes: { key: string; value: string }[];
  showFullData: boolean;
}) {
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
          <AttributeContent>
            <AttributeKey>{attr.key}</AttributeKey>
            <AttributeValue>{renderAttributeValue(attr.key, attr.value)}</AttributeValue>
          </AttributeContent>
          <CopyAttributeButton text={attr.value} label={`Copy ${attr.key}`} />
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

type FlowNode = {
  id: string;
  attributes: { key: string; value: string }[];
};

type FlowNodeGroup = {
  id: string;
  kind: 'http' | 'database' | 'default';
  title?: string;
  nodes: FlowNode[];
};

type FlowEdge = {
  id: string;
  inputId: string;
  outputId: string;
  connectionCount: number;
  firstDetectedAt: string;
};

type FlowEdgeLayout = FlowEdge & {
  path: string;
  labelX: number;
  labelY: number;
  ageLabel: string;
  fresh: boolean;
};

function groupFlowNodes(nodes: FlowNode[]): FlowNodeGroup[] {
  const httpGroups = new Map<string, FlowNode[]>();
  const dbGroups = new Map<string, FlowNode[]>();
  const ungrouped: FlowNode[] = [];

  for (const node of nodes) {
    const routeKey = getHttpRouteKey(node.attributes);
    if (routeKey) {
      const existing = httpGroups.get(routeKey) ?? [];
      existing.push(node);
      httpGroups.set(routeKey, existing);
      continue;
    }

    const dbKey = getDbGroupKey(node.attributes);
    if (dbKey) {
      const existing = dbGroups.get(dbKey) ?? [];
      existing.push(node);
      dbGroups.set(dbKey, existing);
      continue;
    }

    ungrouped.push(node);
  }

  const groups: FlowNodeGroup[] = [
    ...Array.from(httpGroups.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([routeKey, groupNodes]) => ({
        id: `http:${routeKey}`,
        kind: 'http' as const,
        title: routeKey,
        nodes: [...groupNodes].sort((left, right) => left.id.localeCompare(right.id)),
      })),
    ...Array.from(dbGroups.entries())
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([dbKey, groupNodes]) => ({
        id: `db:${dbKey}`,
        kind: 'database' as const,
        title: formatDbGroupTitle(dbKey),
        nodes: [...groupNodes].sort((left, right) => left.id.localeCompare(right.id)),
      })),
    ...ungrouped
      .sort((left, right) => left.id.localeCompare(right.id))
      .map((node) => ({
        id: node.id,
        kind: 'default' as const,
        nodes: [node],
      })),
  ];

  return groups.sort((left, right) => (left.title ?? left.id).localeCompare(right.title ?? right.id));
}

function buildWorkloadFlowGraph(workload: TraceCorrelationsWorkload) {
  const inputs: FlowNode[] = workload.inputs.map((group) => ({
    id: formatAttributes(group.attributes),
    attributes: group.attributes,
  }));

  const outputNodes = new Map<string, FlowNode>();
  const edges: FlowEdge[] = [];

  for (const group of workload.inputs) {
    const inputId = formatAttributes(group.attributes);
    for (const output of group.outputs) {
      const outputId = formatAttributes(output.attributes);
      if (!outputNodes.has(outputId)) {
        outputNodes.set(outputId, { id: outputId, attributes: output.attributes });
      }
      edges.push({
        id: connectionRowKey(group.attributes, output),
        inputId,
        outputId,
        connectionCount: output.connectionCount,
        firstDetectedAt: output.firstDetectedAt,
      });
    }
  }

  const outputs = Array.from(outputNodes.values()).sort((left, right) => left.id.localeCompare(right.id));

  return {
    inputs: [...inputs].sort((left, right) => left.id.localeCompare(right.id)),
    outputs,
    inputGroups: groupFlowNodes(inputs),
    outputGroups: groupFlowNodes(outputs),
    edges,
  };
}

type FlowHoverPopupAnchor = {
  x: number;
  y: number;
  side: 'inbound' | 'outbound';
};

type FlowNodeHoverInfo = {
  node: FlowNode;
  direction: 'inbound' | 'outbound';
  title: string;
  scopeName?: string;
  summary: string;
  totalConnections: number;
  peerCount: number;
  earliestFirstSeen?: string;
  fresh: boolean;
};

function buildFlowNodeHoverInfo(
  nodeId: string,
  graph: ReturnType<typeof buildWorkloadFlowGraph>,
  showFullData: boolean,
  now: number,
): FlowNodeHoverInfo | null {
  const node = graph.inputs.find((entry) => entry.id === nodeId) ?? graph.outputs.find((entry) => entry.id === nodeId);
  if (!node) {
    return null;
  }

  const direction = graph.inputs.some((entry) => entry.id === nodeId) ? 'inbound' : 'outbound';
  const relatedEdges = graph.edges.filter((edge) => edge.inputId === nodeId || edge.outputId === nodeId);
  const totalConnections = relatedEdges.reduce((sum, edge) => sum + edge.connectionCount, 0);
  const peerCount = new Set(
    relatedEdges.map((edge) => (edge.inputId === nodeId ? edge.outputId : edge.inputId)),
  ).size;

  let earliestFirstSeen: string | undefined;
  for (const edge of relatedEdges) {
    if (!earliestFirstSeen || edge.firstDetectedAt < earliestFirstSeen) {
      earliestFirstSeen = edge.firstDetectedAt;
    }
  }

  return {
    node,
    direction,
    title: getScopeGroupTitle(node.attributes, showFullData),
    scopeName: getVisibleScopeName(node.attributes, showFullData),
    summary: summarizeAttributes(node.attributes, showFullData),
    totalConnections,
    peerCount,
    earliestFirstSeen,
    fresh: earliestFirstSeen ? isFreshConnection(earliestFirstSeen, now) : false,
  };
}

function FlowNodeHoverPopup({
  info,
  anchor,
  now,
  showFullData,
}: {
  info: FlowNodeHoverInfo;
  anchor: FlowHoverPopupAnchor;
  now: number;
  showFullData: boolean;
}) {
  const visibleAttributes = filterVisibleAttributes(info.node.attributes, showFullData);
  const peerLabel = info.direction === 'inbound' ? 'outbound' : 'inbound';

  return (
    <FlowHoverPopup $x={anchor.x} $y={anchor.y} $side={anchor.side}>
      <FlowHoverPopupTitle>
        {info.title} · {info.direction === 'inbound' ? 'Inbound' : 'Outbound'}
      </FlowHoverPopupTitle>
      {info.scopeName ? <FlowHoverPopupScope>{info.scopeName}</FlowHoverPopupScope> : null}
      <FlowHoverPopupSummary>{info.summary}</FlowHoverPopupSummary>
      <FlowHoverPopupStats>
        <FlowHoverPopupStat $emphasis>{info.totalConnections.toLocaleString()} total connections</FlowHoverPopupStat>
        <FlowHoverPopupStat>
          {info.peerCount} linked {peerLabel} pattern{info.peerCount === 1 ? '' : 's'}
        </FlowHoverPopupStat>
        {info.earliestFirstSeen ? (
          <FlowHoverPopupStat $emphasis={info.fresh}>
            First seen {formatTimestamp(info.earliestFirstSeen)}
            {formatTimeAgo(info.earliestFirstSeen, now) ? ` (${formatTimeAgo(info.earliestFirstSeen, now)} ago)` : ''}
          </FlowHoverPopupStat>
        ) : null}
      </FlowHoverPopupStats>
      {visibleAttributes.length ? (
        <FlowHoverPopupAttributes>
          {visibleAttributes.map((attr) => (
            <FlowHoverPopupAttribute key={`${attr.key}:${attr.value}`}>
              <span style={{ color: '#64748b' }}>{attr.key}: </span>
              {attr.key === 'db.statement' && looksLikeSqlStatement(attr.value) ? (
                <SqlHighlightedStatement value={attr.value} />
              ) : (
                attr.value
              )}
            </FlowHoverPopupAttribute>
          ))}
        </FlowHoverPopupAttributes>
      ) : null}
    </FlowHoverPopup>
  );
}

function getFlowConnectionsForNode(edges: FlowEdge[], nodeId: string | null) {
  if (!nodeId) {
    return { edgeIds: new Set<string>(), nodeIds: new Set<string>() };
  }

  const edgeIds = new Set<string>();
  const nodeIds = new Set<string>();

  for (const edge of edges) {
    if (edge.inputId === nodeId || edge.outputId === nodeId) {
      edgeIds.add(edge.id);
      nodeIds.add(edge.inputId === nodeId ? edge.outputId : edge.inputId);
    }
  }

  return { edgeIds, nodeIds };
}

function getFlowNodeHighlightState(nodeId: string, activeNodeId: string | null, connectedNodeIds: Set<string>) {
  if (!activeNodeId) {
    return { highlighted: false, dimmed: false };
  }
  if (nodeId === activeNodeId || connectedNodeIds.has(nodeId)) {
    return { highlighted: true, dimmed: false };
  }
  return { highlighted: false, dimmed: true };
}

function FlowNodeView({
  node,
  variant,
  showFullData,
  nodeRef,
  nested,
  highlighted,
  dimmed,
  onHoverStart,
  onClick,
}: {
  node: FlowNode;
  variant: 'inbound' | 'outbound';
  showFullData: boolean;
  nodeRef: (element: HTMLDivElement | null) => void;
  nested?: boolean;
  highlighted?: boolean;
  dimmed?: boolean;
  onHoverStart: (event: React.MouseEvent<HTMLDivElement>) => void;
  onClick: (event: React.MouseEvent<HTMLDivElement>) => void;
}) {
  return (
    <FlowNodeCard
      ref={nodeRef}
      $variant={variant}
      $nested={nested}
      $highlighted={highlighted}
      $dimmed={dimmed}
      onMouseEnter={onHoverStart}
      onClick={onClick}
    >
      <ScopeHeader attributes={node.attributes} direction={variant} showFullData={showFullData} />
      <FlowNodeDetails>
        <AttributeChips attributes={node.attributes} showFullData={showFullData} />
      </FlowNodeDetails>
    </FlowNodeCard>
  );
}

function FlowNodeGroupView({
  group,
  variant,
  showFullData,
  onNodeRef,
  activeNodeId,
  connectedNodeIds,
  onNodeHover,
  onNodeClick,
}: {
  group: FlowNodeGroup;
  variant: 'inbound' | 'outbound';
  showFullData: boolean;
  onNodeRef: (nodeId: string, element: HTMLDivElement | null) => void;
  activeNodeId: string | null;
  connectedNodeIds: Set<string>;
  onNodeHover: (nodeId: string, event: React.MouseEvent<HTMLDivElement>, side: 'inbound' | 'outbound') => void;
  onNodeClick: (nodeId: string, event: React.MouseEvent<HTMLDivElement>) => void;
}) {
  if (group.kind === 'default') {
    const node = group.nodes[0];
    const { highlighted, dimmed } = getFlowNodeHighlightState(node.id, activeNodeId, connectedNodeIds);
    return (
      <FlowNodeView
        node={node}
        variant={variant}
        showFullData={showFullData}
        highlighted={highlighted}
        dimmed={dimmed}
        onHoverStart={(event) => onNodeHover(node.id, event, variant)}
        onClick={(event) => onNodeClick(node.id, event)}
        nodeRef={(element) => onNodeRef(node.id, element)}
      />
    );
  }

  const isHttpGroup = group.kind === 'http';

  return (
    <FlowHttpGroup $variant={variant}>
      <FlowHttpGroupHeader>{group.title}</FlowHttpGroupHeader>
      <FlowHttpGroupMeta>
        {group.nodes.length} {isHttpGroup ? 'endpoint' : 'pattern'}
        {group.nodes.length === 1 ? '' : 's'}
      </FlowHttpGroupMeta>
      {group.nodes.map((node) => {
        const { highlighted, dimmed } = getFlowNodeHighlightState(node.id, activeNodeId, connectedNodeIds);
        return (
          <FlowNodeView
            key={node.id}
            node={node}
            variant={variant}
            showFullData={showFullData}
            nested
            highlighted={highlighted}
            dimmed={dimmed}
            onHoverStart={(event) => onNodeHover(node.id, event, variant)}
            onClick={(event) => onNodeClick(node.id, event)}
            nodeRef={(element) => onNodeRef(node.id, element)}
          />
        );
      })}
    </FlowHttpGroup>
  );
}

function ServiceFlowDiagram({ workload, showFullData }: { workload: TraceCorrelationsWorkload; showFullData: boolean }) {
  const graph = useMemo(() => buildWorkloadFlowGraph(workload), [workload]);
  const containerRef = useRef<HTMLDivElement>(null);
  const nodeRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const [now, setNow] = useState(() => Date.now());
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [hoverPopupAnchor, setHoverPopupAnchor] = useState<FlowHoverPopupAnchor | null>(null);
  const [layout, setLayout] = useState<{ width: number; height: number; edges: FlowEdgeLayout[] }>({
    width: 0,
    height: 0,
    edges: [],
  });

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 60_000);
    return () => window.clearInterval(interval);
  }, []);

  const measureLayout = useCallback(() => {
    const container = containerRef.current;
    if (!container || !graph.edges.length) {
      return { width: container?.offsetWidth ?? 0, height: container?.offsetHeight ?? 0, edges: [] as FlowEdgeLayout[] };
    }

    const containerRect = container.getBoundingClientRect();
    const edges = graph.edges
      .map((edge) => {
        const fromNode = nodeRefs.current[edge.inputId];
        const toNode = nodeRefs.current[edge.outputId];
        if (!fromNode || !toNode) {
          return null;
        }

        const fromRect = fromNode.getBoundingClientRect();
        const toRect = toNode.getBoundingClientRect();
        const x1 = fromRect.right - containerRect.left;
        const y1 = fromRect.top + fromRect.height / 2 - containerRect.top;
        const x2 = toRect.left - containerRect.left;
        const y2 = toRect.top + toRect.height / 2 - containerRect.top;
        const controlX = (x1 + x2) / 2;

        return {
          ...edge,
          path: `M ${x1} ${y1} C ${controlX} ${y1}, ${controlX} ${y2}, ${x2} ${y2}`,
          labelX: controlX,
          labelY: (y1 + y2) / 2,
          ageLabel: formatTimeAgo(edge.firstDetectedAt, now) || 'unknown',
          fresh: isFreshConnection(edge.firstDetectedAt, now),
        };
      })
      .filter((edge): edge is FlowEdgeLayout => edge !== null);

    return {
      width: container.offsetWidth,
      height: container.offsetHeight,
      edges,
    };
  }, [graph.edges, now]);

  const applyLayout = useCallback(() => {
    setLayout(measureLayout());
  }, [measureLayout]);

  useEffect(() => {
    const frame = requestAnimationFrame(applyLayout);
    return () => cancelAnimationFrame(frame);
  }, [now, applyLayout]);

  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) {
      return undefined;
    }

    let frame = requestAnimationFrame(applyLayout);

    const scheduleLayout = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(applyLayout);
    };

    const resizeObserver = new ResizeObserver(scheduleLayout);
    resizeObserver.observe(container);
    window.addEventListener('resize', scheduleLayout);

    return () => {
      cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      window.removeEventListener('resize', scheduleLayout);
    };
  }, [applyLayout, graph.inputs, graph.outputs]);

  const inputSummaryById = useMemo(
    () => new Map(graph.inputs.map((node) => [node.id, summarizeAttributes(node.attributes, showFullData)])),
    [graph.inputs, showFullData],
  );
  const outputSummaryById = useMemo(
    () => new Map(graph.outputs.map((node) => [node.id, summarizeAttributes(node.attributes, showFullData)])),
    [graph.outputs, showFullData],
  );

  const activeNodeId = hoveredNodeId ?? selectedNodeId;

  const { edgeIds: connectedEdgeIds, nodeIds: connectedNodeIds } = useMemo(
    () => getFlowConnectionsForNode(graph.edges, activeNodeId),
    [graph.edges, activeNodeId],
  );

  const highlightActive = activeNodeId !== null;

  const hoveredNodeInfo = useMemo(() => {
    if (!hoveredNodeId) {
      return null;
    }
    return buildFlowNodeHoverInfo(hoveredNodeId, graph, showFullData, now);
  }, [hoveredNodeId, graph, showFullData, now]);

  const handleNodeHover = useCallback(
    (nodeId: string, event: React.MouseEvent<HTMLDivElement>, side: 'inbound' | 'outbound') => {
      setHoveredNodeId(nodeId);
      const container = containerRef.current;
      const target = event.currentTarget;
      if (!container || !target) {
        return;
      }

      const containerRect = container.getBoundingClientRect();
      const nodeRect = target.getBoundingClientRect();
      setHoverPopupAnchor({
        x: side === 'inbound' ? nodeRect.right - containerRect.left : nodeRect.left - containerRect.left,
        y: nodeRect.top + nodeRect.height / 2 - containerRect.top,
        side,
      });
    },
    [],
  );

  const handleNodeClick = useCallback((nodeId: string, event: React.MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    setSelectedNodeId((current) => (current === nodeId ? null : nodeId));
  }, []);

  const clearHover = useCallback(() => {
    setHoveredNodeId(null);
    setHoverPopupAnchor(null);
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedNodeId(null);
  }, []);

  if (!graph.inputs.length) {
    return <HiddenAttributesNote>No connection patterns to visualize yet.</HiddenAttributesNote>;
  }

  return (
    <FlowDiagramWrapper>
      <FlowDiagramContainer ref={containerRef} onMouseLeave={clearHover} onClick={clearSelection}>
        {hoveredNodeInfo && hoverPopupAnchor ? (
          <FlowNodeHoverPopup info={hoveredNodeInfo} anchor={hoverPopupAnchor} now={now} showFullData={showFullData} />
        ) : null}

        {layout.width > 0 && layout.height > 0 ? (
          <FlowSvg viewBox={`0 0 ${layout.width} ${layout.height}`} preserveAspectRatio='none'>
            {layout.edges.map((edge) => {
              const highlighted = !highlightActive || connectedEdgeIds.has(edge.id);
              return (
                <path
                  key={edge.id}
                  d={edge.path}
                  fill='none'
                  stroke={highlighted ? (edge.fresh ? 'rgba(34, 211, 238, 0.9)' : 'rgba(148, 163, 184, 0.65)') : 'rgba(148, 163, 184, 0.14)'}
                  strokeWidth={highlighted ? (edge.fresh ? 3 : 2.5) : 1.25}
                  strokeOpacity={highlighted ? 1 : 0.35}
                />
              );
            })}
          </FlowSvg>
        ) : null}

        {layout.edges.map((edge) => {
          const highlighted = !highlightActive || connectedEdgeIds.has(edge.id);
          return (
            <FlowEdgeLabelBox
              key={`label-${edge.id}`}
              $fresh={edge.fresh}
              $highlighted={highlighted}
              $dimmed={highlightActive && !highlighted}
              $x={edge.labelX}
              $y={edge.labelY}
            >
              {edge.connectionCount.toLocaleString()} · {edge.ageLabel}
            </FlowEdgeLabelBox>
          );
        })}

        <FlowDiagramGrid>
          <FlowColumn $side='inbound'>
            <FlowColumnTitle $variant='inbound' $align='start'>
              Inbound
            </FlowColumnTitle>
            {graph.inputGroups.map((group) => (
              <FlowNodeGroupView
                key={group.id}
                group={group}
                variant='inbound'
                showFullData={showFullData}
                activeNodeId={activeNodeId}
                connectedNodeIds={connectedNodeIds}
                onNodeHover={handleNodeHover}
                onNodeClick={handleNodeClick}
                onNodeRef={(nodeId, element) => {
                  nodeRefs.current[nodeId] = element;
                  if (element) {
                    requestAnimationFrame(applyLayout);
                  }
                }}
              />
            ))}
          </FlowColumn>

          <FlowConnectionLane aria-hidden='true' />

          <FlowMobileEdgeList aria-hidden='true'>
            {graph.edges.map((edge) => {
              const ageLabel = formatTimeAgo(edge.firstDetectedAt, now) || 'unknown';
              const fresh = isFreshConnection(edge.firstDetectedAt, now);
              return (
                <FlowMobileEdge key={`mobile-${edge.id}`} $fresh={fresh}>
                  {inputSummaryById.get(edge.inputId)} → {outputSummaryById.get(edge.outputId)}
                  <br />
                  {edge.connectionCount.toLocaleString()} · {ageLabel}
                </FlowMobileEdge>
              );
            })}
          </FlowMobileEdgeList>

          <FlowColumn $side='outbound'>
            <FlowColumnTitle $variant='outbound' $align='end'>
              Outbound
            </FlowColumnTitle>
            {graph.outputGroups.map((group) => (
              <FlowNodeGroupView
                key={group.id}
                group={group}
                variant='outbound'
                showFullData={showFullData}
                activeNodeId={activeNodeId}
                connectedNodeIds={connectedNodeIds}
                onNodeHover={handleNodeHover}
                onNodeClick={handleNodeClick}
                onNodeRef={(nodeId, element) => {
                  nodeRefs.current[nodeId] = element;
                  if (element) {
                    requestAnimationFrame(applyLayout);
                  }
                }}
              />
            ))}
          </FlowColumn>
        </FlowDiagramGrid>
      </FlowDiagramContainer>
    </FlowDiagramWrapper>
  );
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
        <AttributeChips attributes={output.attributes} showFullData={showFullData} />
      </OutboundPanel>
      <RowMeta>
        <CountBadge>{output.connectionCount.toLocaleString()} connections</CountBadge>
        <FirstSeenTimestamp value={output.firstDetectedAt} />
      </RowMeta>
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
  const [expanded, setExpanded] = useState(false);
  const outboundScopeGroups = useMemo(() => groupOutputsByScope(group.outputs, showFullData), [group.outputs, showFullData]);
  const inboundTitle = useMemo(() => getScopeGroupTitle(group.attributes, showFullData), [group.attributes, showFullData]);
  const inboundScopeName = useMemo(() => getVisibleScopeName(group.attributes, showFullData), [group.attributes, showFullData]);

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
            {expanded ? (
              <HeaderSummary>{group.outputs.length} outbound pattern{group.outputs.length === 1 ? '' : 's'}</HeaderSummary>
            ) : (
              <>
                <ScopeGroupTitle>{inboundTitle}</ScopeGroupTitle>
                {inboundScopeName ? <ScopeGroupSubtitle>{inboundScopeName}</ScopeGroupSubtitle> : null}
                <CollapsedAttributeSummary attributes={group.attributes} showFullData={showFullData} />
              </>
            )}
          </div>
        </InputHeaderRow>
      </InputHeader>

      {expanded ? (
        <FlowBody>
          <InboundPanel>
            <ScopeHeader attributes={group.attributes} direction='inbound' showFullData={showFullData} />
            <AttributeChips attributes={group.attributes} showFullData={showFullData} />
          </InboundPanel>

          <FlowDivider>
            <FlowDividerLine />
            <FlowArrowBadge>→ triggers</FlowArrowBadge>
            <FlowDividerLine />
          </FlowDivider>

          <ScopeGroupList>
            {outboundScopeGroups.map(([scopeName, outputs]) => (
              <ScopeSection key={scopeName}>
                <ScopeHeader attributes={outputs[0].attributes} direction='outbound' showFullData={showFullData} />
                <OutputFlowList>
                  {outputs.map((output) => (
                    <ConnectionRow
                      key={connectionRowKey(group.attributes, output)}
                      output={output}
                      showFullData={showFullData}
                    />
                  ))}
                </OutputFlowList>
              </ScopeSection>
            ))}
          </ScopeGroupList>
        </FlowBody>
      ) : null}
    </InputGroupCard>
  );
}

function formatWorkloadRuntimeLabel(runtimeName?: string | null, runtimeVersion?: string | null) {
  if (runtimeName && runtimeVersion) {
    return `${runtimeName} ${runtimeVersion}`;
  }
  return runtimeName || runtimeVersion || null;
}

function WorkloadHeaderHint({
  label,
  title,
  description,
  variant = 'pill',
}: {
  label: string;
  title: string;
  description: string;
  variant?: 'pill' | 'title';
}) {
  const [open, setOpen] = useState(false);
  const wrapProps = {
    onMouseEnter: () => setOpen(true),
    onMouseLeave: () => setOpen(false),
    onClick: (event: React.MouseEvent) => event.stopPropagation(),
    onFocus: () => setOpen(true),
    onBlur: () => setOpen(false),
  };

  if (variant === 'title') {
    return (
      <WorkloadTitleWrap {...wrapProps} tabIndex={0}>
        <WorkloadTitleButton>{label}</WorkloadTitleButton>
        {open ? (
          <MetaPillPopup>
            <MetaPillPopupTitle>{title}</MetaPillPopupTitle>
            <MetaPillPopupDescription>{description}</MetaPillPopupDescription>
          </MetaPillPopup>
        ) : null}
      </WorkloadTitleWrap>
    );
  }

  return (
    <MetaPillWrap {...wrapProps} tabIndex={0}>
      <Pill>{label}</Pill>
      {open ? (
        <MetaPillPopup>
          <MetaPillPopupTitle>{title}</MetaPillPopupTitle>
          <MetaPillPopupDescription>{description}</MetaPillPopupDescription>
        </MetaPillPopup>
      ) : null}
    </MetaPillWrap>
  );
}

function WorkloadCardView({ workload, showFullData }: { workload: TraceCorrelationsWorkload; showFullData: boolean }) {
  const [expanded, setExpanded] = useState(false);
  const [viewMode, setViewMode] = useState<'graph' | 'details'>('graph');
  const stats = workloadStats(workload);
  const runtimeLabel = formatWorkloadRuntimeLabel(workload.processRuntimeName, workload.processRuntimeVersion);

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
            <WorkloadTitle>
              <WorkloadHeaderHint
                variant='title'
                label={workload.name}
                title='Workload'
                description='The Kubernetes workload resource name for this service.'
              />
            </WorkloadTitle>
            <WorkloadMeta>
              <WorkloadHeaderHint
                label={workload.namespace}
                title='Namespace'
                description='The Kubernetes namespace where this workload is deployed.'
              />
              <WorkloadHeaderHint
                label={workload.kind}
                title='Kind'
                description='The Kubernetes workload resource kind, such as Deployment or StatefulSet.'
              />
              <WorkloadHeaderHint
                label={workload.containerName}
                title='Container'
                description='The instrumented container name within the workload pods.'
              />
              {workload.telemetrySdkLanguage ? (
                <WorkloadHeaderHint
                  label={workload.telemetrySdkLanguage}
                  title='SDK language'
                  description='The OpenTelemetry SDK language detected from trace resource attributes.'
                />
              ) : null}
              {runtimeLabel ? (
                <WorkloadHeaderHint
                  label={runtimeLabel}
                  title='Runtime'
                  description='The process runtime name and version reported by the instrumented application.'
                />
              ) : null}
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
        <>
          <ViewModeBar>
            <ToggleButton
              type='button'
              $active={viewMode === 'graph'}
              onClick={(event) => {
                event.stopPropagation();
                setViewMode('graph');
              }}
            >
              Graph
            </ToggleButton>
            <ToggleButton
              type='button'
              $active={viewMode === 'details'}
              onClick={(event) => {
                event.stopPropagation();
                setViewMode('details');
              }}
            >
              Details
            </ToggleButton>
          </ViewModeBar>

          {viewMode === 'graph' ? (
            <ServiceFlowDiagram workload={workload} showFullData={showFullData} />
          ) : (
            <InputGroupList>
              {workload.inputs.map((input) => (
                <InputGroupView key={formatAttributes(input.attributes)} group={input} showFullData={showFullData} />
              ))}
            </InputGroupList>
          )}
        </>
      ) : null}
    </WorkloadCard>
  );
}

function createDefaultCustomRange(now = Date.now()) {
  const end = new Date(now);
  const start = new Date(now - 60 * 60 * 1000);
  return {
    start: toDatetimeLocalValue(start),
    end: toDatetimeLocalValue(end),
  };
}

export default function TraceCorrelationsPage() {
  const [namespaceFilter, setNamespaceFilter] = useState('');
  const [showFullData, setShowFullData] = useState(false);
  const [timePreset, setTimePreset] = useState<TraceCorrelationsTimePreset>('1h');
  const [customStart, setCustomStart] = useState(() => createDefaultCustomRange().start);
  const [customEnd, setCustomEnd] = useState(() => createDefaultCustomRange().end);
  const [queryAnchor, setQueryAnchor] = useState(() => Date.now());
  const filter = namespaceFilter.trim() ? { namespace: namespaceFilter.trim() } : undefined;
  const timeRange = useMemo(
    () => resolveTraceCorrelationsTimeRange({ preset: timePreset, customStart, customEnd, now: queryAnchor }),
    [timePreset, customStart, customEnd, queryAnchor],
  );
  const { workloads, loading, error, refetch } = useTraceCorrelations(filter, timeRange);

  const customRangeInvalid = timePreset === 'custom' && !timeRange;
  const dataCoverageLabel = timeRange ? formatTraceCorrelationsTimeRangeLabel(timeRange) : null;

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
        <TimeRangeBar>
          <TimeRangeLabel>Time range</TimeRangeLabel>
          <TimeRangePresets>
            {TRACE_CORRELATIONS_TIME_PRESETS.map((preset) => (
              <ToggleButton
                key={preset.id}
                type='button'
                $active={timePreset === preset.id}
                onClick={() => {
                  setQueryAnchor(Date.now());
                  setTimePreset(preset.id);
                }}
              >
                {preset.label}
              </ToggleButton>
            ))}
            <ToggleButton
              type='button'
              $active={timePreset === 'custom'}
              onClick={() => {
                const defaults = createDefaultCustomRange();
                setCustomStart(defaults.start);
                setCustomEnd(defaults.end);
                setTimePreset('custom');
              }}
            >
              Custom
            </ToggleButton>
          </TimeRangePresets>
          {timePreset === 'custom' ? (
            <TimeRangeCustomFields>
              <TimeRangeField>
                From
                <DateTimeInput
                  type='datetime-local'
                  value={customStart}
                  onChange={(event) => setCustomStart(event.target.value)}
                />
              </TimeRangeField>
              <TimeRangeField>
                To
                <DateTimeInput
                  type='datetime-local'
                  value={customEnd}
                  onChange={(event) => setCustomEnd(event.target.value)}
                />
              </TimeRangeField>
            </TimeRangeCustomFields>
          ) : null}
          <TimeRangeHint>Use Refresh to load the latest data for the selected range.</TimeRangeHint>
          {customRangeInvalid ? <TimeRangeError>End time must be after start time.</TimeRangeError> : null}
        </TimeRangeBar>

        {dataCoverageLabel ? (
          <DataCoverageBanner>
            <DataCoverageLabel>{dataCoverageLabel}</DataCoverageLabel>
          </DataCoverageBanner>
        ) : null}

        <Hero>
          <div>
            <Eyebrow>Trace Correlations</Eyebrow>
            <Title>Service I/O Connections</Title>
            <Subtitle>
              Inbound-to-outbound span correlations emitted by the serviceio connector for the selected time window.
              Each service card shows a graph of input and output patterns with connection counts and age, or switch to
              Details for the full breakdown.
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
            <Button
              onClick={() => {
                if (timePreset !== 'custom') {
                  setQueryAnchor(Date.now());
                }
                refetch();
              }}
              disabled={loading || customRangeInvalid}
            >
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

        {!customRangeInvalid && loading && !workloads.length && timeRange ? (
          <LoadingState>Loading trace correlations…</LoadingState>
        ) : null}

        {!customRangeInvalid && !loading && !error && !workloads.length && timeRange ? (
          <EmptyState>No trace correlation data found yet. Make sure trace correlations are enabled and traffic is flowing.</EmptyState>
        ) : null}

        {customRangeInvalid ? (
          <EmptyState>Select a valid custom time range to load trace correlations.</EmptyState>
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
