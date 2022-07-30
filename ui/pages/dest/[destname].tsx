import type { NextPage } from "next";
import Head from "next/head";
import Header from "@/components/Header";
import { useRouter } from "next/router";
import Vendors from "@/vendors/index";
import * as k8s from "@kubernetes/client-node";

type EditDestProps = {
  destName: string;
  destType: string;
  currentValues: { [key: string]: string };
  signals: any;
};

const NewDestination: NextPage<EditDestProps> = ({
  destType,
  currentValues,
  destName,
  signals,
}) => {
  const router = useRouter();
  const vendor = Vendors.filter((v) => v.name === destType)[0];
  if (!vendor) {
    return (
      <div className="text-4xl font-medium">Observability Vendor Not Found</div>
    );
  }

  const fields = vendor.getFields(signals);
  console.log(fields);
  const deleteDest = async () => {
    const response = await fetch(`/api/dest/${destName}`, {
      method: "DELETE",
    });

    if (response.ok) {
      router.push("/destinations");
    }
  };

  const updateDest = async (event: any) => {
    event.preventDefault();
    var formData = new FormData(event.target);
    var object: { [key: string]: string } = {};
    formData.forEach(function (value, key) {
      object[key] = value.toString();
    });
    const JSONdata = JSON.stringify(object);
    const response = await fetch(`/api/dest/${destName}`, {
      body: JSON.stringify({
        destType,
        values: JSONdata,
      }),
      headers: {
        "Content-Type": "application/json",
      },
      method: "POST",
    });

    if (response.ok) {
      router.push("/destinations");
    }
  };

  return (
    <div className="flex flex-col">
      <div className="text-4xl p-8 capitalize text-gray-900">
        Edit Destination: {destName}
      </div>
      <div className="pl-14 max-w-md">
        <form
          className="grid grid-cols-1 gap-6"
          onSubmit={updateDest}
          name="newdest"
        >
          {fields &&
            fields
              .filter((f) => currentValues.hasOwnProperty(f.name))
              .map((f) => {
                return (
                  <label className="block" key={f.id}>
                    <span className="text-gray-700">{f.displayName}</span>
                    <input
                      id={f.id}
                      name={f.name}
                      type={f.type}
                      className="
                    mt-1
                    block
                    w-full
                    rounded-md
                    border-gray-300
                    shadow-sm
                    focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50
                  "
                      placeholder=""
                      required
                      defaultValue={currentValues[f.name]}
                    />
                  </label>
                );
              })}
          <div className="mx-auto flex flex-row justify-between">
            <button
              type="submit"
              className="mt-4 text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-8 py-2.5 mr-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800"
            >
              Save Destination
            </button>
            <button
              type="button"
              onClick={deleteDest}
              className="mt-4 text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:ring-red-300 font-medium rounded-lg text-sm px-5 py-2.5 mr-2 mb-2 dark:bg-red-600 dark:hover:bg-red-700 focus:outline-none dark:focus:ring-red-800"
            >
              Delete Destination
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default NewDestination;

export const getServerSideProps = async ({ query }: any) => {
  const { destname } = query;
  const kc = new k8s.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(k8s.CustomObjectsApi);
  const response = await k8sApi.getNamespacedCustomObject(
    "odigos.io",
    "v1alpha1",
    process.env.CURRENT_NS || "odigos-system",
    "destinations",
    destname
  );

  const { spec }: any = response.body;
  const vendor = Vendors.find((v) => v.name === spec.type);
  if (!vendor) {
    return { props: { destname: "", currentValues: {} } };
  }

  const props: EditDestProps = {
    destName: destname,
    destType: spec.type,
    currentValues: vendor.mapDataToFields(spec.data),
    signals: spec.signals.reduce((acc: any, signal: any) => {
      Object.assign(acc, { [signal]: true });
      return acc;
    }, {}),
  };

  return {
    props,
  };
};
