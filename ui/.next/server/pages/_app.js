(() => {
var exports = {};
exports.id = 888;
exports.ids = [888];
exports.modules = {

/***/ 3903:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

"use strict";
// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* reexport */ _app)
});

// EXTERNAL MODULE: ./styles/globals.css
var globals = __webpack_require__(6764);
// EXTERNAL MODULE: ./node_modules/next/link.js
var next_link = __webpack_require__(1664);
var link_default = /*#__PURE__*/__webpack_require__.n(next_link);
// EXTERNAL MODULE: external "next/router"
var router_ = __webpack_require__(1853);
// EXTERNAL MODULE: ./node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(5893);
;// CONCATENATED MODULE: ./components/Sidebar.tsx





function Sidebar() {
  const router = (0,router_.useRouter)();
  return /*#__PURE__*/jsx_runtime.jsx("aside", {
    className: "w-64",
    "aria-label": "Sidebar",
    children: /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "min-h-screen h-full overflow-y-auto py-4 px-3 rounded bg-gray-800",
      children: [/*#__PURE__*/jsx_runtime.jsx((link_default()), {
        href: "/",
        className: "flex items-center justify-center mb-5",
        children: /*#__PURE__*/jsx_runtime.jsx("span", {
          className: "self-center text-2xl font-semibold whitespace-nowrap text-white",
          children: "odigos"
        })
      }), /*#__PURE__*/jsx_runtime.jsx("ul", {
        className: "space-y-2",
        children: router.pathname === "/setup" ? /*#__PURE__*/jsx_runtime.jsx("li", {
          children: /*#__PURE__*/(0,jsx_runtime.jsxs)((link_default()), {
            href: "/setup",
            className: `flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${router.pathname === "/setup" ? "bg-gray-700" : ""}`,
            children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
              xmlns: "http://www.w3.org/2000/svg",
              className: "w-6 h-6 text-gray-500 transition duration-75 group-hover:text-white",
              viewBox: "0 0 20 20",
              fill: "currentColor",
              children: /*#__PURE__*/jsx_runtime.jsx("path", {
                fillRule: "evenodd",
                d: "M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z",
                clipRule: "evenodd"
              })
            }), /*#__PURE__*/jsx_runtime.jsx("span", {
              className: "ml-3",
              children: "Setup"
            })]
          })
        }) : /*#__PURE__*/(0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
          children: [/*#__PURE__*/jsx_runtime.jsx("li", {
            children: /*#__PURE__*/(0,jsx_runtime.jsxs)((link_default()), {
              href: "/",
              className: `flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${router.pathname === "/" ? "bg-gray-700" : ""}`,
              children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
                className: "flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white",
                fill: "currentColor",
                viewBox: "0 0 20 20",
                xmlns: "http://www.w3.org/2000/svg",
                children: /*#__PURE__*/jsx_runtime.jsx("path", {
                  d: "M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
                })
              }), /*#__PURE__*/jsx_runtime.jsx("span", {
                className: "flex-1 ml-3 whitespace-nowrap",
                children: "Overview"
              })]
            })
          }), /*#__PURE__*/jsx_runtime.jsx("li", {
            children: /*#__PURE__*/(0,jsx_runtime.jsxs)((link_default()), {
              href: "/sources",
              className: `flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${router.pathname === "/sources" ? "bg-gray-700" : ""}`,
              children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
                xmlns: "http://www.w3.org/2000/svg",
                className: "flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white",
                viewBox: "0 0 20 20",
                fill: "currentColor",
                children: /*#__PURE__*/jsx_runtime.jsx("path", {
                  fillRule: "evenodd",
                  d: "M12.316 3.051a1 1 0 01.633 1.265l-4 12a1 1 0 11-1.898-.632l4-12a1 1 0 011.265-.633zM5.707 6.293a1 1 0 010 1.414L3.414 10l2.293 2.293a1 1 0 11-1.414 1.414l-3-3a1 1 0 010-1.414l3-3a1 1 0 011.414 0zm8.586 0a1 1 0 011.414 0l3 3a1 1 0 010 1.414l-3 3a1 1 0 11-1.414-1.414L16.586 10l-2.293-2.293a1 1 0 010-1.414z",
                  clipRule: "evenodd"
                })
              }), /*#__PURE__*/jsx_runtime.jsx("span", {
                className: "flex-1 ml-3 whitespace-nowrap",
                children: "Sources"
              })]
            })
          }), /*#__PURE__*/jsx_runtime.jsx("li", {
            children: /*#__PURE__*/(0,jsx_runtime.jsxs)((link_default()), {
              href: "/destinations",
              className: `flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${router.pathname === "/destinations" ? "bg-gray-700" : ""}`,
              children: [/*#__PURE__*/(0,jsx_runtime.jsxs)("svg", {
                xmlns: "http://www.w3.org/2000/svg",
                className: "flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white",
                viewBox: "0 0 20 20",
                fill: "currentColor",
                children: [/*#__PURE__*/jsx_runtime.jsx("path", {
                  d: "M3 12v3c0 1.657 3.134 3 7 3s7-1.343 7-3v-3c0 1.657-3.134 3-7 3s-7-1.343-7-3z"
                }), /*#__PURE__*/jsx_runtime.jsx("path", {
                  d: "M3 7v3c0 1.657 3.134 3 7 3s7-1.343 7-3V7c0 1.657-3.134 3-7 3S3 8.657 3 7z"
                }), /*#__PURE__*/jsx_runtime.jsx("path", {
                  d: "M17 5c0 1.657-3.134 3-7 3S3 6.657 3 5s3.134-3 7-3 7 1.343 7 3z"
                })]
              }), /*#__PURE__*/jsx_runtime.jsx("span", {
                className: "flex-1 ml-3 whitespace-nowrap",
                children: "Destinations"
              })]
            })
          }), /*#__PURE__*/jsx_runtime.jsx("li", {
            children: /*#__PURE__*/(0,jsx_runtime.jsxs)((link_default()), {
              href: "/collectors",
              className: `flex items-center p-2 text-base font-normal rounded-lg text-white hover:bg-gray-700 ${router.pathname === "/collectors" ? "bg-gray-700" : ""}`,
              children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
                xmlns: "http://www.w3.org/2000/svg",
                className: "flex-shrink-0 w-6 h-6 transition duration-75 text-gray-400 group-hover:text-white",
                viewBox: "0 0 20 20",
                fill: "currentColor",
                children: /*#__PURE__*/jsx_runtime.jsx("path", {
                  fillRule: "evenodd",
                  d: "M3 3a1 1 0 011-1h12a1 1 0 011 1v3a1 1 0 01-.293.707L12 11.414V15a1 1 0 01-.293.707l-2 2A1 1 0 018 17v-5.586L3.293 6.707A1 1 0 013 6V3z",
                  clipRule: "evenodd"
                })
              }), /*#__PURE__*/jsx_runtime.jsx("span", {
                className: "flex-1 ml-3 whitespace-nowrap",
                children: "Collectors"
              })]
            })
          })]
        })
      })]
    })
  });
}
;// CONCATENATED MODULE: external "next/head"
const head_namespaceObject = require("next/head");
var head_default = /*#__PURE__*/__webpack_require__.n(head_namespaceObject);
;// CONCATENATED MODULE: ./pages/_app.tsx
function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }








function App({
  Component,
  pageProps
}) {
  const title = "odigos UI";
  return /*#__PURE__*/(0,jsx_runtime.jsxs)(jsx_runtime.Fragment, {
    children: [/*#__PURE__*/(0,jsx_runtime.jsxs)((head_default()), {
      children: [/*#__PURE__*/jsx_runtime.jsx("title", {
        children: title
      }, "title"), /*#__PURE__*/jsx_runtime.jsx("meta", {
        name: "twitter:title",
        content: title
      }, "twitter:title"), /*#__PURE__*/jsx_runtime.jsx("meta", {
        property: "og:title",
        content: title
      }, "og:title")]
    }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "flex flex-row antialiased bg-white",
      children: [/*#__PURE__*/jsx_runtime.jsx(Sidebar, {}), /*#__PURE__*/jsx_runtime.jsx("div", {
        className: "pt-10 pl-5 w-full text-gray-700 text-xl",
        children: /*#__PURE__*/jsx_runtime.jsx(Component, _objectSpread({}, pageProps))
      })]
    })]
  });
}

/* harmony default export */ const _app = (App);
;// CONCATENATED MODULE: ./node_modules/next/dist/build/webpack/loaders/next-route-loader.js?page=%2F_app&absolutePagePath=private-next-pages%2F_app.tsx&preferredRegion=!

        // Next.js Route Loader
        
        
    

/***/ }),

/***/ 6764:
/***/ (() => {



/***/ }),

/***/ 3280:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/app-router-context.js");

/***/ }),

/***/ 4964:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router-context.js");

/***/ }),

/***/ 1751:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/add-path-prefix.js");

/***/ }),

/***/ 3938:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/format-url.js");

/***/ }),

/***/ 1109:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/is-local-url.js");

/***/ }),

/***/ 8854:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/parse-path.js");

/***/ }),

/***/ 3297:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/remove-trailing-slash.js");

/***/ }),

/***/ 7782:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/router/utils/resolve-href.js");

/***/ }),

/***/ 9232:
/***/ ((module) => {

"use strict";
module.exports = require("next/dist/shared/lib/utils.js");

/***/ }),

/***/ 1853:
/***/ ((module) => {

"use strict";
module.exports = require("next/router");

/***/ }),

/***/ 6689:
/***/ ((module) => {

"use strict";
module.exports = require("react");

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../webpack-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [893,664], () => (__webpack_exec__(3903)));
module.exports = __webpack_exports__;

})();