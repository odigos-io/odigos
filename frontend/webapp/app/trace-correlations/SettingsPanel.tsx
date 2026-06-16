'use client';

import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import type { TraceCorrelationsSettings } from '@/hooks/metrics/useTraceCorrelationsSettings';
import { groupAttributesByPrefix } from './attributeGroups';

const Panel = styled.section`
  margin-bottom: 24px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.14);
  background: rgba(15, 23, 42, 0.72);
  overflow: hidden;
`;

const PanelHeader = styled.button`
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 16px 18px;
  border: none;
  background: transparent;
  color: #e2e8f0;
  cursor: pointer;
  text-align: left;
`;

const PanelTitle = styled.div`
  font-size: 15px;
  font-weight: 700;
  color: #f8fafc;
`;

const PanelSubtitle = styled.div`
  margin-top: 4px;
  font-size: 13px;
  color: #94a3b8;
`;

const PanelBody = styled.div`
  padding: 0 18px 18px;
  display: grid;
  gap: 18px;
`;

const FieldGroup = styled.div`
  display: grid;
  gap: 10px;
`;

const FieldLabel = styled.label`
  display: grid;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #cbd5e1;
`;

const FieldHint = styled.div`
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
  line-height: 1.5;
`;

const TextInput = styled.input`
  width: 100%;
  padding: 11px 12px;
  border-radius: 12px;
  border: 1px solid rgba(148, 163, 184, 0.18);
  background: rgba(2, 6, 23, 0.72);
  color: #f8fafc;
  outline: none;

  &:focus {
    border-color: rgba(34, 211, 238, 0.55);
    box-shadow: 0 0 0 3px rgba(34, 211, 238, 0.12);
  }
`;

const AttributeEditor = styled.div`
  display: grid;
  gap: 10px;
`;

const AttributeGroups = styled.div`
  display: grid;
  gap: 14px;
`;

const AttributeGroup = styled.div`
  display: grid;
  gap: 8px;
`;

const AttributeGroupLabel = styled.div`
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: #64748b;
`;

const AttributeGroupList = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
`;

const AttributeChip = styled.span`
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border-radius: 999px;
  background: rgba(34, 211, 238, 0.1);
  border: 1px solid rgba(34, 211, 238, 0.22);
  color: #cffafe;
  font-size: 12px;
  font-family: 'SF Mono', 'Fira Code', ui-monospace, monospace;
`;

const RemoveChipButton = styled.button`
  border: none;
  background: transparent;
  color: #67e8f9;
  cursor: pointer;
  padding: 0;
  font-size: 14px;
  line-height: 1;
`;

const AddRow = styled.div`
  display: flex;
  gap: 8px;
`;

const ActionButton = styled.button<{ $primary?: boolean }>`
  padding: 10px 14px;
  border-radius: 12px;
  border: 1px solid ${({ $primary }) => ($primary ? 'rgba(34, 211, 238, 0.45)' : 'rgba(148, 163, 184, 0.18)')};
  background: ${({ $primary }) => ($primary ? 'rgba(34, 211, 238, 0.14)' : 'rgba(15, 23, 42, 0.72)')};
  color: ${({ $primary }) => ($primary ? '#67e8f9' : '#cbd5e1')};
  font-weight: 600;
  cursor: pointer;

  &:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }
`;

const ActionsRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: flex-end;
`;

const EmptyAttributes = styled.div`
  color: #64748b;
  font-size: 12px;
  font-style: italic;
`;

const ErrorText = styled.div`
  color: #fca5a5;
  font-size: 12px;
`;

type AttributeListEditorProps = {
  label: string;
  hint: string;
  values: string[];
  onChange: (values: string[]) => void;
};

function AttributeListEditor({ label, hint, values, onChange }: AttributeListEditorProps) {
  const [draftValue, setDraftValue] = useState('');
  const groupedValues = useMemo(() => groupAttributesByPrefix(values), [values]);

  const addValue = () => {
    const trimmed = draftValue.trim();
    if (!trimmed) {
      return;
    }
    const exists = values.some((value) => value.toLowerCase() === trimmed.toLowerCase());
    if (exists) {
      setDraftValue('');
      return;
    }
    onChange([...values, trimmed]);
    setDraftValue('');
  };

  return (
    <FieldGroup>
      <FieldLabel>
        {label}
        <FieldHint>{hint}</FieldHint>
      </FieldLabel>
      <AttributeEditor>
        {values.length ? (
          <AttributeGroups>
            {groupedValues.map((group) => (
              <AttributeGroup key={group.prefix}>
                <AttributeGroupLabel>{group.label}</AttributeGroupLabel>
                <AttributeGroupList>
                  {group.values.map((value) => (
                    <AttributeChip key={value}>
                      {value}
                      <RemoveChipButton
                        type='button'
                        aria-label={`Remove ${value}`}
                        onClick={() => onChange(values.filter((item) => item !== value))}
                      >
                        ×
                      </RemoveChipButton>
                    </AttributeChip>
                  ))}
                </AttributeGroupList>
              </AttributeGroup>
            ))}
          </AttributeGroups>
        ) : (
          <EmptyAttributes>No attributes configured.</EmptyAttributes>
        )}
        <AddRow>
          <TextInput
            value={draftValue}
            placeholder='e.g. http.route'
            onChange={(event) => setDraftValue(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                addValue();
              }
            }}
          />
          <ActionButton type='button' onClick={addValue}>
            Add
          </ActionButton>
        </AddRow>
      </AttributeEditor>
    </FieldGroup>
  );
}

type TraceCorrelationsSettingsPanelProps = {
  open: boolean;
  onToggle: () => void;
  draft: TraceCorrelationsSettings;
  onChange: (settings: TraceCorrelationsSettings) => void;
  isDirty: boolean;
  flushIntervalInvalid: boolean;
  saving: boolean;
  onSave: () => void;
  onReset: () => void;
};

export function TraceCorrelationsSettingsPanel({
  open,
  onToggle,
  draft,
  onChange,
  isDirty,
  flushIntervalInvalid,
  saving,
  onSave,
  onReset,
}: TraceCorrelationsSettingsPanelProps) {
  return (
    <Panel>
      <PanelHeader type='button' onClick={onToggle}>
        <div>
          <PanelTitle>Service I/O settings</PanelTitle>
          <PanelSubtitle>Configure span attributes used for correlations and how often metrics are flushed.</PanelSubtitle>
        </div>
        <span aria-hidden='true'>{open ? '▾' : '▸'}</span>
      </PanelHeader>

      {open ? (
        <PanelBody>
          <AttributeListEditor
            label='Inbound span attributes'
            hint='Read from server-side spans when correlating inbound service I/O. Use low-cardinality attributes.'
            values={draft.inputSpanAttributes}
            onChange={(inputSpanAttributes) => onChange({ ...draft, inputSpanAttributes })}
          />

          <AttributeListEditor
            label='Outbound span attributes'
            hint='Read from client-side spans when correlating outbound service I/O. Use low-cardinality attributes.'
            values={draft.outputSpanAttributes}
            onChange={(outputSpanAttributes) => onChange({ ...draft, outputSpanAttributes })}
          />

          <FieldGroup>
            <FieldLabel>
              Metrics flush interval
              <FieldHint>How often the serviceio connector flushes correlation metrics (for example 60s or 1m).</FieldHint>
            </FieldLabel>
            <TextInput
              value={draft.metricsFlushInterval}
              onChange={(event) => onChange({ ...draft, metricsFlushInterval: event.target.value })}
              placeholder='60s'
            />
            {flushIntervalInvalid ? <ErrorText>Enter a valid duration such as 60s, 1m, or 15s.</ErrorText> : null}
          </FieldGroup>

          <ActionsRow>
            <ActionButton type='button' onClick={onReset} disabled={!isDirty || saving}>
              Reset
            </ActionButton>
            <ActionButton
              type='button'
              $primary
              onClick={() => void onSave()}
              disabled={!isDirty || flushIntervalInvalid || saving}
            >
              {saving ? 'Saving…' : 'Save settings'}
            </ActionButton>
          </ActionsRow>
        </PanelBody>
      ) : null}
    </Panel>
  );
}
