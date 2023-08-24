"use strict";
(() => {
var exports = {};
exports.id = 291;
exports.ids = [291];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 8918:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ handler)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var _utils_kube__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(9302);


async function handler(req, res) {
  const kubeObjects = await (0,_utils_kube__WEBPACK_IMPORTED_MODULE_1__/* .GetAllKubernetesObjects */ .Y)();

  if (kubeObjects instanceof Error) {
    return res.status(500).json({
      message: kubeObjects.message
    });
  }

  const enrichedApps = await enrichKubeObjectsWithRuntime(kubeObjects);

  if (enrichedApps instanceof Error) {
    return res.status(500).json({
      message: enrichedApps.message
    });
  }

  return res.status(200).json({
    apps: enrichedApps
  });
}

async function enrichKubeObjectsWithRuntime(kubeObjects) {
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CustomObjectsApi);
  const response = await k8sApi.listClusterCustomObject("odigos.io", "v1alpha1", "instrumentedapplications");
  const enrichedApps = kubeObjects.namespaces.flatMap(ns => {
    return ns.objects.map(obj => {
      const app = response.body.items.find(item => {
        return obj.kind.toString().toLowerCase() === item.metadata.ownerReferences[0].kind.toString().toLowerCase() && obj.name.toLowerCase() === item.metadata.ownerReferences[0].name.toLowerCase();
      });

      if (!app) {
        return {
          id: `${obj.kind}-${obj.name}`,
          name: obj.name,
          namespace: ns.name,
          kind: obj.kind,
          instrumented: false,
          languages: []
        };
      }

      return {
        id: app.metadata.uid,
        name: obj.name,
        namespace: ns.name,
        kind: obj.kind,
        instrumented: true,
        languages: app.spec.languages.map(lang => lang.language)
      };
    });
  });
  return enrichedApps;
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [302], () => (__webpack_exec__(8918)));
module.exports = __webpack_exports__;

})();