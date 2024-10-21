import { useComputePlatform } from '../compute-platform';

// Define the hook
export const useGetActions = () => {
  const { data } = useComputePlatform();

  return {
    actions: data?.computePlatform.actions || [], // Return the actions or an empty array if not available
  };
};
