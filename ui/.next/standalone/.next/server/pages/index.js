"use strict";
(() => {
var exports = {};
exports.id = 405;
exports.ids = [405];
exports.modules = {

/***/ 7413:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* reexport */ pages),
  "getServerSideProps": () => (/* reexport */ getServerSideProps)
});

// EXTERNAL MODULE: external "swr"
var external_swr_ = __webpack_require__(549);
var external_swr_default = /*#__PURE__*/__webpack_require__.n(external_swr_);
// EXTERNAL MODULE: ./components/Loading.tsx + 1 modules
var Loading = __webpack_require__(7046);
// EXTERNAL MODULE: ./utils/icons.tsx + 7 modules
var icons = __webpack_require__(9916);
// EXTERNAL MODULE: ./vendors/index.tsx + 48 modules
var vendors = __webpack_require__(8434);
// EXTERNAL MODULE: ./node_modules/next/link.js
var next_link = __webpack_require__(1664);
var link_default = /*#__PURE__*/__webpack_require__.n(next_link);
// EXTERNAL MODULE: external "@kubernetes/client-node"
var client_node_ = __webpack_require__(276);
// EXTERNAL MODULE: ./node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(5893);
;// CONCATENATED MODULE: ./pages/index.tsx









const Home = () => {
  const fetcher = args => fetch(args).then(res => res.json());

  const {
    data,
    error
  } = external_swr_default()("/api/overview", fetcher);
  if (error) return /*#__PURE__*/jsx_runtime.jsx("div", {
    children: "failed to load"
  });
  if (!data) return /*#__PURE__*/jsx_runtime.jsx(Loading/* default */.Z, {});
  const appsByLang = data.sources.reduce((acc, app) => {
    const lang = app.languages ? app.languages[0] : "unrecognized";

    if (!acc[lang]) {
      acc[lang] = [];
    }

    acc[lang].push(app);
    return acc;
  }, {});
  const totalCollectors = data.collectors.length;
  const readyCollectors = data.collectors.filter(c => c.ready).length;
  return /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
    className: "w-full h-full",
    children: [/*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "h-1/3 w-full",
      children: [/*#__PURE__*/jsx_runtime.jsx("div", {
        className: "text-4xl font-medium",
        children: "Sources"
      }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
        className: "mt-4 grid grid-flow-col grid-rows-4 gap-x-10 gap-y-2 w-fit",
        children: [appsByLang["unrecognized"]?.length > 0 && /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          className: "flex flex-row items-center space-x-2",
          children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
            xmlns: "http://www.w3.org/2000/svg",
            className: "h-8 w-8 text-red-500",
            viewBox: "0 0 20 20",
            fill: "currentColor",
            children: /*#__PURE__*/jsx_runtime.jsx("path", {
              fillRule: "evenodd",
              d: "M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z",
              clipRule: "evenodd"
            })
          }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
            className: "font-bold",
            children: [appsByLang["unrecognized"].length, " unrecognized applications"]
          })]
        }), Object.keys(appsByLang).filter(lang => lang !== "unrecognized").map(lang => /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          className: "flex flex-row items-center space-x-2",
          children: [/*#__PURE__*/jsx_runtime.jsx("div", {
            children: (0,icons/* getLangIcon */.j)(lang, "w-8 h-8")
          }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
            className: "",
            children: [/*#__PURE__*/jsx_runtime.jsx("span", {
              className: "text-bold text-2xl",
              children: appsByLang[lang].length
            }), " ", lang, " applications"]
          })]
        }, lang))]
      })]
    }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "h-1/3 w-full",
      children: [/*#__PURE__*/jsx_runtime.jsx("div", {
        className: "text-4xl font-medium",
        children: "Destinations"
      }), data.dests && data.dests.length > 0 ? /*#__PURE__*/jsx_runtime.jsx("div", {
        className: "mt-4 grid grid-flow-col grid-rows-4 gap-x-10 gap-y-2 w-fit",
        children: data.dests.map(dest => /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          className: "flex flex-row items-center space-x-2",
          children: [vendors/* default.find */.ZP.find(v => v.name === dest.type)?.getLogo({
            className: "w-8 h-8"
          }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
            className: "",
            children: ["Sending", " ", /*#__PURE__*/jsx_runtime.jsx("span", {
              children: dest.signals.map(s => s.toLowerCase()).join(", ")
            }), " ", "to ", /*#__PURE__*/jsx_runtime.jsx("span", {
              className: "font-bold",
              children: dest.name
            })]
          })]
        }, dest.id))
      }) : /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
        className: "mt-4 w-fit flex p-4 mb-4 text-sm rounded-lg bg-yellow-200 text-yellow-700",
        role: "alert",
        children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
          className: "inline flex-shrink-0 mr-3 w-5 h-5",
          fill: "currentColor",
          viewBox: "0 0 20 20",
          xmlns: "http://www.w3.org/2000/svg",
          children: /*#__PURE__*/jsx_runtime.jsx("path", {
            "fill-rule": "evenodd",
            d: "M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z",
            "clip-rule": "evenodd"
          })
        }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          children: [/*#__PURE__*/jsx_runtime.jsx("span", {
            className: "font-medium",
            children: "No destinations configured!"
          }), " ", /*#__PURE__*/jsx_runtime.jsx((link_default()), {
            href: "/dest/new",
            className: "font-medium underline",
            children: "Click here to add a destination"
          })]
        })]
      })]
    }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "h-1/3 w-full",
      children: [/*#__PURE__*/jsx_runtime.jsx("div", {
        className: "text-4xl font-medium",
        children: "Collectors"
      }), totalCollectors > 0 ? /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
        className: "mt-4 flex flex-row space-x-2 items-center",
        children: [readyCollectors === totalCollectors ? /*#__PURE__*/jsx_runtime.jsx("svg", {
          xmlns: "http://www.w3.org/2000/svg",
          className: "h-8 w-8 text-green-600",
          viewBox: "0 0 20 20",
          fill: "currentColor",
          children: /*#__PURE__*/jsx_runtime.jsx("path", {
            fillRule: "evenodd",
            d: "M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z",
            clipRule: "evenodd"
          })
        }) : /*#__PURE__*/jsx_runtime.jsx("svg", {
          xmlns: "http://www.w3.org/2000/svg",
          className: "h-8 w-8 text-red-500",
          viewBox: "0 0 20 20",
          fill: "currentColor",
          children: /*#__PURE__*/jsx_runtime.jsx("path", {
            fillRule: "evenodd",
            d: "M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z",
            clipRule: "evenodd"
          })
        }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          className: "font-bold",
          children: [readyCollectors, " / ", totalCollectors, " collectors ready"]
        })]
      }) : /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
        className: "w-fit mt-4 flex p-4 mb-4 text-sm rounded-lg bg-blue-200 text-blue-800",
        role: "alert",
        children: [/*#__PURE__*/jsx_runtime.jsx("svg", {
          className: "inline flex-shrink-0 mr-3 w-5 h-5",
          fill: "currentColor",
          viewBox: "0 0 20 20",
          xmlns: "http://www.w3.org/2000/svg",
          children: /*#__PURE__*/jsx_runtime.jsx("path", {
            "fill-rule": "evenodd",
            d: "M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z",
            "clip-rule": "evenodd"
          })
        }), /*#__PURE__*/jsx_runtime.jsx("div", {
          className: "font-medium max-w-md",
          children: "No collectors running. Odigos will automaticly deploy collectors once a destination is configured."
        })]
      })]
    })]
  });
};

const getServerSideProps = async ({
  query
}) => {
  const kc = new client_node_.KubeConfig();
  kc.loadFromDefault();
  const foundLabeled = await isSomethingLabeled(kc);

  if (!foundLabeled) {
    return {
      redirect: {
        destination: "/setup",
        permanent: false
      }
    };
  } // Check if any destination is configured


  const kubeCrdApi = kc.makeApiClient(client_node_.CustomObjectsApi);
  const destinations = await kubeCrdApi.listNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations");

  if (destinations.body.items && destinations.body.items.length === 0) {
    return {
      redirect: {
        destination: "/dest/new",
        permanent: false
      }
    };
  }

  return {
    props: {}
  };
};

async function isSomethingLabeled(kc) {
  // Check if there is any namespace labeled with odigos
  const k8sApi = kc.makeApiClient(client_node_.CoreV1Api);
  const namespaces = await k8sApi.listNamespace();
  const odigosNamespaces = namespaces.body.items.filter(ns => {
    return ns.metadata?.labels && ns.metadata?.labels["odigos-instrumentation"] === "enabled";
  });

  if (odigosNamespaces.length > 0) {
    return true;
  } // Check if there is any deployment labeled with odigos


  const k8sAppsApi = kc.makeApiClient(client_node_.AppsV1Api);
  const deployments = await k8sAppsApi.listDeploymentForAllNamespaces();
  const odigosDeployments = deployments.body.items.filter(d => {
    return d.metadata?.labels && d.metadata?.labels["odigos-instrumentation"] === "enabled";
  });

  if (odigosDeployments.length > 0) {
    return true;
  } // Check if there is any daemonset labeled with odigos


  const daemonsets = await k8sAppsApi.listDaemonSetForAllNamespaces();
  const odigosDaemonsets = daemonsets.body.items.filter(d => {
    return d.metadata?.labels && d.metadata?.labels["odigos-instrumentation"] === "enabled";
  });

  if (odigosDaemonsets.length > 0) {
    return true;
  } // Check if there is any statefulset labeled with odigos


  const statefulsets = await k8sAppsApi.listStatefulSetForAllNamespaces();
  const odigosStatefulsets = statefulsets.body.items.filter(d => {
    return d.metadata?.labels && d.metadata?.labels["odigos-instrumentation"] === "enabled";
  });

  if (odigosStatefulsets.length > 0) {
    return true;
  }

  return false;
}

/* harmony default export */ const pages = (Home);
;// CONCATENATED MODULE: ./node_modules/next/dist/build/webpack/loaders/next-route-loader.js?page=%2F&absolutePagePath=private-next-pages%2Findex.tsx&preferredRegion=!

        // Next.js Route Loader
        
        
    

/***/ }),

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 3280:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/app-router-context.js");

/***/ }),

/***/ 4964:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router-context.js");

/***/ }),

/***/ 1751:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/add-path-prefix.js");

/***/ }),

/***/ 3938:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/format-url.js");

/***/ }),

/***/ 1109:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/is-local-url.js");

/***/ }),

/***/ 8854:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/parse-path.js");

/***/ }),

/***/ 3297:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/remove-trailing-slash.js");

/***/ }),

/***/ 7782:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/router/utils/resolve-href.js");

/***/ }),

/***/ 9232:
/***/ ((module) => {

module.exports = require("next/dist/shared/lib/utils.js");

/***/ }),

/***/ 6689:
/***/ ((module) => {

module.exports = require("react");

/***/ }),

/***/ 549:
/***/ ((module) => {

module.exports = require("swr");

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../webpack-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [893,664,434,46,524,916], () => (__webpack_exec__(7413)));
module.exports = __webpack_exports__;

})();