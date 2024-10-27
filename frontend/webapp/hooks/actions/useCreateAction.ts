import type { ActionInput } from '@/types';
import { useMutation } from '@apollo/client';
import { CREATE_ACTION } from '@/graphql/mutations/action';

export const useCreateAction = () => {
  const [createAction] = useMutation(CREATE_ACTION);

  const createNewAction = async (action: ActionInput) => {
    try {
      const { data } = await createAction({
        variables: { action },
      });
      return data?.createAction?.id;
    } catch (error) {
      console.error('Error creating new action:', error);
      throw error;
    }
  };

  return { createNewAction };
};
