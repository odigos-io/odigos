import { AppKind, ApplicationData, KubernetesObject } from "@/types/apps";
import DaemonSetLogo from "@/img/tech/ds.svg";
import DeploymentLogo from "@/img/tech/deployment.svg";
import StatefulSetLogo from "@/img/tech/sts.svg";
import { useState } from "react";

interface AppCardProps {
  disabled: boolean;
  selected: boolean;
  object: KubernetesObject;
  onSelection: (obj: KubernetesObject) => void;
}

export default function AppCard({
  disabled,
  selected,
  object,
  onSelection,
}: ApplicationData & AppCardProps) {
  return (
    <label
      className={`shadow-lg border border-gray-200 rounded-lg ${
        selected
          ? "bg-blue-500 text-white cursor-pointer"
          : disabled
          ? "bg-gray-100 cursor-not-allowed"
          : "bg-white hover:bg-gray-100 cursor-pointer"
      }`}
    >
      <input
        type="checkbox"
        className="hidden"
        disabled={disabled}
        onChange={() => {
          onSelection(object);
        }}
      />
      <div className="flex flex-row p-3 items-center space-x-4">
        {getLangIcon(object.kind.toString(), "w-12 h-12")}
        <div className="flex flex-col items-start">
          <div className="font-bold">{object.name}</div>
          <div>{object.kind}</div>
          <div>{object.instances} {object.instances === 1 ? "running instance" : "running instances"}</div>
        </div>
      </div>
    </label>
  );
}

function getLangIcon(kind: string, classes: string) {
  return <DeploymentLogo className={classes} />;
}
