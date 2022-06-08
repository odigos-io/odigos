import type { NextPage } from "next";
import Head from "next/head";
import Header from "@/components/Header";
import Topology from "@/components/Topology";
import Image from "next/image";

const Home: NextPage = () => {
  return (
    <div className="flex h-screen flex-col">
      <Head>
        <title>Observability Control Plane - UI</title>
        <link rel="icon" href="/favicon.ico" />
      </Head>
      <Header />
      <Topology />
    </div>
  );
};

export default Home;
