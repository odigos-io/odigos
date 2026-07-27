'use client';

import React from 'react';
import { Badge } from '@odigos/ui-kit/components';
import { StatusType } from '@odigos/ui-kit/types';

export function isActionUiGenerated(action: unknown): boolean {
  if (!action || typeof action !== 'object') {
    return false;
  }
  return !!(action as { uiGenerated?: boolean | null }).uiGenerated;
}

export function actionOriginLabel(uiGenerated: boolean): string {
  return uiGenerated ? 'UI' : 'External';
}

export function actionOriginTooltip(uiGenerated: boolean): string {
  return uiGenerated
    ? 'Created from the Odigos UI'
    : 'Created outside the UI (GitOps or manual, for example YAML, Helm, or kubectl)';
}

export function actionOriginTitleBadge(uiGenerated: boolean): { label: string } {
  return { label: actionOriginLabel(uiGenerated) };
}

export function ActionOriginBadge({ uiGenerated }: { uiGenerated: boolean }) {
  return (
    <Badge
      label={actionOriginLabel(uiGenerated)}
      tooltip={actionOriginTooltip(uiGenerated)}
      status={uiGenerated ? StatusType.Info : StatusType.Default}
      useSecondaryTone={!uiGenerated}
    />
  );
}
