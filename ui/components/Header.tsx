import Link from "next/link";

export default function Header({}) {
  return (
    <div className="py-4 pl-6 bg-slate-500 flex flex-row">
      <div className="text-4xl text-white font-semibold hidden md:block">
        keyval
      </div>
      <div className="text-3xl mx-auto text-white font-thin">
        <Link href="/">Observability Control Plane</Link>
      </div>
    </div>
  );
}
