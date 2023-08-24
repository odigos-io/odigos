"use strict";
(() => {
var exports = {};
exports.id = 626;
exports.ids = [626];
exports.modules = {

/***/ 8174:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* reexport */ _destname_),
  "getServerSideProps": () => (/* reexport */ getServerSideProps)
});

// EXTERNAL MODULE: external "next/router"
var router_ = __webpack_require__(1853);
// EXTERNAL MODULE: ./vendors/index.tsx + 48 modules
var vendors = __webpack_require__(8434);
// EXTERNAL MODULE: external "@kubernetes/client-node"
var client_node_ = __webpack_require__(276);
// EXTERNAL MODULE: ./node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(5893);
;// CONCATENATED MODULE: ./pages/dest/[destname].tsx






const NewDestination = ({
  destType,
  currentValues,
  destName,
  signals
}) => {
  const router = (0,router_.useRouter)();
  const vendor = vendors/* default.filter */.ZP.filter(v => v.name === destType)[0];

  if (!vendor) {
    return /*#__PURE__*/jsx_runtime.jsx("div", {
      className: "text-4xl font-medium",
      children: "Observability Vendor Not Found"
    });
  }

  const fields = vendor.getFields(signals);
  console.log(fields);

  const deleteDest = async () => {
    const response = await fetch(`/api/dest/${destName}`, {
      method: "DELETE"
    });

    if (response.ok) {
      router.push("/destinations");
    }
  };

  const updateDest = async event => {
    event.preventDefault();
    var formData = new FormData(event.target);
    var object = {};
    formData.forEach(function (value, key) {
      object[key] = value.toString();
    });
    const JSONdata = JSON.stringify(object);
    const response = await fetch(`/api/dest/${destName}`, {
      body: JSON.stringify({
        destType,
        values: JSONdata
      }),
      headers: {
        "Content-Type": "application/json"
      },
      method: "POST"
    });

    if (response.ok) {
      router.push("/destinations");
    }
  };

  return /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
    className: "flex flex-col",
    children: [/*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "text-4xl p-8 capitalize text-gray-900",
      children: ["Edit Destination: ", destName]
    }), /*#__PURE__*/jsx_runtime.jsx("div", {
      className: "pl-14 max-w-md",
      children: /*#__PURE__*/(0,jsx_runtime.jsxs)("form", {
        className: "grid grid-cols-1 gap-6",
        onSubmit: updateDest,
        name: "newdest",
        children: [fields && fields.filter(f => currentValues.hasOwnProperty(f.name)).map(f => {
          return /*#__PURE__*/(0,jsx_runtime.jsxs)("label", {
            className: "block",
            children: [/*#__PURE__*/jsx_runtime.jsx("span", {
              className: "text-gray-700",
              children: f.displayName
            }), /*#__PURE__*/jsx_runtime.jsx("input", {
              id: f.id,
              name: f.name,
              type: f.type,
              className: " mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50 ",
              placeholder: "",
              required: true,
              defaultValue: currentValues[f.name]
            })]
          }, f.id);
        }), /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
          className: "mx-auto flex flex-row justify-between",
          children: [/*#__PURE__*/jsx_runtime.jsx("button", {
            type: "submit",
            className: "mt-4 text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-8 py-2.5 mr-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800",
            children: "Save Destination"
          }), /*#__PURE__*/jsx_runtime.jsx("button", {
            type: "button",
            onClick: deleteDest,
            className: "mt-4 text-white bg-red-700 hover:bg-red-800 focus:ring-4 focus:ring-red-300 font-medium rounded-lg text-sm px-5 py-2.5 mr-2 mb-2 dark:bg-red-600 dark:hover:bg-red-700 focus:outline-none dark:focus:ring-red-800",
            children: "Delete Destination"
          })]
        })]
      })
    })]
  });
};

/* harmony default export */ const _destname_ = (NewDestination);
const getServerSideProps = async ({
  query
}) => {
  const {
    destname
  } = query;
  const kc = new client_node_.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(client_node_.CustomObjectsApi);
  const response = await k8sApi.getNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations", destname);
  const {
    spec
  } = response.body;
  const vendor = vendors/* default.find */.ZP.find(v => v.name === spec.type);

  if (!vendor) {
    return {
      props: {
        destname: "",
        currentValues: {}
      }
    };
  }

  const props = {
    destName: destname,
    destType: spec.type,
    currentValues: vendor.mapDataToFields(spec.data),
    signals: spec.signals.reduce((acc, signal) => {
      Object.assign(acc, {
        [signal]: true
      });
      return acc;
    }, {})
  };
  return {
    props
  };
};
;// CONCATENATED MODULE: ./node_modules/next/dist/build/webpack/loaders/next-route-loader.js?page=%2Fdest%2F%5Bdestname%5D&absolutePagePath=private-next-pages%2Fdest%2F%5Bdestname%5D.tsx&preferredRegion=!

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
var __webpack_require__ = require("../../webpack-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [893,434], () => (__webpack_exec__(8174)));
module.exports = __webpack_exports__;

})();