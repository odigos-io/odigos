'use client';

import React from 'react';
import styled from 'styled-components';
import { useConfig } from '@/hooks';
import { PlatformType } from '@odigos/ui-kit/types';
import { Settings } from '@odigos/ui-kit/containers';
import { FlexColumn, Typography, TypographySize } from '@odigos/ui-kit/components';

const MarketplaceSettings = styled(FlexColumn)`
  flex: 1;
  align-self: stretch;
  min-width: 0;
  min-height: 0;
`;

const LicenseNotice = styled.section`
  padding: 16px 24px;
  border-bottom: 1px solid ${({ theme }) => theme.v2.colors.silver['700']};
`;

const SettingsContent = styled.div`
  display: flex;
  flex: 1;
  min-height: 0;
`;

// Settings is being migrated to consume `useOdigosApi()` directly. Once that
// migration lands, this page becomes a one-line `return <Settings />;`.
export default function Page() {
  const { config } = useConfig();
  const settings = (
    <Settings
      minSupportedVersion={{
        [PlatformType.K8s]: 1.2,
      }}
    />
  );

  if (config?.licenseProvider !== 'aws-marketplace') return settings;

  return (
    <MarketplaceSettings $gap={0}>
      <LicenseNotice data-id='marketplace-license' aria-label='AWS Marketplace license'>
        <Typography size={TypographySize.XS}>License provider: AWS Marketplace</Typography>
        <Typography size={TypographySize.XS}>
          Manage your Enterprise subscription and renewals through your AWS Marketplace agreement.
        </Typography>
      </LicenseNotice>
      <SettingsContent>{settings}</SettingsContent>
    </MarketplaceSettings>
  );
}
