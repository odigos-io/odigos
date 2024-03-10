'use client';
import { OVERVIEW } from '@/utils';
import { OverviewHeader } from '@/components';
import { DestinationContainer } from '@/containers';

export default function DestinationDashboardPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <DestinationContainer />
    </>
  );
}
