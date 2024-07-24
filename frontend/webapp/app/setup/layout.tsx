'use client';
import React from 'react';
import { SetupHeader } from '@/components';

export default function SetupLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <SetupHeader />
      {children}
    </>
  );
}
