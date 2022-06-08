export default function Collectors() {
  return <NoCollectorsCard />;
}

function NoCollectorsCard() {
  return (
    <div className="mx-auto cursor-not-allowed mt-24 bg-gray-100 shadow-lg border border-gray-200 rounded-lg w-64">
      <div className="flex flex-col items-center justify-center p-3 text-center">
        <div>Collectors not deployed yet.</div>
        <div>Configure destinations to start</div>
      </div>
    </div>
  );
}
