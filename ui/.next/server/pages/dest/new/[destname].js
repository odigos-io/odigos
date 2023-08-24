"use strict";
(() => {
var exports = {};
exports.id = 775;
exports.ids = [775];
exports.modules = {

/***/ 6809:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

// ESM COMPAT FLAG
__webpack_require__.r(__webpack_exports__);

// EXPORTS
__webpack_require__.d(__webpack_exports__, {
  "default": () => (/* reexport */ _destname_),
  "getServerSideProps": () => (/* reexport */ getServerSideProps)
});

// EXTERNAL MODULE: external "react"
var external_react_ = __webpack_require__(6689);
// EXTERNAL MODULE: external "next/router"
var router_ = __webpack_require__(1853);
// EXTERNAL MODULE: ./vendors/index.tsx + 48 modules
var vendors = __webpack_require__(8434);
// EXTERNAL MODULE: ./node_modules/react/jsx-runtime.js
var jsx_runtime = __webpack_require__(5893);
;// CONCATENATED MODULE: ./pages/dest/new/[destname].tsx
function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }







const NewDestination = ({
  destname
}) => {
  const router = (0,router_.useRouter)();
  const vendor = vendors/* default.filter */.ZP.filter(v => v.name === destname)[0];

  if (!vendor) {
    return /*#__PURE__*/jsx_runtime.jsx("div", {
      className: "text-4xl font-medium",
      children: "Observability Vendor Not Found"
    });
  }

  const initialSignalsState = vendor.supportedSignals.reduce((acc, signal) => {
    Object.assign(acc, {
      [signal]: true
    });
    return acc;
  }, {});
  const {
    0: signals,
    1: setSignals
  } = (0,external_react_.useState)(initialSignalsState);
  const fields = vendor.getFields(signals);
  const {
    0: urlValidateError,
    1: setUrlValidateError
  } = (0,external_react_.useState)('');

  const handleSubmit = async event => {
    event.preventDefault();
    var formData = new FormData(event.target);
    var object = {};
    formData.forEach(function (value, key) {
      object[key] = value.toString();
    });
    const JSONdata = JSON.stringify(object); // checking for elastic search url pattern match

    if (object.elasticsearch_url) {
      const regex = /^(http[s]?:\/\/)([a-zA-Z\d\.]{2,})\.([a-zA-Z]{2,})(:9200)\S*/gm;
      const checkregex = regex.test(object.elasticsearch_url);
      checkregex === false ? setUrlValidateError("Your URL must contain the port number 9200") : setUrlValidateError('');
    } // checking for open telementary url pattern match


    if (object.otlp_url) {
      const regex = /^(http[s]?:\/\/)([a-zA-Z\d\.]{2,})\.([a-zA-Z]{2,})(:4317)\S*/gm;
      const checkregex = regex.test(object.otlp_url);
      checkregex === false ? setUrlValidateError("Your URL must contain the port number 4317") : setUrlValidateError('');
    }

    if (urlValidateError === '') {
      const response = await fetch("/api/dests", {
        body: JSONdata,
        headers: {
          "Content-Type": "application/json"
        },
        method: "POST"
      });

      if (response.ok) {
        router.push("/");
      }
    }
  };

  return /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
    className: "flex flex-col",
    children: [/*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
      className: "text-4xl p-8 capitalize text-gray-900",
      children: ["Add new ", vendor.displayName, " destination"]
    }), /*#__PURE__*/jsx_runtime.jsx("div", {
      className: "pl-14 max-w-md",
      children: /*#__PURE__*/(0,jsx_runtime.jsxs)("form", {
        className: "grid grid-cols-1 gap-6",
        onSubmit: handleSubmit,
        name: "newdest",
        children: [vendor.supportedSignals && vendor.supportedSignals.length > 0 && /*#__PURE__*/jsx_runtime.jsx("div", {
          className: "flex flex-row space-x-10 items-center",
          children: vendor.supportedSignals.map(signal => /*#__PURE__*/(0,jsx_runtime.jsxs)("div", {
            className: "space-x-2 items-center",
            hidden: Object.keys(signals).length < 2,
            children: [/*#__PURE__*/jsx_runtime.jsx("input", {
              type: "checkbox",
              name: signal,
              id: signal,
              checked: signals[signal],
              onChange: () => {
                const newSignals = _objectSpread({}, signals);

                newSignals[signal] = !newSignals[signal];
                setSignals(newSignals);
              }
            }), /*#__PURE__*/jsx_runtime.jsx("label", {
              htmlFor: signal,
              children: signal.charAt(0).toUpperCase() + signal.slice(1).toLowerCase()
            })]
          }, signal))
        }), /*#__PURE__*/(0,jsx_runtime.jsxs)("label", {
          className: "block",
          children: [/*#__PURE__*/jsx_runtime.jsx("span", {
            className: "text-gray-700",
            children: "Destination Name"
          }), /*#__PURE__*/jsx_runtime.jsx("input", {
            type: "text",
            id: "name",
            name: "name",
            className: " mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50 ",
            placeholder: "",
            required: true
          })]
        }), fields && fields.map(f => {
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
              required: true
            }), (f.name === 'elasticsearch_url' || f.name === 'otlp_url' || f.name === 'site') && /*#__PURE__*/jsx_runtime.jsx("span", {
              className: "mt-1 text-red-500 text-sm",
              children: urlValidateError
            })]
          }, f.id);
        }), /*#__PURE__*/jsx_runtime.jsx("input", {
          name: "type",
          id: "type",
          hidden: true,
          value: destname,
          readOnly: true
        }), /*#__PURE__*/jsx_runtime.jsx("button", {
          type: "submit",
          className: "mt-4 text-white bg-blue-700 hover:bg-blue-800 focus:ring-4 focus:ring-blue-300 font-medium rounded-lg text-sm px-5 py-2.5 mr-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 focus:outline-none dark:focus:ring-blue-800",
          children: "Create Destination"
        })]
      })
    })]
  });
};

/* harmony default export */ const _destname_ = (NewDestination);
const getServerSideProps = async ({
  query
}) => {
  return {
    props: {
      destname: query.destname
    }
  };
};
;// CONCATENATED MODULE: ./node_modules/next/dist/build/webpack/loaders/next-route-loader.js?page=%2Fdest%2Fnew%2F%5Bdestname%5D&absolutePagePath=private-next-pages%2Fdest%2Fnew%2F%5Bdestname%5D.tsx&preferredRegion=!

        // Next.js Route Loader
        
        
    

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
var __webpack_require__ = require("../../../webpack-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [893,434], () => (__webpack_exec__(6809)));
module.exports = __webpack_exports__;

})();