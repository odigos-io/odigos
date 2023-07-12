import { DestinationOptionMenu } from "@/components/setup";
import React, { useEffect, useMemo, useState } from "react";

export function DestinationSection() {
  const [searchFilter, setSearchFilter] = useState<string>("");

  useEffect(() => {
    console.log({ searchFilter });
  }, [searchFilter]);
  return (
    <>
      <DestinationOptionMenu
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
      />
    </>
  );
}
