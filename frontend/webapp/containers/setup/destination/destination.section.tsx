import { DestinationOptionMenu } from "@/components/setup";
import React, { useEffect, useMemo, useState } from "react";

export function DestinationSection() {
  const [searchFilter, setSearchFilter] = useState<string>("");

  return (
    <>
      <DestinationOptionMenu
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
      />
    </>
  );
}
