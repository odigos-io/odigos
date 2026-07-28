'use client';

import React, { useEffect, useMemo, useState } from 'react';
import styled from 'styled-components';
import { ActionIcon, EditIcon, TrashIcon } from '@odigos/ui-kit/icons';
import type { Action } from '@odigos/ui-kit/types';
import { StatusType } from '@odigos/ui-kit/types';
import {
  Badge,
  Button,
  ButtonSize,
  ButtonVariants,
  Divider,
  Drawer,
  FlexColumn,
  FlexRow,
  Note,
  Typography,
  TypographyColor,
  TypographySize,
  WarningModal,
} from '@odigos/ui-kit/components';
import { useApiMutation } from '@odigos/ui-kit/contexts';
import { ActionOriginBadge, actionOriginTitleBadge } from './action-origin-badge';
import { ScopeTokens } from './scope-tokens';
import { SectionHelpButton } from './section-help-button';
import { SKIP_POLICY_HELP } from './skip-policy-help';
import { buildUpdateActionWithDefaultGroupDeleted } from './action-default-group-update';
import type { DefaultTemplatizationCrGroup } from './url-templatization-aggregate';

const DrawerBody = styled(FlexColumn)`
  width: 100%;
  padding: 0 24px 24px;
  gap: 16px;
`;

const DetailBlock = styled(FlexColumn)`
  width: 100%;
  gap: 8px;
`;

const FieldList = styled(FlexColumn)`
  width: 100%;
  gap: 10px;
`;

const Field = styled(FlexColumn)`
  width: 100%;
  gap: 2px;
`;

const ResourceName = styled.span`
  font-family: ${({ theme }) => theme.font_family.code ?? theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xs}px;
  line-height: 1.4;
  color: ${({ theme }) => theme.v2.colors.white[500]};
  word-break: break-word;
`;

function DetailField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <Field $gap={2}>
      <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
        {label}
      </Typography>
      {children}
    </Field>
  );
}

type DefaultTemplatizationDetailDrawerProps = {
  group: DefaultTemplatizationCrGroup | null;
  action?: Action | null;
  onEdit?: (group: DefaultTemplatizationCrGroup) => void;
  onDeleted?: () => void;
  onClose: () => void;
};

export function DefaultTemplatizationDetailDrawer({
  group,
  action = null,
  onEdit,
  onDeleted,
  onClose,
}: DefaultTemplatizationDetailDrawerProps) {
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [updateAction, { loading: deleting }] = useApiMutation('UPDATE_ACTION', {
    refetchQueries: ['GetActions'],
    awaitRefetchQueries: true,
  });

  useEffect(() => {
    setConfirmDeleteOpen(false);
    setDeleteError(null);
  }, [group]);

  const headerBadges = useMemo(() => {
    if (!group) {
      return undefined;
    }
    const badges = [actionOriginTitleBadge(group.actionUiGenerated)];
    if (group.actionDisabled) {
      badges.push({ label: 'Action disabled' });
    }
    if (group.disabled) {
      badges.push({ label: 'Default Templating Disabled' });
    } else {
      badges.push({ label: 'Default Templating Enabled' });
    }
    return badges;
  }, [group]);

  const canEdit = !!action && !!onEdit;
  const canDelete = !!action && !!onDeleted;

  const handleDelete = async () => {
    if (!group || !action || !onDeleted) {
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
      onDeleted();
      onClose();
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete default rule');
    }
  };

  const headerActions = useMemo(() => {
    if (!group) {
      return undefined;
    }
    const actions = [];
    if (canEdit) {
      actions.push({
        id: 'edit-default-rule',
        label: 'Edit rule',
        icon: EditIcon,
        onClick: () => onEdit?.(group),
      });
    }
    if (canDelete) {
      actions.push({
        id: 'delete-default-rule',
        label: 'Delete rule',
        icon: TrashIcon,
        onClick: () => {
          setDeleteError(null);
          setConfirmDeleteOpen(true);
        },
      });
    }
    return actions.length ? actions : undefined;
  }, [canDelete, canEdit, group, onEdit]);

  return (
    <>
      <Drawer
        isOpen={!!group}
        width="min(720px, 96vw)"
        header={
          group
            ? {
                icon: ActionIcon,
                title: 'Default templatization',
                subTitle: group.actionLabel,
                titleBadges: headerBadges,
                actions: headerActions,
                onClose,
              }
            : undefined
        }
      >
        {group ? (
          <DrawerBody $gap={16}>
            <DetailBlock $gap={8}>
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                Scope
              </Typography>
              <ScopeTokens tokens={group.scopeTokens} />
              <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                Group {group.groupIndex + 1}
              </Typography>
            </DetailBlock>

            <Divider />

            <DetailBlock $gap={8}>
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                Status
              </Typography>
              {group.disabled ? <Badge label="Default Templating Disabled" /> : <Badge label="Default Templating Enabled" />}
              <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                {group.disabled
                  ? 'Heuristic default templating is turned off for this scope. Custom templates still apply when they match.'
                  : 'Spans try custom templates first; if none match, heuristic default templating is applied for this scope.'}
              </Typography>
            </DetailBlock>

            <Divider />

            <DetailBlock $gap={8}>
              <FlexRow $gap={6} $alignItems="center">
                <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                  Skip policy
                </Typography>
                <SectionHelpButton
                  data-id="url-templating-default-skip-policy-help"
                  title={SKIP_POLICY_HELP.title}
                  text={SKIP_POLICY_HELP.text}
                />
              </FlexRow>
              {group.skipForNonSuccessCodes || group.skipHttpStatusCodes.length > 0 ? (
                <>
                  <FlexRow $gap={6} $alignItems="center" style={{ flexWrap: 'wrap' }}>
                    {group.skipForNonSuccessCodes ? <Badge label="Skip non-2xx" /> : null}
                    {group.skipHttpStatusCodes.length > 0 ? (
                      <Badge label={`Skip HTTP ${group.skipHttpStatusCodes.join(', ')}`} />
                    ) : null}
                  </FlexRow>
                  <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                    {group.skipForNonSuccessCodes
                      ? 'Heuristic default templating is skipped for non-2xx HTTP responses on server spans.'
                      : `Heuristic default templating is skipped for HTTP status codes: ${group.skipHttpStatusCodes.join(', ')}.`}
                  </Typography>
                </>
              ) : (
                <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
                  No skip policy configured.
                </Typography>
              )}
            </DetailBlock>

            <Divider />

            <DetailBlock $gap={8}>
              <Typography size={TypographySize.XXS} color={TypographyColor.Secondary}>
                Action
              </Typography>
              <FieldList $gap={10}>
                <DetailField label="Display name">
                  <Typography size={TypographySize.XS}>{group.actionLabel}</Typography>
                </DetailField>
                <DetailField label="Resource name">
                  <ResourceName>{group.actionId}</ResourceName>
                </DetailField>
                <DetailField label="Origin">
                  <ActionOriginBadge uiGenerated={group.actionUiGenerated} />
                </DetailField>
                {group.actionDisabled ? (
                  <DetailField label="Action status">
                    <Badge label="Disabled" />
                  </DetailField>
                ) : null}
              </FieldList>
            </DetailBlock>

            {canEdit ? (
              <>
                <Divider />
                <FlexRow $gap={8} style={{ justifyContent: 'flex-end' }}>
                  <Button
                    data-id="url-templating-default-detail-edit"
                    label="Edit rule"
                    variant={ButtonVariants.Primary}
                    size={ButtonSize.S}
                    onClick={() => onEdit?.(group)}
                  />
                </FlexRow>
              </>
            ) : null}
          </DrawerBody>
        ) : null}
      </Drawer>

      <WarningModal
        isOpen={confirmDeleteOpen}
        title="Delete this default rule?"
        description="This default templatization rule will be removed from the action and take effect in the cluster immediately."
        onClose={() => {
          if (!deleting) {
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
          disabled: deleting,
        }}
        approveButton={{
          label: 'Delete Rule',
          onClick: () => {
            void handleDelete();
          },
          loading: deleting,
          disabled: deleting,
        }}
      >
        {deleteError ? <Note status={StatusType.Error} message={deleteError} fullWidth /> : null}
      </WarningModal>
    </>
  );
}
