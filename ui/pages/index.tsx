import type { NextPage } from "next";
import { getConfiguration } from "@/utils/config";

const Home: NextPage = () => {
  return (
    <div className="w-full h-full">
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Sources</div>
      </div>
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Collectors</div>
      </div>
      <div className="h-1/3 w-full">
        <div className="text-4xl font-medium">Destinations</div>
      </div>
    </div>
  );
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

export default Home;
