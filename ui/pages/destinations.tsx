import CurrentDestsGrid from "@/components/CurrentDestsGrid";
import { getConfiguration } from "@/utils/config";
import type { NextPage } from "next";
import Link from "next/link";

const DestinationsPage: NextPage = () => {
  return (
    <div className="space-y-12">
      <div className="flex flex-row items-center">
        <div className="text-4xl font-medium">Active Destinations</div>
        <Link href="/dest/new">
          <a className="hover:cursor-pointer ml-12 text-white focus:ring-4 focus:outline-none font-medium rounded-md text-sm px-6 py-3 text-center inline-flex items-center mr-4 bg-green-600 hover:bg-green-700 focus:ring-green-800">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="w-5 h-5 mr-2 -ml-1"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z"
                clipRule="evenodd"
              />
            </svg>
            Add New Destination
          </a>
        </Link>
      </div>
      <CurrentDestsGrid />
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

export default DestinationsPage;
