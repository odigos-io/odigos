"use strict";
(() => {
var exports = {};
exports.id = 77;
exports.ids = [77];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 3126:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ handler)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);
function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }



async function UpdateDest(req, res) {
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  const current = await k8sApi.getNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations", req.query.destname);
  const updated = current.body;
  const {
    spec
  } = updated;
  spec.data = {
    [req.body.destType]: JSON.parse(req.body.values)
  };
  const resp = await k8sApi.replaceNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations", req.query.destname, _objectSpread(_objectSpread({}, updated), {}, {
    spec
  }));
  return res.status(200).json({
    success: true
  });
}

async function DeleteDest(req, res) {
  console.log(`Deleting destination ${req.query.destname}`);
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  await k8sApi.deleteNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations", req.query.destname); // if secret with name req.query.destname exists, delete it

  const coreApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CoreV1Api);
  const secret = await coreApi.readNamespacedSecret(req.query.destname, process.env.CURRENT_NS || "odigos-system");

  if (secret) {
    await coreApi.deleteNamespacedSecret(req.query.destname, process.env.CURRENT_NS || "odigos-system");
  }

  return res.status(200).json({
    success: true
  });
}

async function handler(req, res) {
  if (req.method === "POST") {
    return UpdateDest(req, res);
  } else if (req.method === "DELETE") {
    return DeleteDest(req, res);
  }

  return res.status(405).end(`Method ${req.method} Not Allowed`);
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = (__webpack_exec__(3126));
module.exports = __webpack_exports__;

})();