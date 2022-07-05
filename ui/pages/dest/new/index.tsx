import DestsGrid from "@/components/DestsGrid";
import { getConfiguration } from "@/utils/config";
import type { NextPage } from "next";

const AddNewDestinationPage: NextPage = () => {
  return (
    <div className="flex flex-col">
      <div className="text-4xl font-medium">Add New Destination</div>
      <div className="text-2xl mt-4 mb-6">
        Choose an observability backend from the list
      </div>
      <DestsGrid />
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

export default AddNewDestinationPage;
