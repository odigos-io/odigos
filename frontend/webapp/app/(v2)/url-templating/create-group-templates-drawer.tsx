'use client';

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, TrashIcon } from '@odigos/ui-kit/icons';
import type { Action, SourcesScopes } from '@odigos/ui-kit/types';
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
  Typography,
  TypographyColor,
  TypographySize,
} from '@odigos/ui-kit/components';
import { useApiMutation, useOdigosApi } from '@odigos/ui-kit/contexts';
import { SourceScopeSection } from '@odigos/ui-kit/snippets';
import { actionOriginTitleBadge, ActionOriginBadge, isActionUiGenerated } from './action-origin-badge';
import { TemplatePathSimulator } from './template-path-simulator';
import {
  buildUpdateActionWithAppendedGroup,
  makeEmptySourcesScopes,
  sourcesScopesToAppendRuleGroupFilters,
} from './action-group-templates-update';
import { sortUrlTemplates } from './template-sort';
import { DrawerScrollEdgeHint, useCanScrollDown } from './drawer-scroll-edge-hint';
import { ScopeTokens } from './scope-tokens';

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
};

let draftRowKeyCounter = 0;

function nextDraftRowKey(): string {
  draftRowKeyCounter += 1;
  return `create-draft-row-${draftRowKeyCounter}`;
}

function withTrailingEmptyRow(rows: DraftTemplateRow[]): DraftTemplateRow[] {
  const copy = [...rows];
  let trailingEmpty: DraftTemplateRow | null = null;
  while (copy.length > 0 && !copy[copy.length - 1].value.trim()) {
    trailingEmpty = copy.pop() ?? null;
  }
  // Keep row keys/order stable while typing so focus is not stolen by a remounted
  // trailing empty input. Templates are sorted on save.
  return [...copy, trailingEmpty ?? { key: nextDraftRowKey(), value: '' }];
}

function normalizedTemplateValues(rows: DraftTemplateRow[]): string[] {
  return rows
    .map((row) => row.value.trim())
    .filter(Boolean);
}

function isTrailingEmptyInputRow(rows: DraftTemplateRow[], index: number): boolean {
  const row = rows[index];
  if (!row || row.value.trim()) {
    return false;
  }
  return index === rows.length - 1;
}

type CreateGroupTemplatesDrawerProps = {
  action: Action | null;
  /** When true, scope is fixed to entire cluster (empty SourcesScopes). */
  lockEntireClusterScope?: boolean;
  onClose: () => void;
  onSaved: () => void;
};

function buildCreateSaveWarning(templateCount: number): string {
  const templates =
    templateCount === 1 ? 'add 1 template as a new rule group' : `add ${templateCount} templates as a new rule group`;
  return `On save: ${templates} — takes effect in the cluster immediately.`;
}

export function CreateGroupTemplatesDrawer({
  action,
  lockEntireClusterScope = false,
  onClose,
  onSaved,
}: CreateGroupTemplatesDrawerProps) {
  const { sourcesApi } = useOdigosApi();
  const { items: sources = [] } = sourcesApi.useSources();

  const [scopes, setScopes] = useState<SourcesScopes>(makeEmptySourcesScopes);
  const [draftTemplates, setDraftTemplates] = useState<DraftTemplateRow[]>([]);
  const [selectedRowKey, setSelectedRowKey] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [updateAction, { loading: saving }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  const sourceOptions = useMemo(
    () =>
      sources.map(({ id }) => ({
        id: `${id.namespace}/${id.kind}/${id.name}`,
        label: id.name,
        badge: id.kind,
        secondaryLabel: id.namespace,
      })),
    [sources],
  );

  const namespaceOptions = useMemo(
    () =>
      Array.from(new Set(sources.map(({ id }) => id.namespace)))
        .sort()
        .map((namespace) => ({ id: namespace, label: namespace })),
    [sources],
  );

  useEffect(() => {
    if (!action) {
      setScopes(makeEmptySourcesScopes());
      setDraftTemplates([]);
      setSelectedRowKey(null);
      setSaveError(null);
      return;
    }
    const rows = withTrailingEmptyRow([]);
    setScopes(makeEmptySourcesScopes());
    setDraftTemplates(rows);
    setSelectedRowKey(rows[rows.length - 1]?.key ?? null);
    setSaveError(null);
  }, [action, lockEntireClusterScope]);

  useEffect(() => {
    if (!action) {
      return undefined;
    }
    const timer = window.setTimeout(() => {
      const input = document.querySelector(
        'input[name="url-templating-create-new-template"]',
      ) as HTMLInputElement | null;
      if (!input) {
        return;
      }
      input.focus();
      input.scrollIntoView({ block: 'center', behavior: 'smooth' });
    }, 120);
    return () => window.clearTimeout(timer);
  }, [action]);

  const templates = useMemo(() => sortUrlTemplates(normalizedTemplateValues(draftTemplates)), [draftTemplates]);

  const scopeMapping = useMemo(() => sourcesScopesToAppendRuleGroupFilters(scopes), [scopes]);
  const scopeError = scopeMapping.ok ? null : scopeMapping.error;

  const validationError = useMemo(() => {
    if (templates.length === 0) {
      return 'Add at least one template before saving.';
    }
    const seen = new Set<string>();
    for (const t of templates) {
      if (seen.has(t)) {
        return 'Each template must be unique within this rule group.';
      }
      seen.add(t);
    }
    return null;
  }, [templates]);

  const canSave = !scopeError && !validationError && !saving && !!action && templates.length > 0;

  const saveWarningMessage = useMemo(() => {
    if (!templates.length || scopeError || validationError) {
      return undefined;
    }
    return buildCreateSaveWarning(templates.length);
  }, [scopeError, templates.length, validationError]);

  const footerNotice = useMemo(() => {
    if (saveError) {
      return {
        status: StatusType.Error,
        message: saveError,
        fullWidth: true as const,
      };
    }
    if (scopeError) {
      return {
        status: StatusType.Warning,
        message: scopeError,
        fullWidth: true as const,
      };
    }
    if (validationError) {
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
  }, [saveError, saveWarningMessage, scopeError, validationError]);

  const updateRow = useCallback((index: number, value: string) => {
    setDraftTemplates((rows) => withTrailingEmptyRow(rows.map((row, i) => (i === index ? { ...row, value } : row))));
  }, []);

  const removeRow = useCallback((index: number, rowKey: string) => {
    setSelectedRowKey((current) => (current === rowKey ? null : current));
    setDraftTemplates((rows) => withTrailingEmptyRow(rows.filter((_, i) => i !== index)));
  }, []);

  const handleSave = async () => {
    if (!action || !canSave || !scopeMapping.ok) {
      return;
    }
    setSaveError(null);
    try {
      const variables = buildUpdateActionWithAppendedGroup(action, {
        ...scopeMapping.filters,
        templates,
      });
      const { error } = await updateAction(variables);
      if (error) {
        setSaveError(error.message || 'Failed to create rule group');
        return;
      }
      onSaved();
      onClose();
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to create rule group');
    }
  };

  const actionLabel = action?.name?.trim() || action?.id || '';
  const headerBadges = action
    ? [
        actionOriginTitleBadge(isActionUiGenerated(action)),
        ...(action.disabled ? [{ label: 'Disabled' as const }] : []),
      ]
    : undefined;

  const scrollAnchorRef = useRef<HTMLDivElement>(null);
  const scrollRefreshKey = useMemo(
    () =>
      JSON.stringify({
        templates: draftTemplates.map((row) => row.value),
        selectedRowKey,
        scopes,
        footerNotice,
      }),
    [draftTemplates, footerNotice, scopes, selectedRowKey],
  );
  const canScrollDown = useCanScrollDown(scrollAnchorRef, scrollRefreshKey);

  return (
    <Drawer
      isOpen={!!action}
      width="min(720px, 96vw)"
      header={
        action
          ? {
              icon: ActionIcon,
              title: 'Add rule group templates',
              subTitle: actionLabel,
              titleBadges: headerBadges,
              onClose,
            }
          : undefined
      }
      footer={
        action
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
      {action ? (
        <DrawerBody $gap={16}>
          <MetaPanel $gap={12}>
            <DetailBlock $gap={8}>
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                Action
              </Typography>
              <FlexRow $gap={8} $alignItems="center" style={{ flexWrap: 'wrap' }}>
                <Typography size={TypographySize.XS}>{actionLabel}</Typography>
                <ActionOriginBadge uiGenerated={isActionUiGenerated(action)} />
                {action.disabled ? <Badge label="Disabled" /> : null}
              </FlexRow>
            </DetailBlock>

            {lockEntireClusterScope ? (
              <DetailBlock $gap={8}>
                <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                  Scope
                </Typography>
                <ScopeTokens tokens={[{ type: 'entire_cluster' }]} />
                <Note
                  status={StatusType.Warning}
                  message="Entire-cluster scope applies these templates to every span — prefer a narrower group when possible."
                  fullWidth
                  smallIcon
                />
              </DetailBlock>
            ) : (
              <SourceScopeSection
                scopes={scopes}
                onChange={setScopes}
                sourceOptions={sourceOptions}
                namespaceOptions={namespaceOptions}
                dataIdPrefix="url-templating-create-scope"
                slim
                disabled={saving}
              />
            )}
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
                      <Input
                        data-id={`url-templating-create-group-template-${index}`}
                        name={
                          trailingEmpty
                            ? 'url-templating-create-new-template'
                            : `url-templating-create-group-template-${index}`
                        }
                        value={row.value}
                        onChange={(event) => updateRow(index, event.target.value)}
                        onFocus={() => setSelectedRowKey(row.key)}
                        placeholder="/api/users/{id}"
                        width="100%"
                        disabled={saving}
                      />
                    </TemplateRowMain>
                    {trailingEmpty ? null : (
                      <IconButton
                        data-id={`url-templating-create-group-template-remove-${index}`}
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
                        actionDisabled={!!action.disabled}
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
