import { Listbox, Transition } from '@headlessui/react'
import { CheckIcon, ChevronUpDownIcon } from '@heroicons/react/20/solid'
import { useState, Fragment } from "react";
import { KubernetesNamespace, KubernetesObjectsInNamespaces } from "@/types/apps";

type NamespaceSelectorProps = {
  data: KubernetesObjectsInNamespaces
  selectedNamespace: KubernetesNamespace
  setSelectedNamespace: (namespace: KubernetesNamespace) => void
}

export default function NamespaceSelector({ data, selectedNamespace, setSelectedNamespace }: NamespaceSelectorProps) {
  return (
    <div className="w-72">
      <Listbox value={selectedNamespace} onChange={setSelectedNamespace}>
        <div className="relative">
          <Listbox.Button className="relative w-full cursor-default rounded-lg bg-white py-2 pl-3 pr-10 text-left shadow-md focus:outline-none focus-visible:border-indigo-500 focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-opacity-75 focus-visible:ring-offset-2 focus-visible:ring-offset-blue-300 sm:text-sm">
            {isLabeled(selectedNamespace) && <span className="absolute inset-y-0 flex left-0 items-center pl-3 text-green-600">
              <CheckIcon className="h-5 w-5" aria-hidden="true" />
            </span>}
            <span className="block truncate pl-6">{selectedNamespace.name}</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
              <ChevronUpDownIcon
                className="h-5 w-5 text-gray-400"
                aria-hidden="true"
              />
            </span>
          </Listbox.Button>
          <Transition
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="absolute mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm">
              {data && data.namespaces && data.namespaces.map((namespace, namespaceIdx) => (
                <Listbox.Option
                  key={namespaceIdx}
                  className={({ active }) =>
                    `relative cursor-default select-none py-2 pl-10 pr-4 ${active ? 'bg-blue-100 text-blue-900' : 'text-gray-900'
                    }`
                  }
                  value={namespace}
                >
                  {({ selected }) => (
                    <>
                      <span
                        className={`block truncate ${selected ? 'font-medium' : 'font-normal'
                          }`}
                      >
                        {namespace.name}
                      </span>
                      {isLabeled(namespace) ? (
                        <span className="absolute inset-y-0 left-0 flex items-center pl-3 text-green-600">
                          <CheckIcon className="h-5 w-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              ))}
            </Listbox.Options>
          </Transition>
        </div>
      </Listbox>
    </div>
  )
}

function isLabeled(namespace: KubernetesNamespace): boolean {
  return namespace.labeled || namespace.objects.some((obj) => obj.labeled);
}