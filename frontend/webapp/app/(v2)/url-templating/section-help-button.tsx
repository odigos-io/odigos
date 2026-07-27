'use client';

import React, { useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import styled from 'styled-components';
import { QuestionCircleIcon } from '@odigos/ui-kit/icons';

const Trigger = styled.span`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  color: ${({ theme }) => theme.v2.colors.grey[400]};
  cursor: help;

  &:hover {
    color: ${({ theme }) => theme.v2.colors.white[500]};
  }
`;

const Popup = styled.div`
  position: fixed;
  z-index: ${({ theme }) => theme.v2.zIndex.tooltip};
  box-sizing: border-box;
  width: min(420px, calc(100vw - 32px));
  padding: 12px 14px;
  border-radius: 8px;
  pointer-events: none;
  background: ${({ theme }) => theme.v2.colors.silver[200]};
  border: 1px solid ${({ theme }) => theme.v2.colors.silver[400]};
  box-shadow: 0 8px 20px color-mix(in srgb, ${({ theme }) => theme.v2.colors.black[500]} 22%, transparent);
`;

const PopupTitle = styled.div`
  margin: 0 0 6px;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xs}px;
  font-weight: 600;
  line-height: 1.35;
  color: ${({ theme }) => theme.v2.colors.grey[900]};
`;

const PopupBody = styled.div`
  margin: 0;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-size: ${({ theme }) => theme.v2.text.size.xxs}px;
  font-weight: 400;
  line-height: 1.5;
  color: ${({ theme }) => theme.v2.colors.grey[800]};
`;

type SectionHelpButtonProps = {
  'data-id'?: string;
  title: string;
  text: string;
};

export function SectionHelpButton({ 'data-id': dataId, title, text }: SectionHelpButtonProps) {
  const triggerRef = useRef<HTMLSpanElement>(null);
  const [open, setOpen] = useState(false);
  const [position, setPosition] = useState<{ top: number; left: number } | null>(null);

  useLayoutEffect(() => {
    if (!open || !triggerRef.current) {
      return;
    }
    const rect = triggerRef.current.getBoundingClientRect();
    const width = Math.min(420, window.innerWidth - 32);
    const left = Math.min(Math.max(16, rect.left), window.innerWidth - width - 16);
    const top = Math.min(rect.bottom + 8, window.innerHeight - 16);
    setPosition({ top, left });
  }, [open]);

  return (
    <>
      <Trigger
        ref={triggerRef}
        data-id={dataId}
        aria-label={`Help: ${title}`}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        tabIndex={0}
      >
        <QuestionCircleIcon size={14} />
      </Trigger>
      {open && position
        ? createPortal(
            <Popup role="tooltip" style={{ top: position.top, left: position.left }}>
              <PopupTitle>{title}</PopupTitle>
              <PopupBody>{text}</PopupBody>
            </Popup>,
            document.body,
          )
        : null}
    </>
  );
}
