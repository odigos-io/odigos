export function filterDataByTextQuery(data: any, searchFilter: string) {
  //filter each category items by search query
  if (!searchFilter) return data?.categories;

  const filteredData = data?.categories.map((category: any) => {
    const items = category.items.filter((item: any) => {
      const displayType = item.display_name?.toLowerCase();
      return displayType?.includes(searchFilter.toLowerCase());
    });

    return {
      ...category,
      items,
    };
  });

  return filteredData;
}

export function filterDataByMonitorsOption(data: any, monitoringOption: any) {
  const selectedMonitors = monitoringOption
    .filter((monitor: any) => monitor.tapped)
    .map((monitor: any) => monitor.title.toLowerCase());

  // if all monitors are selected, return all data
  if (selectedMonitors.length === 3) return data;

  const filteredData: any[] = [];

  data?.forEach((category: any) => {
    const supportedItems: any[] = [];

    category.items.filter((item: any) => {
      // get all supported signals for each item
      const supportedSignals: any[] = [];

      for (const monitor in item.supported_signals) {
        if (item.supported_signals[monitor].supported) {
          supportedSignals.push(monitor);
        }
      }

      const isSelected = selectedMonitors.some((monitor) =>
        supportedSignals.includes(monitor)
      );
      // if item is supported by any of the selected monitors, add it to the list
      isSelected && supportedItems.push(item);
    });

    filteredData.push({
      items: supportedItems,
      name: category.name,
    });
  });

  return filteredData;
}
