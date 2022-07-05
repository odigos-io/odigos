import Spinner from "@/components/Spinner";

export default function Alert({ message }: any) {
  return (
    <div
      className="w-fit p-4 mb-4 text-sm rounded-lg bg-blue-200 text-blue-800 flex flex-row items-center"
      role="alert"
    >
      <Spinner className="w-6 h-6" /> <p className="text-md">{message}</p>
    </div>
  );
}
