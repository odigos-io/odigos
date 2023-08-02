"use client";
import { getSources } from "@/services";
import { QUERIES } from "@/utils/constants";
import { useSearchParams } from "next/navigation";
import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";

const SOURCE = "source";

export default function ManageSourcePage() {
  const [currentSource, setCurrentSource] = useState(null);
  const searchParams = useSearchParams();

  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  useEffect(onPageLoad, [sources]);
  function onPageLoad() {
    const search = searchParams.get(SOURCE);
    const source = sources?.find((item) => item.name === search);
    source && setCurrentSource(source);
  }

  return (
    <>
      <div>{currentSource?.name}</div>
    </>
  );
}
