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
    id: "choose-source",
    title: SETUP.STEPS.CHOOSE_SOURCE,
    status: SETUP.STEPS.STATUS.ACTIVE,
  },
  {
    index: 2,
    id: "choose-destination",
    title: SETUP.STEPS.CHOOSE_DESTINATION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
  {
    index: 3,
    id: "create-connection",
    title: SETUP.STEPS.CREATE_CONNECTION,
    status: SETUP.STEPS.STATUS.DISABLED,
  },
];
