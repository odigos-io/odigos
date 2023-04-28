import DestsGrid from "@/components/DestsGrid";
import { getConfiguration } from "@/utils/config";
import Vendors, { VendorType } from "@/vendors/index";
import { Combobox } from "@headlessui/react";
import type { NextPage } from "next";
import { useState } from "react";

const AddNewDestinationPage: NextPage = () => {

  const people = [
  'Durward Reynolds',
  'Kenton Towne',
  'Therese Wunsch',
  'Benedict Kessler',
  'Katelyn Rohan',
]


  const [selectedDest, setSelectedDest] = useState(Vendors[0])
  const [query, setQuery] = useState('')

  const filteredDest =
    query === ''
      ? Vendors
      : Vendors.filter((vendor) => {
          return vendor.displayName.toLowerCase().includes(query.toLowerCase()) || vendor.name.toLowerCase().includes(query.toLowerCase())
        })
      

  return (
    <div className="flex flex-col">
      <div className="flex flex-col md:flex-row">
      <div className="text-4xl font-medium">Add New Destination</div>
      <div className="mx-1 md:mx-20 z-30 bg-white relative">
      <Combobox value={selectedDest} onChange={setSelectedDest}>
        <Combobox.Input className="w-full md:w-[600px] rounded-[2px] overflow-hidden list-none" placeholder="Search a destination" onChange={(event) => setQuery(event.target.value)} />
          <Combobox.Options>
            {filteredDest.map((vendor) => (
              <Combobox.Option onClick={() => setQuery(vendor.name)} key={vendor?.name} value={vendor.name}>
                {vendor.name}
              </Combobox.Option>
            ))}
          </Combobox.Options>
      </Combobox>
      </div>
      
      </div>
      <div className="text-2xl mt-24 mb-6 absolute">
        Choose an observability backend from the list
      </div>

      {query && 
          <div className="space-y-10 absolute mt-36">
              <>
                <DestsGrid
                  vendors={filteredDest.filter((v) => v.type === VendorType.MANAGED)}
                  title="Managed"
                />   
                <DestsGrid
                  vendors={filteredDest.filter((v) => v.type === VendorType.HOSTED)}
                  title="Self-hosted"
                />
              </>
          </div>
      }

      { !query && 
        <div className="space-y-10 absolute mt-36">
          <DestsGrid
            vendors={Vendors.filter((v) => v.type === VendorType.MANAGED)}
            title="Managed"
          />
          <DestsGrid
            vendors={Vendors.filter((v) => v.type === VendorType.HOSTED)}
            title="Self-hosted"
          />
        </div>
      }
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
