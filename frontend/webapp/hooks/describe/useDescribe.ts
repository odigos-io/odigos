import { useEffect, useState } from 'react';
import { useQuery } from 'react-query';
import { getOdigosDescription, getSourceDescription } from '@/services';

export function useDescribe() {
  const [namespace, setNamespace] = useState<string>('');
  const [kind, setKind] = useState<string>('');
  const [name, setName] = useState<string>('');

  useEffect(() => {
    if (namespace && kind && name) {
      refetchSourceDescription();
    }
  }, [namespace, kind, name]);

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
    ['sourceDescription', namespace, kind, name],
    () => getSourceDescription(namespace, kind, name),
    {
      enabled: false,
    }
  );

  // Function to set parameters for source description and refetch
  function fetchSourceDescription(
    newNamespace: string,
    newKind: string,
    newName: string
  ) {
    setNamespace(newNamespace);
    setKind(newKind);
    setName(newName);
    console.log({ newNamespace, newKind, newName });
    try {
      if (newNamespace && newKind && newName) {
        console.log('object');
        // refetchSourceDescription();
      }
    } catch (error) {
      console.error('Error fetching source description:', error);
    }
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
