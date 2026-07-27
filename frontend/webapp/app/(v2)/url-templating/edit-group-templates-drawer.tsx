'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, TrashIcon } from '@odigos/ui-kit/icons';
import type { Action } from '@odigos/ui-kit/types';
import { StatusType } from '@odigos/ui-kit/types';
import {
  Badge,
  Button,
  ButtonSize,
  ButtonVariants,
  Drawer,
  FlexColumn,
  FlexRow,
  IconButton,
  IconButtonSize,
  Input,
  Note,
  Tooltip,
  Typography,
  TypographyColor,
  TypographySize,
} from '@odigos/ui-kit/components';
import { useApiMutation } from '@odigos/ui-kit/contexts';
import { ActionOriginBadge, actionOriginTitleBadge } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
import { TemplatePath } from './template-path';
import { TemplatePathSimulator } from './template-path-simulator';
import { buildUpdateActionWithGroupTemplates } from './action-group-templates-update';
import { sortUrlTemplates } from './template-sort';
import type { CustomTemplateCrGroup } from './url-templatization-aggregate';
import { DrawerScrollEdgeHint, useCanScrollDown } from './drawer-scroll-edge-hint';

const DrawerBody = styled(FlexColumn)`
  width: 100%;
  padding: 0 24px 8px;
  gap: 16px;
`;

const MetaPanel = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 8px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const TemplatesPanel = styled(FlexColumn)`
  width: 100%;
  gap: 2px;
  padding: 12px 14px;
  border-radius: 8px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const DetailBlock = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
`;

const TemplateRow = styled(FlexRow)`
  width: 100%;
  align-items: center;
  gap: 8px;
  min-width: 0;
`;

const TemplateRowMain = styled(FlexRow)<{ $selectable?: boolean }>`
  flex: 1;
  min-width: 0;
  align-items: center;
  cursor: ${({ $selectable }) => ($selectable ? 'pointer' : 'default')};
`;

const TemplateEntry = styled(FlexColumn)<{ $selected?: boolean }>`
  width: 100%;
  gap: 0;
  padding: ${({ $selected }) => ($selected ? '6px 8px 8px' : '2px 8px')};
  margin: 0 -8px;
  border-radius: 8px;
  background: ${({ theme, $selected }) => ($selected ? theme.v2.colors.silver[800] : 'transparent')};
  border: 1px solid
    ${({ theme, $selected }) => ($selected ? theme.v2.colors.blue[400] : 'transparent')};
  box-shadow: ${({ theme, $selected }) =>
    $selected ? `inset 0 1px 0 color-mix(in srgb, ${theme.v2.colors.white[500]} 8%, transparent)` : 'none'};
  transition: background 0.12s ease, border-color 0.12s ease;
`;

const SelectedSimulatorRegion = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
  margin-top: 10px;
  padding: 10px 10px 8px;
  border-radius: 6px;
  background: color-mix(
    in srgb,
    ${({ theme }) => theme.v2.colors.black[500]} 40%,
    ${({ theme }) => theme.v2.colors.silver[800]}
  );
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const DrawerSaveFooter = styled(FlexColumn)`
  width: 100%;
  gap: 12px;
  padding: 16px 24px;
  box-sizing: border-box;
  border-top: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
  background: ${({ theme }) => theme.v2.colors.black[500]};
`;

const DrawerSaveFooterActions = styled(FlexRow)`
  width: 100%;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
`;

type DraftTemplateRow = {
  key: string;
  value: string;
  isExisting: boolean;
};

let draftRowKeyCounter = 0;

function nextDraftRowKey(): string {
  draftRowKeyCounter += 1;
  return `draft-row-${draftRowKeyCounter}`;
}

function draftRowsFromTemplates(templates: string[]): DraftTemplateRow[] {
  return templates.map((value) => ({
    key: nextDraftRowKey(),
    value,
    isExisting: true,
  }));
}

function withTrailingEmptyRow(rows: DraftTemplateRow[]): DraftTemplateRow[] {
  const copy = [...rows];
  let trailingEmpty: DraftTemplateRow | null = null;
  while (copy.length > 0) {
    const last = copy[copy.length - 1];
    if (!last.isExisting && !last.value.trim()) {
      trailingEmpty = copy.pop() ?? null;
    } else {
      break;
    }
  }
  // Keep row keys/order stable while typing so focus is not stolen by a remounted
  // trailing empty input. Templates are sorted on save / initial load.
  return [...copy, trailingEmpty ?? { key: nextDraftRowKey(), value: '', isExisting: false }];
}

function draftRowValues(rows: DraftTemplateRow[]): string[] {
  return rows.map((row) => row.value);
}

function normalizedTemplateValues(rows: DraftTemplateRow[]): string[] {
  return draftRowValues(rows)
    .map((t) => t.trim())
    .filter(Boolean);
}

function isTrailingEmptyInputRow(rows: DraftTemplateRow[], index: number): boolean {
  const row = rows[index];
  if (!row || row.isExisting || row.value.trim()) {
    return false;
  }
  return index === rows.length - 1;
}

type EditGroupTemplatesDrawerProps = {
  group: CustomTemplateCrGroup | null;
  action: Action | null;
  focusNewTemplate?: boolean;
  onClose: () => void;
  onSaved: () => void;
};

function templatesEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  return a.every((value, index) => value === b[index]);
}

function templateChangeCounts(
  initial: string[],
  draft: string[],
): {
  added: number;
  removed: number;
} {
  const initialSet = new Set(initial);
  const draftSet = new Set(draft);
  let added = 0;
  for (const template of draft) {
    if (!initialSet.has(template)) {
      added += 1;
    }
  }
  let removed = 0;
  for (const template of initial) {
    if (!draftSet.has(template)) {
      removed += 1;
    }
  }
  return { added, removed };
}

function buildSaveWarningMessage(added: number, removed: number): string {
  const changes: string[] = [];
  if (added > 0) {
    changes.push(added === 1 ? 'add 1 template' : `add ${added} templates`);
  }
  if (removed > 0) {
    changes.push(removed === 1 ? 'remove 1 template' : `remove ${removed} templates`);
  }
  const summary = changes.join(' and ');
  return `On save: ${summary} — takes effect in the cluster immediately.`;
}

export function EditGroupTemplatesDrawer({
  group,
  action,
  focusNewTemplate = false,
  onClose,
  onSaved,
}: EditGroupTemplatesDrawerProps) {
  const [draftTemplates, setDraftTemplates] = useState<DraftTemplateRow[]>([]);
  const [selectedRowKey, setSelectedRowKey] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [updateAction, { loading: saving }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  useEffect(() => {
    if (!group) {
      setDraftTemplates([]);
      setSelectedRowKey(null);
      setSaveError(null);
      return;
    }
    const rows = withTrailingEmptyRow(
      group.templates.length ? draftRowsFromTemplates(sortUrlTemplates(group.templates)) : [],
    );
    setDraftTemplates(rows);
    setSelectedRowKey(focusNewTemplate ? (rows[rows.length - 1]?.key ?? null) : null);
    setSaveError(null);
  }, [group, focusNewTemplate]);

  useEffect(() => {
    if (!group || !focusNewTemplate) {
      return undefined;
    }
    const timer = window.setTimeout(() => {
      const input = document.querySelector(
        'input[name="url-templating-new-template"]',
      ) as HTMLInputElement | null;
      if (!input) {
        return;
      }
      input.focus();
      input.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }, 120);
    return () => window.clearTimeout(timer);
  }, [group, focusNewTemplate]);

  const initialTemplates = useMemo(() => {
    if (!group) {
      return [];
    }
    return group.templates.length ? sortUrlTemplates(group.templates) : [];
  }, [group]);

  const dirty = useMemo(() => {
    const normalizedDraft = sortUrlTemplates(normalizedTemplateValues(draftTemplates));
    return !templatesEqual(normalizedDraft, initialTemplates);
  }, [draftTemplates, initialTemplates]);

  const validationError = useMemo(() => {
    const normalized = normalizedTemplateValues(draftTemplates);
    const seen = new Set<string>();
    for (const t of normalized) {
      if (seen.has(t)) {
        return 'Each template must be unique within this rule group.';
      }
      seen.add(t);
    }
    return null;
  }, [draftTemplates]);

  const canSave = dirty && !validationError && !saving && !!action && !!group;

  const saveWarningMessage = useMemo(() => {
    if (!dirty) {
      return undefined;
    }
    const draft = sortUrlTemplates(normalizedTemplateValues(draftTemplates));
    const { added, removed } = templateChangeCounts(initialTemplates, draft);
    return buildSaveWarningMessage(added, removed);
  }, [dirty, draftTemplates, initialTemplates]);

  const footerNotice = useMemo(() => {
    if (saveError) {
      return {
        status: StatusType.Error,
        message: saveError,
        fullWidth: true as const,
      };
    }
    if (dirty && validationError) {
      return {
        status: StatusType.Warning,
        message: validationError,
        fullWidth: true as const,
      };
    }
    if (saveWarningMessage) {
      return {
        status: StatusType.Warning,
        message: saveWarningMessage,
        fullWidth: true as const,
      };
    }
    return undefined;
  }, [dirty, saveError, saveWarningMessage, validationError]);

  const updateRow = useCallback((index: number, value: string) => {
    setDraftTemplates((rows) => withTrailingEmptyRow(rows.map((row, i) => (i === index ? { ...row, value } : row))));
  }, []);

  const removeRow = useCallback((index: number, rowKey: string) => {
    setSelectedRowKey((current) => (current === rowKey ? null : current));
    setDraftTemplates((rows) => withTrailingEmptyRow(rows.filter((_, i) => i !== index)));
  }, []);

  const handleSave = async () => {
    if (!group || !action || !canSave) {
      return;
    }
    setSaveError(null);
    try {
      const variables = buildUpdateActionWithGroupTemplates(
        action,
        group.groupIndex,
        normalizedTemplateValues(draftTemplates),
      );
      const { error } = await updateAction(variables);
      if (error) {
        setSaveError(error.message || 'Failed to update templates');
        return;
      }
      onSaved();
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to update templates');
    }
  };

  const headerBadges = group
    ? [
        actionOriginTitleBadge(group.actionUiGenerated),
        ...(group.actionDisabled ? [{ label: 'Disabled' as const }] : []),
      ]
    : undefined;

  const scrollAnchorRef = useRef<HTMLDivElement>(null);
  const scrollRefreshKey = useMemo(
    () =>
      JSON.stringify({
        templates: draftTemplates.map((row) => row.value),
        selectedRowKey,
        saveWarningMessage,
        footerNotice,
        saveError,
      }),
    [draftTemplates, footerNotice, saveError, saveWarningMessage, selectedRowKey, validationError],
  );
  const canScrollDown = useCanScrollDown(scrollAnchorRef, scrollRefreshKey);

  return (
    <Drawer
      isOpen={!!group}
      width="min(720px, 96vw)"
      header={
        group
          ? {
              icon: ActionIcon,
              title: 'Edit rule group templates',
              subTitle: group.actionLabel,
              titleBadges: headerBadges,
              onClose,
            }
          : undefined
      }
      footer={
        group
          ? {
              children: (
                <DrawerSaveFooter $gap={12}>
                  {footerNotice ? <Note {...footerNotice} /> : null}
                  <DrawerSaveFooterActions $gap={12}>
                    <Button
                      label="Cancel"
                      variant={ButtonVariants.Secondary}
                      size={ButtonSize.S}
                      onClick={onClose}
                      disabled={saving}
                    />
                    <Button
                      label="Save"
                      variant={ButtonVariants.Primary}
                      size={ButtonSize.S}
                      onClick={() => {
                        void handleSave();
                      }}
                      disabled={!canSave}
                      loading={saving}
                    />
                  </DrawerSaveFooterActions>
                </DrawerSaveFooter>
              ),
            }
          : undefined
      }
    >
      {group ? (
        <DrawerBody $gap={16}>
          <MetaPanel $gap={12}>
            <DetailBlock $gap={8}>
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                Action
              </Typography>
              <FlexRow $gap={8} $alignItems="center" style={{ flexWrap: 'wrap' }}>
                <Typography size={TypographySize.XS}>
                  {group.actionDisplayName?.trim() || group.actionLabel}
                </Typography>
                <ActionOriginBadge uiGenerated={group.actionUiGenerated} />
                {group.actionDisabled ? <Badge label="Disabled" /> : null}
              </FlexRow>
            </DetailBlock>

            <DetailBlock $gap={8}>
              <Tooltip text="Scope cannot be changed after a rule group is created." inline>
                <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                  Scope
                </Typography>
              </Tooltip>
              <ScopeTokens tokens={group.scopeTokens} />
              <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                Group {group.groupIndex + 1}
                {group.groupNotes?.trim() ? ` · ${group.groupNotes.trim()}` : ''}
              </Typography>
            </DetailBlock>
          </MetaPanel>

          <TemplatesPanel $gap={2}>
            <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
              Templates
            </Typography>

            {draftTemplates.map((row, index) => {
              const trailingEmpty = isTrailingEmptyInputRow(draftTemplates, index);
              const selected = selectedRowKey === row.key;
              const templateForSimulator = row.value.trim();

              return (
                <TemplateEntry key={row.key} $gap={0} $selected={selected && !!templateForSimulator}>
                  <TemplateRow $gap={8}>
                    <TemplateRowMain
                      $gap={8}
                      $selectable={!!templateForSimulator}
                      onClick={
                        templateForSimulator
                          ? () => setSelectedRowKey(row.key)
                          : undefined
                      }
                    >
                      {row.isExisting ? (
                        <TemplatePath template={row.value} disabled={group.actionDisabled} />
                      ) : (
                        <Input
                          data-id={`url-templating-group-template-${index}`}
                          name={trailingEmpty ? 'url-templating-new-template' : `url-templating-group-template-${index}`}
                          value={row.value}
                          onChange={(event) => updateRow(index, event.target.value)}
                          onFocus={() => setSelectedRowKey(row.key)}
                          placeholder="/api/users/{id}"
                          width="100%"
                          disabled={saving}
                        />
                      )}
                    </TemplateRowMain>
                    {trailingEmpty ? null : (
                      <IconButton
                        data-id={`url-templating-group-template-remove-${index}`}
                        icon={TrashIcon}
                        size={IconButtonSize.XS}
                        onClick={() => removeRow(index, row.key)}
                        disabled={saving}
                      />
                    )}
                  </TemplateRow>
                  {selected && templateForSimulator ? (
                    <SelectedSimulatorRegion $gap={8}>
                      <TemplatePathSimulator
                        template={templateForSimulator}
                        actionDisabled={group.actionDisabled}
                        embedded
                      />
                    </SelectedSimulatorRegion>
                  ) : null}
                </TemplateEntry>
              );
            })}
          </TemplatesPanel>

          <div ref={scrollAnchorRef} aria-hidden style={{ width: '100%', height: 1 }} />
          <DrawerScrollEdgeHint visible={canScrollDown} />
        </DrawerBody>
      ) : null}
    </Drawer>
  );
}
