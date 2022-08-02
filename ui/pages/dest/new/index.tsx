import DestsGrid from "@/components/DestsGrid";
import { getConfiguration } from "@/utils/config";
import Vendors, { VendorType } from "@/vendors/index";
import type { NextPage } from "next";

const AddNewDestinationPage: NextPage = () => {
  return (
    <div className="flex flex-col">
      <div className="text-4xl font-medium">Add New Destination</div>
      <div className="text-2xl mt-4 mb-6">
        Choose an observability backend from the list
      </div>
      <div className="space-y-10">
        <DestsGrid
          vendors={Vendors.filter((v) => v.type === VendorType.MANAGED)}
          title="Managed"
        />
        <DestsGrid
          vendors={Vendors.filter((v) => v.type === VendorType.HOSTED)}
          title="Self-hosted"
        />
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

export default AddNewDestinationPage;
