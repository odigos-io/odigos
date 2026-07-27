'use client';

import React, { useEffect, useState, type RefObject } from 'react';
import styled from 'styled-components';
import { ChevronRightIcon } from '@odigos/ui-kit/icons';
import { FlexRow, Typography, TypographyColor, TypographySize } from '@odigos/ui-kit/components';

function findScrollableParent(element: HTMLElement | null): HTMLElement | null {
  let node = element?.parentElement ?? null;
  while (node && node !== document.body) {
    const { overflowY } = getComputedStyle(node);
    if (overflowY === 'auto' || overflowY === 'scroll') {
      return node;
    }
    node = node.parentElement;
  }
  return null;
}

export function useCanScrollDown(
  anchorRef: RefObject<HTMLElement | null>,
  refreshKey: unknown,
): boolean {
  const [canScrollDown, setCanScrollDown] = useState(false);

  useEffect(() => {
    const anchor = anchorRef.current;
    if (!anchor) {
      return undefined;
    }

    const scrollEl = findScrollableParent(anchor);
    if (!scrollEl) {
      return undefined;
    }

    const update = () => {
      setCanScrollDown(scrollEl.scrollHeight - scrollEl.scrollTop - scrollEl.clientHeight > 20);
    };

    update();
    scrollEl.addEventListener('scroll', update, { passive: true });
    const resizeObserver = new ResizeObserver(update);
    resizeObserver.observe(scrollEl);
    resizeObserver.observe(anchor);

    return () => {
      scrollEl.removeEventListener('scroll', update);
      resizeObserver.disconnect();
    };
  }, [anchorRef, refreshKey]);

  return canScrollDown;
}

const ScrollEdgeHintRoot = styled(FlexRow)<{ $visible: boolean }>`
  position: sticky;
  bottom: 0;
  z-index: 2;
  width: 100%;
  min-height: 52px;
  margin-top: -52px;
  pointer-events: none;
  align-items: flex-end;
  justify-content: center;
  padding-bottom: 8px;
  opacity: ${({ $visible }) => ($visible ? 1 : 0)};
  transition: opacity 0.15s ease;
  background: linear-gradient(
    to bottom,
    transparent 0%,
    ${({ theme }) => theme.v2.colors.black[500]}aa 45%,
    ${({ theme }) => theme.v2.colors.black[500]} 100%
  );
`;

const ScrollEdgeLabel = styled(FlexRow)`
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 999px;
  background: ${({ theme }) => theme.v2.colors.silver[900]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[700]};
`;

const ScrollChevron = styled.span`
  display: inline-flex;
  transform: rotate(90deg);
  color: ${({ theme }) => theme.v2.colors.grey[400]};
`;

type DrawerScrollEdgeHintProps = {
  visible: boolean;
};

export function DrawerScrollEdgeHint({ visible }: DrawerScrollEdgeHintProps) {
  return (
    <ScrollEdgeHintRoot $visible={visible} $gap={4}>
      <ScrollEdgeLabel $gap={4}>
        <ScrollChevron>
          <ChevronRightIcon size={12} />
        </ScrollChevron>
        <Typography size={TypographySize.XXXS} color={TypographyColor.Secondary}>
          Scroll for more
        </Typography>
      </ScrollEdgeLabel>
    </ScrollEdgeHintRoot>
  );
}
