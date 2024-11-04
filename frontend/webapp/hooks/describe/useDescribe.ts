import { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
import { getOdigosDescription, getSourceDescription } from '@/services';
import { use } from 'chai';

export function useDescribe() {
  const [namespace, setNamespace] = useState<string>('');
  const [kind, setKind] = useState<string>('');
  const [name, setName] = useState<string>('');

  // Fetch Odigos description
  const {
    data: odigosDescription,
    isLoading: isOdigosLoading,
    refetch: refetchOdigosDescription,
  } = useQuery(['odigosDescription'], getOdigosDescription, {
    enabled: false,
  });

  // Fetch source description based on namespace, kind, and name
  const {
    data: sourceDescription,
    isLoading: isSourceLoading,
    refetch: refetchSourceDescription,
  } = useQuery(
    ['sourceDescription'],
    () => getSourceDescription(namespace, kind.toLowerCase(), name),

    {
      onError: (error) => {
        console.log(error);
      },
      enabled: false,
    }
  );

  useEffect(() => {
    if (namespace && kind && name) {
      refetchSourceDescription();
    }
  }, [namespace, kind, name]);

  useEffect(() => {
    console.log({ sourceDescription });
  }, [sourceDescription]);

  // Function to set parameters for source description and refetch
  function fetchSourceDescription(
    newNamespace: string,
    newKind: string,
    newName: string
  ) {
    setNamespace(newNamespace);
    setKind(newKind);
    setName(newName);
    // refetchSourceDescription();
  }

  return {
    odigosDescription,
    sourceDescription,
    isOdigosLoading,
    isSourceLoading,
    refetchOdigosDescription,
    fetchSourceDescription,
  };
}
