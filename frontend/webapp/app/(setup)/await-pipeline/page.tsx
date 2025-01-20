import React from 'react';
import dynamic from 'next/dynamic';

const AwaitPipelineContainer = dynamic(() => import('@/containers/setup/await-pipeline'));

export default function AwaitPipelinePage() {
  return (
    <>
      <AwaitPipelineContainer />
    </>
  );
}
