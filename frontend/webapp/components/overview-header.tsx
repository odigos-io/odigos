import React, { useEffect, useMemo, useState } from 'react';
import styled, { css, keyframes } from 'styled-components';
import { useConfig } from '@/hooks';
import { StatusType } from '@odigos/ui-kit/types';
import { OdigosLogo, TerminalIcon } from '@odigos/ui-kit/icons';
import { useOdigosApi } from '@odigos/ui-kit/contexts';
import { StatusKeys, useStatusStore } from '../store';
import { FORM_ALERTS, RECOMMENDATIONS_DRAWER_TITLE } from '@odigos/ui-kit/constants';
import { OdigosLogoTextByTier } from '@odigos/ui-kit/snippets';
import { countRecommended, RecommendationsDrawer, SystemDrawer } from '@odigos/ui-kit/containers';
import { IconButton, Badge as V2Badge, Header as V2Header, Tooltip, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';

const RECOMMENDATIONS_INTRO_SESSION_KEY = 'odigos.recommendations.button.introSeen';
const RECOMMENDATIONS_INTRO_MS = 5000;

/** Same motion as the overview ColorfulBar. */
const colorfulBorderAnimation = keyframes`
  0% {
    background-position: 200% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
`;

const RecommendationsButton = styled.button<{ $emphasized: boolean; $animating: boolean }>`
  display: flex;
  align-items: center;
  padding: ${({ $emphasized }) => ($emphasized ? '1.5px' : '0')};
  border: ${({ theme, $emphasized }) => ($emphasized ? 'none' : `1px solid ${theme.v2.colors.silver['700']}`)};
  border-radius: 100px;
  background: ${({ theme, $emphasized }) =>
    $emphasized
      ? `linear-gradient(90deg, ${theme.v2.colors.pink['500']} 0%, ${theme.v2.colors.purple['400']} 25%, ${theme.v2.colors.green['500']} 50%, ${theme.v2.colors.purple['400']} 75%, ${theme.v2.colors.pink['500']} 100%)`
      : 'transparent'};
  background-size: 200% 100%;
  animation: ${({ $animating }) => ($animating ? css`${colorfulBorderAnimation} 4s linear infinite` : 'none')};
  cursor: pointer;

  &:hover > span {
    background-color: ${({ theme }) => theme.v2.colors.silver['900']};
  }
`;

const RecommendationsButtonInner = styled.span`
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px 6px 10px;
  border-radius: 100px;
  background-color: ${({ theme }) => theme.v2.colors.black['500']};
  transition: background-color 0.15s ease;
`;

const hasSeenRecommendationsIntro = (): boolean => {
  try {
    return sessionStorage.getItem(RECOMMENDATIONS_INTRO_SESSION_KEY) === '1';
  } catch {
    return false;
  }
};

const markRecommendationsIntroSeen = () => {
  try {
    sessionStorage.setItem(RECOMMENDATIONS_INTRO_SESSION_KEY, '1');
  } catch {
    // ignore unavailable sessionStorage (private mode, etc.)
  }
};

export const OverviewHeader = () => {
  const { isReadonly } = useConfig();
  const { recommendationsApi, capabilities } = useOdigosApi();
  const { items: recommendations } = recommendationsApi.useRecommendations();
  const recommendedCount = countRecommended(recommendations);
  const recommendationsTotal = recommendations.length;
  const hasPendingRecommendations = recommendedCount > 0;

  const [isSystemDrawerOpen, setIsSystemDrawerOpen] = useState(false);
  const [isRecommendationsDrawerOpen, setIsRecommendationsDrawerOpen] = useState(false);
  const [isIntroAnimating, setIsIntroAnimating] = useState(false);
  const toggleSystemDrawer = () => setIsSystemDrawerOpen((prev) => !prev);
  const toggleRecommendationsDrawer = () => setIsRecommendationsDrawerOpen((prev) => !prev);

  useEffect(() => {
    if (!hasPendingRecommendations) {
      setIsIntroAnimating(false);
      return;
    }
    if (hasSeenRecommendationsIntro()) {
      setIsIntroAnimating(false);
      return;
    }

    setIsIntroAnimating(true);
    const timeoutId = window.setTimeout(() => {
      setIsIntroAnimating(false);
      markRecommendationsIntroSeen();
    }, RECOMMENDATIONS_INTRO_MS);

    return () => window.clearTimeout(timeoutId);
  }, [hasPendingRecommendations]);

  const tokenStatus = useStatusStore((state) => state[StatusKeys.Token]);
  const backendStatus = useStatusStore((state) => state[StatusKeys.Backend]);
  const instrumentationStatus = useStatusStore((state) => state[StatusKeys.Instrumentation]);

  const left = useMemo(() => {
    const arr = [
      <div key='logo' data-id='logo'>
        <OdigosLogoTextByTier size={200} />
      </div>,
    ];
    if (isReadonly) {
      arr.push(
        <div key='readonly' data-id='readonly'>
          <Tooltip text={FORM_ALERTS.READONLY_WARNING}>
            <V2Badge status={StatusType.Disabled} label='Read Only' />
          </Tooltip>
        </div>,
      );
    }
    if (backendStatus) {
      arr.push(
        <div key='backend-status' data-id='backend-status'>
          <Tooltip text={backendStatus.tooltip}>
            <V2Badge {...backendStatus} />
          </Tooltip>
        </div>,
      );
    }
    if (instrumentationStatus) {
      arr.push(
        <div key='instrumentation-status' data-id='instrumentation-status'>
          <Tooltip text={instrumentationStatus.tooltip}>
            <V2Badge {...instrumentationStatus} />
          </Tooltip>
        </div>,
      );
    }

    return arr;
  }, [isReadonly, backendStatus, instrumentationStatus]);

  const right = useMemo(() => {
    const arr = [];

    if (tokenStatus) {
      arr.push(
        <div key='token-status' data-id='token-status'>
          <Tooltip text={tokenStatus.tooltip}>
            <V2Badge {...tokenStatus} />
          </Tooltip>
        </div>,
      );
    }

    if (capabilities.canManageRecommendations) {
      const label =
        recommendationsTotal > 0
          ? `${RECOMMENDATIONS_DRAWER_TITLE} (${recommendedCount}/${recommendationsTotal})`
          : RECOMMENDATIONS_DRAWER_TITLE;
      arr.push(
        <RecommendationsButton
          key='recommendations-drawer'
          data-id='recommendations-drawer'
          type='button'
          $emphasized={hasPendingRecommendations}
          $animating={isIntroAnimating}
          onClick={toggleRecommendationsDrawer}
        >
          <RecommendationsButtonInner>
            <OdigosLogo size={16} />
            <Typography size={TypographySize.XS} color={TypographyColor.Primary} nowrap>
              {label}
            </Typography>
          </RecommendationsButtonInner>
        </RecommendationsButton>,
      );
    }

    arr.push(
      <div key='system-drawer' data-id='system-drawer'>
        <IconButton icon={TerminalIcon} onClick={toggleSystemDrawer} />
      </div>,
    );

    return arr;
  }, [tokenStatus, capabilities.canManageRecommendations, recommendedCount, recommendationsTotal, hasPendingRecommendations, isIntroAnimating]);

  return (
    <>
      <V2Header left={left} right={right} />
      <SystemDrawer isOpen={isSystemDrawerOpen} onClose={toggleSystemDrawer} />
      {capabilities.canManageRecommendations && (
        <RecommendationsDrawer isOpen={isRecommendationsDrawerOpen} onClose={toggleRecommendationsDrawer} />
      )}
    </>
  );
};
