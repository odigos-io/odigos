import { ApplicationData } from "@/types/apps";
import GolangLogo from "@/img/tech/go.svg";
import PythonLogo from "@/img/tech/python.svg";
import DotNetLogo from "@/img/tech/dotnet.svg";
import JavaLogo from "@/img/tech/java.svg";
import JavascriptLogo from "@/img/tech/nodejs.svg";
import DaemonSetLogo from "@/img/tech/ds.svg";
import DeploymentLogo from "@/img/tech/deployment.svg";
import StatefulSetLogo from "@/img/tech/sts.svg";
import { useState } from "react";

interface AppCardProps {
  disabled: boolean;
  selected: boolean;
  onSelection: (id: string, selected: boolean) => void;
}

export default function AppCard({
  id,
  name,
  namespace,
  languages,
  kind,
  disabled,
  selected,
  onSelection,
}: ApplicationData & AppCardProps) {
  return (
    <label
      className={`shadow-lg border border-gray-200 rounded-lg ${
        selected
          ? "bg-blue-500 text-white cursor-pointer"
          : disabled
          ? "bg-gray-100 opacity-50 cursor-not-allowed"
          : "bg-white hover:bg-gray-100 cursor-pointer"
      }`}
    >
      <input
        type="checkbox"
        className="hidden"
        disabled={disabled}
        onChange={() => {
          onSelection(id, !selected);
        }}
      />
      <div className="flex flex-row p-3 items-center space-x-4">
        {getLangIcon(languages[0], kind.toString(), "w-12 h-12")}
        <div className="flex flex-col items-start">
          <div className="font-bold">{name}</div>
          <div>{kind}</div>
          <div>namespace: {namespace}</div>
        </div>
      </div>
    </label>
  );
}

function getLangIcon(lang: string, kind: string, classes: string) {
  switch (lang) {
    case "go":
      return <GolangLogo className={classes} />;
    case "python":
      return <PythonLogo className={classes} />;
    case "java":
      return <JavaLogo className={classes} />;
    case "javascript":
      return <JavascriptLogo className={classes} />;
    case "dotnet":
      return <DotNetLogo className={classes} />;
    default:
      switch (kind) {
        case "DaemonSet":
          return <DaemonSetLogo className={classes} />;
        case "StatefulSet":
          return <StatefulSetLogo className={classes} />;
        default:
          return <DeploymentLogo className={classes} />;
      }
  }
}
