"use strict";
(() => {
var exports = {};
exports.id = 562;
exports.ids = [562];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 477:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ handler)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);


async function DeleteCollector(req, res) {
  console.log(`deleting collector ${req.body.name}`);
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  await k8sApi.deleteNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "collectorsgroups", req.body.name);
  return res.status(200).json({
    success: true
  });
}

async function GetCollectors(req, res) {
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  const kubeResp = await k8sApi.listNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "collectorsgroups");
  const resp = {
    collectors: kubeResp.body.items.map(item => {
      return {
        name: item.metadata.name,
        ready: item.status.ready
      };
    })
  };
  return res.status(200).json(resp);
}

async function handler(req, res) {
  if (req.method === "GET") {
    await GetCollectors(req, res);
  } else if (req.method === "POST") {
    await DeleteCollector(req, res);
  } else {
    return res.status(405).end(`Method ${req.method} Not Allowed`);
  }
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = (__webpack_exec__(477));
module.exports = __webpack_exports__;

})();