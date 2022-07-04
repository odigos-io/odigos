import { getConfiguration } from "@/utils/config";
import type { NextPage } from "next";

const DestinationsPage: NextPage = () => {
  return <div className="text-3xl">This is the Destinations Screen</div>;
};

export const getServerSideProps = async () => {
  const config = await getConfiguration();
  if (!config) {
    return {
      redirect: {
        destination: "/setup",
        permanent: false,
      },
    };
  }

  return {
    props: {},
  };
};

export default DestinationsPage;
