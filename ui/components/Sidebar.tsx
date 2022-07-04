import Link from "next/link";
import { useRouter } from "next/router";

export default function Sidebar() {
  const router = useRouter();
  return (
    <aside className="w-64" aria-label="Sidebar">
      <div className="h-screen overflow-y-auto py-4 px-3 rounded bg-gray-800">
        <Link href="/">
          <a className="flex items-center justify-center mb-5">
            <span className="self-center text-2xl font-semibold whitespace-nowrap text-white">
              odigos
            </span>
          </a>
        </Link>

        <ul className="space-y-2">
          {router.pathname === "/setup" ? (
            <li>
              <Link href="/setup">
                <a
                  className={`flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${
                    router.pathname === "/setup" ? "bg-gray-700" : ""
                  }`}
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    className="w-6 h-6 text-gray-500 transition duration-75 group-hover:text-white"
                    viewBox="0 0 20 20"
                    fill="currentColor"
                  >
                    <path
                      fillRule="evenodd"
                      d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className="ml-3">Setup</span>
                </a>
              </Link>
            </li>
          ) : (
            <>
              <li>
                <Link href="/">
                  <a
                    className={`flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${
                      router.pathname === "/" ? "bg-gray-700" : ""
                    }`}
                  >
                    <svg
                      className="flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                      xmlns="http://www.w3.org/2000/svg"
                    >
                      <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"></path>
                    </svg>
                    <span className="flex-1 ml-3 whitespace-nowrap">
                      Overview
                    </span>
                  </a>
                </Link>
              </li>
              <li>
                <Link href="/sources">
                  <a
                    className={`flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${
                      router.pathname === "/sources" ? "bg-gray-700" : ""
                    }`}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <span className="flex-1 ml-3 whitespace-nowrap">
                      Sources
                    </span>
                  </a>
                </Link>
              </li>
              <li>
                <Link href="/destinations">
                  <a
                    className={`flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${
                      router.pathname === "/destinations" ? "bg-gray-700" : ""
                    }`}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path d="M3 12v3c0 1.657 3.134 3 7 3s7-1.343 7-3v-3c0 1.657-3.134 3-7 3s-7-1.343-7-3z" />
                      <path d="M3 7v3c0 1.657 3.134 3 7 3s7-1.343 7-3V7c0 1.657-3.134 3-7 3S3 8.657 3 7z" />
                      <path d="M17 5c0 1.657-3.134 3-7 3S3 6.657 3 5s3.134-3 7-3 7 1.343 7 3z" />
                    </svg>
                    <span className="flex-1 ml-3 whitespace-nowrap">
                      Destinations
                    </span>
                  </a>
                </Link>
              </li>
              <li>
                <Link href="/collectors">
                  <a
                    className={`flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${
                      router.pathname === "/collectors" ? "bg-gray-700" : ""
                    }`}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      className="flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M3 3a1 1 0 011-1h12a1 1 0 011 1v3a1 1 0 01-.293.707L12 11.414V15a1 1 0 01-.293.707l-2 2A1 1 0 018 17v-5.586L3.293 6.707A1 1 0 013 6V3z"
                        clipRule="evenodd"
                      />
                    </svg>
                    <span className="flex-1 ml-3 whitespace-nowrap">
                      Collectors
                    </span>
                  </a>
                </Link>
              </li>
            </>
          )}
        </ul>
      </div>
    </aside>
  );
}
