export function getDestinationNodes(height: number, destinations: any[]) {
  console.log({ height });
  if (!destinations || isNaN(height)) return [];
  let nodes: any = [];
  const canvasHeight = height;
  const listItemHeight = 120; // Adjust this value to the desired height of each list item
  const totalListItemsHeight = destinations?.length * listItemHeight;

  let topPosition = (canvasHeight - totalListItemsHeight) / 2;

  destinations.forEach((data, index) => {
    const y = topPosition;
    nodes.push({
      id: `destination-${index}`,
      type: "destination",
      data,
      position: { x: 800, y },
    });
    topPosition += 100;
  });
  return nodes;
}

export function getSourcesNodes(height: number, sources: any[]) {
  if (!sources || isNaN(height)) return [];
  let nodes: any = [];
  const canvasHeight = height;
  const listItemHeight = 120; // Adjust this value to the desired height of each list item
  const totalListItemsHeight = sources.length * listItemHeight;

  let topPosition = (canvasHeight - totalListItemsHeight) / 2;

  sources.forEach((data, index) => {
    const y = topPosition;
    nodes.push({
      id: `source-${index}`,
      type: "namespace",
      data,
      position: { x: 0, y },
    });
    topPosition += 100;
  });
  return nodes;
}
