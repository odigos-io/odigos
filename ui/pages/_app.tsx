import "../styles/globals.css";
import type { AppProps } from "next/app";
import Sidebar from "@/components/Sidebar";
import Head from "next/head";

function App({ Component, pageProps }: AppProps) {
  const title = "odigos UI";
  return (
    <>
      <Head>
        <title key="title">{title}</title>
        <meta key="twitter:title" name="twitter:title" content={title} />
        <meta key="og:title" property="og:title" content={title} />
      </Head>
      <div className="flex flex-row antialiased bg-white">
        <Sidebar />
        <div className="pt-10 pl-5 w-full text-gray-700 text-xl">
          <Component {...pageProps} />
        </div>
      </div>
    </>
  );
}

export default App;
