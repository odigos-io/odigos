import Card from "@/components/Cards";
import type { ApplicationData } from "@/types/apps";
import useSWR, { Key, Fetcher } from "swr";
import GolangLogo from "@/img/tech/go.svg";
import PythonLogo from "@/img/tech/python.svg";
import DotNetLogo from "@/img/tech/dotnet.svg";
import JavaLogo from "@/img/tech/java.svg";
import JavascriptLogo from "@/img/tech/nodejs.svg";

export default function Sources() {
  const fetcher: Fetcher<ApplicationData[], any> = (args: any) =>
    fetch(args).then((res) => res.json());
  const { data, error } = useSWR<ApplicationData[]>("/api/apps", fetcher);
  if (error) return <div>failed to load</div>;
  if (!data) return <div>loading...</div>;

  return (
    <div className="flex flex-col space-y-8">
      {data
        .filter((source) => source.languages && source.languages.length > 0)
        .map((source) => (
          <Card key={source.id}>
            {getLangIcon(source.languages[0], "w-8 h-8")}
            <div className="flex flex-col">
              <div>{source.name}</div>
              {source.instrumented ? (
                <div className="text-green-600">Instrumented</div>
              ) : (
                <div className="text-orange-400">Not Instrumented</div>
              )}
            </div>
          </Card>
        ))}
    </div>
  );
}

function getLangIcon(lang: string, classes: string) {
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
