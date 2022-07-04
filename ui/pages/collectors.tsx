import { getConfiguration } from "@/utils/config";
import type { NextPage } from "next";

const CollectorsPage: NextPage = () => {
  return <div className="text-3xl">This is the Collectors Screen</div>;
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

export default CollectorsPage;
