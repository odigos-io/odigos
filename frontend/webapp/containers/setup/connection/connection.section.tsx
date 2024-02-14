import React, { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery } from 'react-query';
import { CreateConnectionForm, QuickHelp } from '@/components/setup';
import {
  CreateConnectionContainer,
  LoaderWrapper,
} from './connection.section.styled';
import { getDestination, setDestination } from '@/services';
import { QUERIES, ROUTES, SETUP } from '@/utils/constants';
import { KeyvalLoader } from '@/design.system';
import { useNotification } from '@/hooks';
import { useRouter, useSearchParams } from 'next/navigation';

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

export function ConnectionSection({ sectionData }) {
  const [type, setType] = useState('');
  const { show, Notification } = useNotification();

  const searchParams = useSearchParams();

  const router = useRouter();
  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATION_TYPE],
    () => getDestination(type),
    {
      enabled: !!type,
    }
  );

  const { mutate } = useMutation((body) => setDestination(body));

  useEffect(onPageLoad, []);

  useEffect(() => {
    console.log({ data });
  }, [data]);

  function onPageLoad() {
    const search = searchParams.get('type');
    if (search) {
      console.log({ search });
      setType(search);
    }
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
    // const { type } = sectionData;
    const body: any = {
      type,
      ...formData,
    };

    mutate(body, {
      onSuccess: () => router.push(ROUTES.OVERVIEW),
      onError: ({ response }) => {
        const message = response?.data?.message || SETUP.ERROR;
        show({
          type: 'error',
          message,
        });
      },
    });
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
          supportedSignals={{}} //{sectionData?.supported_signals}
        />
      )}
      {videoList?.length > 0 && <QuickHelp data={videoList} />}
      <Notification />
    </CreateConnectionContainer>
  );
}
