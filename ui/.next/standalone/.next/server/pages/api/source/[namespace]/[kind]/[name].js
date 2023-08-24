"use strict";
(() => {
var exports = {};
exports.id = 968;
exports.ids = [968];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 1985:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ UpdateSource)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);

async function UpdateSource(req, res) {
  if (!req.query.kind || typeof req.query.kind !== "string") {
    return res.status(400).json({
      message: "kind is required"
    });
  }

  console.log(`updating source ${req.query.name} enabled: ${req.body.enabled}`);
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.AppsV1Api);

  switch (req.query.kind.toLowerCase()) {
    case "deployment":
      await updateDeployment(k8sApi, req.query.namespace, req.query.name, req.body.enabled, req.body.reportedName);
      break;

    case "statefulset":
      await updateStatefulSet(k8sApi, req.query.namespace, req.query.name, req.body.enabled, req.body.reportedName);
      break;

    case "daemonset":
      await updateDaemonSet(k8sApi, req.query.namespace, req.query.name, req.body.enabled, req.body.reportedName);
      break;

    default:
      return res.status(400).json({
        message: "kind is not supported"
      });
  }

  return res.status(200).json({
    message: "success"
  });
}

async function updateDeployment(k8sApi, namespace, name, enabled, reportedName) {
  const resp = await k8sApi.readNamespacedDeployment(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedDeployment(name, namespace, resp.body);
}

async function updateStatefulSet(k8sApi, namespace, name, enabled, reportedName) {
  const resp = await k8sApi.readNamespacedStatefulSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedStatefulSet(name, namespace, resp.body);
}

async function updateDaemonSet(k8sApi, namespace, name, enabled, reportedName) {
  const resp = await k8sApi.readNamespacedDaemonSet(name, namespace);
  resp.body.metadata.labels = resp.body.metadata.labels || {};
  resp.body.metadata.labels["odigos-instrumentation"] = enabled ? "enabled" : "disabled";
  resp.body.metadata.annotations = resp.body.metadata.annotations || {};
  resp.body.metadata.annotations["odigos.io/reported-name"] = reportedName;
  await k8sApi.replaceNamespacedDaemonSet(name, namespace, resp.body);
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../../../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = (__webpack_exec__(1985));
module.exports = __webpack_exports__;

})();