export default function Card({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-white hover:bg-gray-100 shadow-lg border border-gray-200 rounded-lg">
      <div className="flex flex-row p-3 items-center space-x-2">{children}</div>
    </div>
  );
}
