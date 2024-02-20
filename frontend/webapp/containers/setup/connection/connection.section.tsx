import React, { useLayoutEffect, useMemo, useState } from 'react';
import { useQuery } from 'react-query';
import { QUERIES } from '@/utils/constants';
import { getDestination } from '@/services';
import { KeyvalLoader } from '@/design.system';
import { useSearchParams } from 'next/navigation';
import { useConnect, useNotification } from '@/hooks';
import { CreateConnectionForm, QuickHelp } from '@/components/setup';
import {
  LoaderWrapper,
  CreateConnectionContainer,
} from './connection.section.styled';

export interface DestinationBody {
  name: string;
  type?: string;
  signals: {
    [key: string]: boolean;
  };
  fields: {
    [key: string]: string;
  };
}

export function ConnectionSection({ supportedSignals }) {
  const [type, setType] = useState<string>('');

  const { connect } = useConnect();
  const searchParams = useSearchParams();
  const { Notification } = useNotification();

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATION_TYPE],
    () => getDestination(type),
    {
      enabled: !!type,
    }
  );

  useLayoutEffect(onPageLoad, []);

  function onPageLoad() {
    const search = searchParams.get('type');
    search && setType(search);
  }

  const videoList = useMemo(
    () =>
      data?.fields
        ?.filter((field) => field?.video_url)
        ?.map((field) => ({
          name: field.display_name,
          src: field.video_url,
          thumbnail_url: field.thumbnail_url,
        })),
    [data]
  );

  function createDestination(formData: DestinationBody) {
    connect({ type, ...formData });
  }

  if (isLoading)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <CreateConnectionContainer>
      {data?.fields && (
        <CreateConnectionForm
          fields={data?.fields}
          onSubmit={createDestination}
          supportedSignals={supportedSignals}
        />
      )}
      {videoList?.length > 0 && <QuickHelp data={videoList} />}
      <Notification />
    </CreateConnectionContainer>
  );
}
