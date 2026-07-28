'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, ChevronRightIcon, TrashIcon } from '@odigos/ui-kit/icons';
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
  Input,
  Note,
  Toggle,
  ToggleSize,
  Typography,
  TypographyColor,
  TypographySize,
  WarningModal,
} from '@odigos/ui-kit/components';
import { SourceScopeSection } from '@odigos/ui-kit/snippets';
import { useApiMutation, useOdigosApi } from '@odigos/ui-kit/contexts';
import { actionOriginTitleBadge } from './action-origin-badge';
import { SectionHelpButton } from './section-help-button';
import { SKIP_POLICY_HELP } from './skip-policy-help';
import {
  buildDefaultGroupFromEditInput,
  buildUpdateActionWithAppendedDefaultGroup,
  buildUpdateActionWithDefaultGroup,
  buildUpdateActionWithDefaultGroupDeleted,
  defaultGroupEditInputsEqual,
  defaultGroupScopesToSourcesScopes,
  formatSkipHttpStatusCodesInput,
  parseSkipHttpStatusCodesInput,
  type DefaultGroupEditInput,
} from './action-default-group-update';
import { makeEmptySourcesScopes } from './action-group-templates-update';
import type { DefaultTemplatizationCrGroup } from './url-templatization-aggregate';

const DrawerBody = styled(FlexColumn)`
  width: 100%;
  padding: 0 24px 24px;
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

const DetailBlock = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
`;

const CollapseHeader = styled(FlexRow)`
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 2px 0;
  cursor: pointer;
  user-select: none;

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.v2.colors.blue[400]};
    outline-offset: 2px;
    border-radius: 4px;
  }
`;

const CollapseChevron = styled.span<{ $open: boolean }>`
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  transform: rotate(${({ $open }) => ($open ? '90deg' : '0deg')});
  transition: transform 0.15s ease;
  color: ${({ theme }) => theme.v2.colors.grey[400]};
`;

const CollapseBody = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
  padding-top: 8px;
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

type EditDefaultTemplatizationDrawerProps = {
  group: DefaultTemplatizationCrGroup | null;
  action: Action | null;
  /** Open the drawer to create a new default rule on `action` (group is ignored). */
  creating?: boolean;
  onClose: () => void;
  onSaved: () => void;
};

const EMPTY_CREATE_INPUT: DefaultGroupEditInput = {
  scopes: { sources: [], namespaces: [], languages: [] },
  disabled: false,
  skipForNonSuccessCodes: false,
  skipHttpStatusCodes: [],
};

function editInputFromGroup(group: DefaultTemplatizationCrGroup): DefaultGroupEditInput {
  return {
    scopes: defaultGroupScopesToSourcesScopes(group.scopes),
    disabled: group.disabled,
    skipForNonSuccessCodes: group.skipForNonSuccessCodes,
    skipHttpStatusCodes: [...group.skipHttpStatusCodes],
  };
}

export function EditDefaultTemplatizationDrawer({
  group,
  action,
  creating = false,
  onClose,
  onSaved,
}: EditDefaultTemplatizationDrawerProps) {
  const isCreate = creating && !!action;
  const isOpen = isCreate || !!group;

  const { sourcesApi } = useOdigosApi();
  const { items: sources = [] } = sourcesApi.useSources();

  const [scopes, setScopes] = useState<SourcesScopes>(makeEmptySourcesScopes);
  const [disabled, setDisabled] = useState(false);
  const [skipForNonSuccessCodes, setSkipForNonSuccessCodes] = useState(false);
  const [skipHttpStatusCodesText, setSkipHttpStatusCodesText] = useState('');
  const [skipPolicyOpen, setSkipPolicyOpen] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [updateAction, { loading: saving }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  useEffect(() => {
    if (isCreate) {
      setScopes(makeEmptySourcesScopes());
      setDisabled(false);
      setSkipForNonSuccessCodes(false);
      setSkipHttpStatusCodesText('');
      setSkipPolicyOpen(false);
      setSaveError(null);
      setConfirmDeleteOpen(false);
      setDeleteError(null);
      return;
    }
    if (!group) {
      setScopes(makeEmptySourcesScopes());
      setDisabled(false);
      setSkipForNonSuccessCodes(false);
      setSkipHttpStatusCodesText('');
      setSkipPolicyOpen(false);
      setSaveError(null);
      setConfirmDeleteOpen(false);
      setDeleteError(null);
      return;
    }
    const initial = editInputFromGroup(group);
    setScopes(initial.scopes);
    setDisabled(initial.disabled);
    setSkipForNonSuccessCodes(initial.skipForNonSuccessCodes);
    setSkipHttpStatusCodesText(formatSkipHttpStatusCodesInput(initial.skipHttpStatusCodes));
    setSkipPolicyOpen(false);
    setSaveError(null);
    setConfirmDeleteOpen(false);
    setDeleteError(null);
  }, [group, isCreate]);

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

  const parsedSkipCodes = useMemo(
    () => parseSkipHttpStatusCodesInput(skipHttpStatusCodesText),
    [skipHttpStatusCodesText],
  );

  const draftInput = useMemo<DefaultGroupEditInput>(
    () => ({
      scopes,
      disabled,
      skipForNonSuccessCodes,
      skipHttpStatusCodes: parsedSkipCodes.ok ? parsedSkipCodes.codes : [],
    }),
    [disabled, parsedSkipCodes, scopes, skipForNonSuccessCodes],
  );

  const initialInput = useMemo(() => {
    if (isCreate) {
      return EMPTY_CREATE_INPUT;
    }
    return group ? editInputFromGroup(group) : null;
  }, [group, isCreate]);

  const dirty = useMemo(() => {
    if (!initialInput) {
      return false;
    }
    if (!parsedSkipCodes.ok && skipHttpStatusCodesText.trim()) {
      return true;
    }
    return !defaultGroupEditInputsEqual(draftInput, initialInput);
  }, [draftInput, initialInput, parsedSkipCodes.ok, skipHttpStatusCodesText]);

  const validationError = useMemo(() => {
    if (!skipForNonSuccessCodes && !parsedSkipCodes.ok) {
      return parsedSkipCodes.error;
    }
    return null;
  }, [parsedSkipCodes, skipForNonSuccessCodes]);

  const canSave =
    !validationError && !saving && !!action && (isCreate || (!!group && dirty));

  const footerNotice = useMemo(() => {
    if (saveError) {
      return { status: StatusType.Error, message: saveError, fullWidth: true as const };
    }
    if (validationError) {
      return { status: StatusType.Warning, message: validationError, fullWidth: true as const };
    }
    if (isCreate || dirty) {
      return {
        status: StatusType.Warning,
        message: isCreate
          ? 'On create: add this default rule — takes effect in the cluster immediately.'
          : 'On save: update this default rule — takes effect in the cluster immediately.',
        fullWidth: true as const,
      };
    }
    return undefined;
  }, [dirty, isCreate, saveError, validationError]);

  const headerBadges = group
    ? [
        actionOriginTitleBadge(group.actionUiGenerated),
        ...(group.actionDisabled ? [{ label: 'Disabled' as const }] : []),
      ]
    : action
      ? [actionOriginTitleBadge(!!(action as Action & { uiGenerated?: boolean }).uiGenerated)]
      : undefined;

  const handleSave = async () => {
    if (!action || !canSave) {
      return;
    }
    setSaveError(null);
    try {
      const variables = isCreate
        ? buildUpdateActionWithAppendedDefaultGroup(action, buildDefaultGroupFromEditInput(draftInput))
        : buildUpdateActionWithDefaultGroup(action, group!.groupIndex, draftInput);
      const { error } = await updateAction(variables);
      if (error) {
        setSaveError(error.message || (isCreate ? 'Failed to create default rule' : 'Failed to update default rule'));
        return;
      }
      onSaved();
      onClose();
    } catch (err) {
      setSaveError(
        err instanceof Error
          ? err.message
          : isCreate
            ? 'Failed to create default rule'
            : 'Failed to update default rule',
      );
    }
  };

  const handleDelete = async () => {
    if (!group || !action || isCreate) {
      return;
    }
    setDeleteError(null);
    try {
      const variables = buildUpdateActionWithDefaultGroupDeleted(action, group.groupIndex);
      const { error } = await updateAction(variables);
      if (error) {
        setDeleteError(error.message || 'Failed to delete default rule');
        return;
      }
      setConfirmDeleteOpen(false);
      onSaved();
      onClose();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete default rule');
    }
  };

  const actionLabel = group?.actionLabel || action?.name?.trim() || action?.id || '';

  return (
    <>
      <Drawer
        isOpen={isOpen}
        width="min(720px, 96vw)"
        header={
          isOpen
            ? {
                icon: ActionIcon,
                title: isCreate ? 'Create default templatization' : 'Edit default templatization',
                subTitle: actionLabel,
                titleBadges: headerBadges,
                actions:
                  !isCreate && action
                    ? [
                        {
                          id: 'delete-default-rule',
                          label: 'Delete rule',
                          icon: TrashIcon,
                          onClick: () => {
                            setDeleteError(null);
                            setConfirmDeleteOpen(true);
                          },
                        },
                      ]
                    : undefined,
                onClose,
              }
            : undefined
        }
        footer={
          isOpen
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
                        label={isCreate ? 'Create' : 'Save'}
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
        {isOpen ? (
          <DrawerBody $gap={16}>
            <MetaPanel $gap={12}>
              <SourceScopeSection
                scopes={scopes}
                onChange={setScopes}
                sourceOptions={sourceOptions}
                namespaceOptions={namespaceOptions}
                dataIdPrefix={isCreate ? 'url-templating-create-default-scope' : 'url-templating-edit-default-scope'}
                slim
                disabled={saving}
              />
            </MetaPanel>

            <MetaPanel $gap={12}>
              <DetailBlock $gap={8}>
                <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                  Default templating
                </Typography>
                <Toggle
                  name="url-templating-edit-default-enabled"
                  size={ToggleSize.S}
                  label="Enable default templating for this scope"
                  tooltip="Applies heuristic templates when no custom rule matches."
                  value={!disabled}
                  onChange={(enabled) => setDisabled(!enabled)}
                  disabled={saving}
                />
              </DetailBlock>
            </MetaPanel>

            <MetaPanel $gap={12}>
              <DetailBlock $gap={0}>
                <CollapseHeader
                  $gap={12}
                  role="button"
                  tabIndex={0}
                  aria-expanded={skipPolicyOpen}
                  onClick={() => setSkipPolicyOpen((value) => !value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      setSkipPolicyOpen((value) => !value);
                    }
                  }}
                >
                  <FlexRow $gap={6} $alignItems="center" style={{ flex: 1, minWidth: 0 }}>
                    <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                      Skip policy
                    </Typography>
                    <Badge label="Advanced" />
                    <span
                      onClick={(event) => event.stopPropagation()}
                      onKeyDown={(event) => event.stopPropagation()}
                    >
                      <SectionHelpButton
                        data-id="url-templating-edit-default-skip-policy-help"
                        title={SKIP_POLICY_HELP.title}
                        text={SKIP_POLICY_HELP.text}
                      />
                    </span>
                  </FlexRow>
                  <CollapseChevron $open={skipPolicyOpen}>
                    <ChevronRightIcon size={14} />
                  </CollapseChevron>
                </CollapseHeader>
                {skipPolicyOpen ? (
                  <CollapseBody $gap={8}>
                    <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                      Skip heuristic default templating on selected error responses (server spans). Custom
                      templates still match first.
                    </Typography>
                    <Toggle
                      name="url-templating-edit-default-skip-non-2xx"
                      size={ToggleSize.S}
                      label="Skip for non-2xx responses"
                      tooltip="When on, status-code list below is ignored."
                      value={skipForNonSuccessCodes}
                      onChange={(value) => {
                        setSkipForNonSuccessCodes(value);
                        if (value) {
                          setSkipHttpStatusCodesText('');
                        }
                      }}
                      disabled={saving || disabled}
                    />
                    <Input
                      data-id="url-templating-edit-default-skip-codes"
                      name="url-templating-edit-default-skip-codes"
                      label="Skip specific HTTP status codes"
                      placeholder="404, 401"
                      value={skipHttpStatusCodesText}
                      onChange={(event) => setSkipHttpStatusCodesText(event.target.value)}
                      width="100%"
                      disabled={saving || disabled || skipForNonSuccessCodes}
                    />
                  </CollapseBody>
                ) : null}
              </DetailBlock>
            </MetaPanel>
          </DrawerBody>
        ) : null}
      </Drawer>

      <WarningModal
        isOpen={confirmDeleteOpen}
        title="Delete this default rule?"
        description="This default templatization rule will be removed from the action and take effect in the cluster immediately."
        onClose={() => {
          if (!saving) {
            setConfirmDeleteOpen(false);
            setDeleteError(null);
          }
        }}
        denyButton={{
          label: 'Go Back',
          onClick: () => {
            setConfirmDeleteOpen(false);
            setDeleteError(null);
          },
          disabled: saving,
        }}
        approveButton={{
          label: 'Delete Rule',
          onClick: () => {
            void handleDelete();
          },
          loading: saving,
          disabled: saving,
        }}
      >
        {deleteError ? <Note status={StatusType.Error} message={deleteError} fullWidth /> : null}
      </WarningModal>
    </>
  );
}
