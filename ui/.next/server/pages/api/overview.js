"use strict";
(() => {
var exports = {};
exports.id = 577;
exports.ids = [577];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 3057:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ handler)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);

async function handler(req, res) {
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  const instAppsResponse = await k8sApi.listClusterCustomObject("odigos.io", "v1alpha1", "instrumentedapplications");
  const appsFound = instAppsResponse.body.items.map(item => {
    return {
      id: item.metadata.uid,
      name: item.metadata.ownerReferences[0].name,
      languages: item.spec.languages?.map(lang => lang.language),
      instrumented: item.spec.languages?.length > 0,
      kind: item.metadata.ownerReferences[0].kind,
      namespace: item.metadata.namespace
    };
  });
  const collectorsResp = await k8sApi.listNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "collectorsgroups");
  const collectors = collectorsResp.body.items.map(item => {
    return {
      name: item.metadata.name,
      ready: item.status.ready
    };
  });
  const destResp = await k8sApi.listNamespacedCustomObject("odigos.io", "v1alpha1", process.env.CURRENT_NS || "odigos-system", "destinations");
  const dests = destResp.body.items.map(item => {
    return {
      id: item.metadata.uid,
      name: item.metadata.name,
      type: item.spec.type,
      signals: item.spec.signals
    };
  });
  return res.status(200).json({
    sources: appsFound,
    collectors: collectors,
    dests: dests
  });
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = (__webpack_exec__(3057));
module.exports = __webpack_exports__;

})();