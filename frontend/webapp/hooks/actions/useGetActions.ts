import { GET_ACTIONS } from '@/graphql';
import { useQuery } from '@apollo/client';

// Define the type for an individual action
interface IcaInstanceResponse {
  id: string;
  type: string;
  spec: string; // The spec is returned as a JSON string
}

// Define the response type for the query
interface GetActionsData {
  actions: IcaInstanceResponse[];
}

// Define the hook
export const useGetActions = () => {
  // Use Apollo's useQuery hook to fetch the actions
  const { loading, error, data, refetch } =
    useQuery<GetActionsData>(GET_ACTIONS);

  return {
    actions: data?.actions || [], // Return the actions or an empty array if not available
    loading, // Return the loading state
    error, // Return the error state
    refetch, // Function to refetch the data manually if needed
  };
};
