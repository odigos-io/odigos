import "../styles/globals.css";
import type { AppProps } from "next/app";
import Sidebar from "@/components/Sidebar";

function MyApp({ Component, pageProps }: AppProps) {
  return (
    <div className="flex flex-row antialiased bg-white">
      <Sidebar />
      <div className="pt-10 pl-5 w-full text-gray-700 text-xl">
        <Component {...pageProps} />
      </div>
    </div>
  );
}

export default MyApp;
