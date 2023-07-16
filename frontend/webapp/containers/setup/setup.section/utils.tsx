import { SETUP } from "@/utils/constants";

export type Step = {
  id: string;
  status: string;
  index: number;
  title: string;
};

export const STEPS = [
  {
    index: 1,
    id: SETUP.STEPS.ID.CHOOSE_SOURCE,
    title: SETUP.STEPS.CHOOSE_SOURCE,
    status: SETUP.STEPS.STATUS.ACTIVE,
  },
  {
    index: 2,
    id: SETUP.STEPS.ID.CHOOSE_DESTINATION,
    title: SETUP.STEPS.CHOOSE_DESTINATION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
  {
    index: 3,
    id: SETUP.STEPS.ID.CREATE_CONNECTION,
    title: SETUP.STEPS.CREATE_CONNECTION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
];
