import Spinner from "@/components/Spinner";

export default function LoadingPage() {
  return (
    <div className="flex items-center justify-center w-full h-full">
      <Spinner className="w-12 h-12" />
    </div>
  );
}
