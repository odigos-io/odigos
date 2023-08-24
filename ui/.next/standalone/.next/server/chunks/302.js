"use strict";
exports.id = 302;
exports.ids = [302];
exports.modules = {

/***/ 2786:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "O": () => (/* binding */ AppKind)
/* harmony export */ });
let AppKind;

(function (AppKind) {
  AppKind[AppKind["Deployment"] = 0] = "Deployment";
  AppKind[AppKind["StatefulSet"] = 1] = "StatefulSet";
  AppKind[AppKind["DaemonSet"] = 2] = "DaemonSet";
})(AppKind || (AppKind = {}));

/***/ }),

/***/ 9302:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "Y": () => (/* binding */ GetAllKubernetesObjects)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var _types_apps__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(2786);


async function GetAllKubernetesObjects() {
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CoreV1Api);
  const namespacesResponse = await k8sApi.listNamespace();
  const objectsByNamespace = {};

  for (const namespace of namespacesResponse.body.items) {
    const objectsInNamespace = await getObjectsInNamespace(namespace.metadata.name, kc);
    objectsByNamespace[namespace.metadata.name] = objectsInNamespace;
  }

  const namespaces = namespacesResponse.body.items.map(item => {
    return {
      name: item.metadata.name,
      labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled",
      objects: objectsByNamespace[item.metadata.name]
    };
  });
  return {
    namespaces: namespaces
  };
}

async function getObjectsInNamespace(namespace, kc) {
  // Get deployments, statefulsets and daemonsets
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.AppsV1Api);
  const deploymentsResponse = await k8sApi.listNamespacedDeployment(namespace);
  const deployments = deploymentsResponse.body.items.map(item => {
    return {
      name: item.metadata.name,
      kind: _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.Deployment */ .O.Deployment],
      instances: item.status.availableReplicas || 0,
      labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled"
    };
  });
  const statefulsetsResponse = await k8sApi.listNamespacedStatefulSet(namespace);
  const statefulsets = statefulsetsResponse.body.items.map(item => {
    return {
      name: item.metadata.name,
      kind: _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.StatefulSet */ .O.StatefulSet],
      instances: item.status.readyReplicas || 0,
      labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled"
    };
  });
  const daemonsetsResponse = await k8sApi.listNamespacedDaemonSet(namespace);
  const daemonsets = daemonsetsResponse.body.items.map(item => {
    return {
      name: item.metadata.name,
      kind: _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.DaemonSet */ .O.DaemonSet],
      instances: item.status.numberReady || 0,
      labeled: item.metadata.labels && item.metadata.labels["odigos-instrumentation"] === "enabled"
    };
  });
  return deployments.concat(statefulsets).concat(daemonsets);
}

/***/ })

};
;