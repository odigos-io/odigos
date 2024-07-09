import React from 'react';
import { useConnect } from '@/hooks';
import { useQuery } from 'react-query';
import { QUERIES } from '@/utils/constants';
import { getDestination } from '@/services';
import { KeyvalLoader } from '@/design.system';
import { CreateConnectionForm } from '@/components/setup';
import {
  LoaderWrapper,
  CreateConnectionContainer,
} from './connection.section.styled';

export interface DestinationBody {
  name: string;
  type: string;
  signals: {
    [key: string]: boolean;
  };
  fields: {
    [key: string]: string;
  };
}

export function ConnectionSection({ destination, type }) {
  const { connect } = useConnect();

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATION_TYPE],
    () => getDestination(type),
    {
      enabled: !!type,
    }
  );

  function createDestination(formData: DestinationBody) {
    connect({ ...formData });
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
          fields={data.fields}
          onSubmit={createDestination}
          destination={destination}
        />
      )}
    </CreateConnectionContainer>
  );
}
