'use client';

import React from 'react';
import { Settings } from '@odigos/ui-kit/containers/v2';
import { useConfigYamls, useEffectiveConfig, useUpdateLocalUiConfig } from '@/hooks';

export default function Page() {
  const { configYamls, configYamlsLoading } = useConfigYamls();
  const { updateLocalUiConfig, loading: saveLoading } = useUpdateLocalUiConfig();
  const { effectiveConfig, effectiveConfigLoading, refetchEffectiveConfig } = useEffectiveConfig();

  return (
    <Settings
      pageHeightOffset={62}
      minSupportedVersion={1.2}
      configYamls={configYamls}
      configYamlsLoading={configYamlsLoading}
      effectiveConfig={effectiveConfig}
      effectiveConfigLoading={effectiveConfigLoading}
      onSave={async (config) => {
        await updateLocalUiConfig(config);
        await refetchEffectiveConfig();
      }}
      saveLoading={saveLoading}
    />
  );
}
