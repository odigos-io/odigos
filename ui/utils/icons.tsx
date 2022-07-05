import GolangLogo from "@/img/tech/go.svg";
import PythonLogo from "@/img/tech/python.svg";
import DotNetLogo from "@/img/tech/dotnet.svg";
import JavaLogo from "@/img/tech/java.svg";
import JavascriptLogo from "@/img/tech/nodejs.svg";

export function getLangIcon(lang: string, classes: string) {
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
      return null;
  }
}
