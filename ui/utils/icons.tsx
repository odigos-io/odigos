import GolangLogo from "@/img/tech/go.svg";
import PythonLogo from "@/img/tech/python.svg";
import DotNetLogo from "@/img/tech/dotnet.svg";
import JavaLogo from "@/img/tech/java.svg";
import JavascriptLogo from "@/img/tech/nodejs.svg";
import DaemonSetLogo from "@/img/tech/ds.svg";
import DeploymentLogo from "@/img/tech/deployment.svg";
import StatefulSetLogo from "@/img/tech/sts.svg";

export function getLangIcon(lang: string, classes: string, kind?: string) {
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
      if (!kind) {
        return null;
      } else {
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
}
