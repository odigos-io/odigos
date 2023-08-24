"use strict";
(() => {
var exports = {};
exports.id = 760;
exports.ids = [760];
exports.modules = {

/***/ 6221:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* reexport */ _name_),
  "getServerSideProps": () => (/* reexport */ getServerSideProps)
});

// EXTERNAL MODULE: external "next/router"
var router_ = __webpack_require__(1853);
// EXTERNAL MODULE: external "@kubernetes/client-node"
var client_node_ = __webpack_require__(276);
// EXTERNAL MODULE: external "react"
var external_react_ = __webpack_require__(6689);
// EXTERNAL MODULE: ./node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(5893);
;// CONCATENATED MODULE: ./pages/source/[namespace]/[kind]/[name].tsx






const EditAppPage = ({
  enabled,
  reportedName
}) => {
  const router = (0,router_.useRouter)();
  const {
    name,
    kind,
    namespace
  } = router.query;
  const {
    0: isEnabled,
    1: setIsEnabled
  } = (0,external_react_.useState)(enabled);
  const {
    0: updatedReportedName,
    1: setUpdatedReportedName
  } = (0,external_react_.useState)(reportedName);

  const updateApp = async () => {
    const resp = await fetch(`/api/source/${namespace}/${kind}/${name}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        enabled: isEnabled,
        reportedName: updatedReportedName
      })
    });

    if (resp.ok) {
      router.push("/sources");
    }
  };

  return /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
    className: "flex flex-col w-fit",
    children: [/*#__PURE__*/jsx_runtime.jsx("div", {
      className: "text-4xl font-medium",
      children: name
    }), /*#__PURE__*/jsx_runtime.jsx("div", {
      children: /*#__PURE__*/(0,jsx_runtime.jsxs)("label", {
        className: "block mt-6",
        children: [/*#__PURE__*/jsx_runtime.jsx("span", {
          className: "text-gray-700",
          children: "Reported Name"
        }), /*#__PURE__*/jsx_runtime.jsx("input", {
          name: "reportedName",
          type: "text",
          className: " mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50 ",
          placeholder: "",
          required: true,
          defaultValue: reportedName,
          onChange: e => {
            setUpdatedReportedName(e.target.value);
          }
        })]
      })
    }), /*#__PURE__*/(0,jsx_runtime.jsxs)("label", {
      htmlFor: "default-toggle",
      className: "mt-6 inline-flex relative items-center cursor-pointer",
      children: [/*#__PURE__*/jsx_runtime.jsx("input", {
        type: "checkbox",
        value: "",
        id: "default-toggle",
        className: "sr-only peer",
        onChange: () => {
          setIsEnabled(!isEnabled);
        },
        checked: isEnabled
      }), /*#__PURE__*/jsx_runtime.jsx("div", {
        className: "w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
      }), /*#__PURE__*/jsx_runtime.jsx("span", {
        className: "ml-3 text-md font-medium text-gray-900",
        children: "Enabled"
      })]
    }), /*#__PURE__*/jsx_runtime.jsx("button", {
      type: "submit",
      disabled: isEnabled === enabled && reportedName === updatedReportedName,
      onClick: updateApp,
      className: "mt-4 disabled:cursor-not-allowed disabled:hover:bg-gray-500 disabled:bg-gray-500 text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 mr-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800",
      children: "Save Changes"
    })]
  });
};

const getServerSideProps = async ({
  query
}) => {
  const {
    name,
    kind,
    namespace
  } = query;
  const kc = new client_node_.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(client_node_.AppsV1Api);
  const reportedNameAnootation = "odigos.io/reported-name";
  var obj = null;
  var instrumented = false;
  var reportedName = name;

  switch (kind) {
    case "deployment":
      obj = await getDeployment(name, namespace, kc);
      break;

    case "statefulset":
      obj = await getStatefulSet(name, namespace, kc);
      break;

    case "daemonset":
      obj = await getDaemonSet(name, namespace, kc);
      break;

    default:
      return {
        redirect: {
          destination: "/",
          permanent: false
        }
      };
  }

  if (!obj) {
    return {
      redirect: {
        destination: "/",
        permanent: false
      }
    };
  }

  if (obj?.metadata?.annotations?.[reportedNameAnootation]) {
    reportedName = obj?.metadata?.annotations?.[reportedNameAnootation];
  }

  instrumented = isLabeled(obj?.metadata?.labels);

  if (!instrumented) {
    instrumented = await isNamespaceLabeled(namespace, kc);
  }

  return {
    props: {
      enabled: instrumented,
      reportedName: reportedName
    }
  };
};

async function getDeployment(name, namespace, kc) {
  const kubeClient = kc.makeApiClient(client_node_.AppsV1Api);
  const resp = await kubeClient.readNamespacedDeployment(name, namespace);

  if (!resp) {
    return null;
  }

  return resp.body;
}

async function getStatefulSet(name, namespace, kc) {
  const kubeClient = kc.makeApiClient(client_node_.AppsV1Api);
  const resp = await kubeClient.readNamespacedStatefulSet(name, namespace);

  if (!resp) {
    return null;
  }

  return resp.body;
}

async function getDaemonSet(name, namespace, kc) {
  const kubeClient = kc.makeApiClient(client_node_.AppsV1Api);
  const resp = await kubeClient.readNamespacedDaemonSet(name, namespace);

  if (!resp) {
    return null;
  }

  return resp.body;
}

function isLabeled(labels) {
  return labels && labels["odigos-instrumentation"] === "enabled";
}

async function isNamespaceLabeled(name, kc) {
  const kubeClient = kc.makeApiClient(client_node_.CoreV1Api);
  const resp = await kubeClient.readNamespace(name);

  if (!resp || !resp.body.metadata) {
    return false;
  }

  return isLabeled(resp.body.metadata.labels);
}

/* harmony default export */ const _name_ = (EditAppPage);
;// CONCATENATED MODULE: ./node_modules/next/dist/build/webpack/loaders/next-route-loader.js?page=%2Fsource%2F%5Bnamespace%5D%2F%5Bkind%5D%2F%5Bname%5D&absolutePagePath=private-next-pages%2Fsource%2F%5Bnamespace%5D%2F%5Bkind%5D%2F%5Bname%5D.tsx&preferredRegion=!

        // Next.js Route Loader
        
        
    

/***/ }),

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 1853:
/***/ ((module) => {

module.exports = require("next/router");

/***/ }),

/***/ 6689:
/***/ ((module) => {

module.exports = require("react");

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../../../webpack-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [893], () => (__webpack_exec__(6221)));
module.exports = __webpack_exports__;

})();