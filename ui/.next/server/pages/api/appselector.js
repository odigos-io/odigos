"use strict";
(() => {
var exports = {};
exports.id = 18;
exports.ids = [18];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 5794:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ persistApplicationSelection)
/* harmony export */ });
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(276);
/* harmony import */ var _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var _types_apps__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(2786);
function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }



const odigosLabelKey = "odigos-instrumentation";
const odigosLabelValue = "enabled";
async function persistApplicationSelection(req, res) {
  const data = req.body.data;
  const kc = new _kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.KubeConfig();
  kc.loadFromDefault();
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.CoreV1Api);
  await k8sApi.listNamespace().then(async response => {
    response.body.items.forEach(async item => {
      const kubeNamespace = data.namespaces.find(ns => ns.name === item.metadata?.name);

      if (!kubeNamespace || !item.metadata?.name) {
        return;
      }

      const labeledReq = kubeNamespace?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];

      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling namespace", item.metadata?.name);
        item.metadata.labels = _objectSpread(_objectSpread({}, item.metadata?.labels), {}, {
          [odigosLabelKey]: odigosLabelValue
        });
        await k8sApi.replaceNamespace(item.metadata.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling namespace", item.metadata?.name);
        delete item.metadata.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespace(item.metadata.name, item);
      }

      await syncObjectsInNamespace(kc, kubeNamespace);
    });
  });
  return res.status(200).end();
}

async function syncObjectsInNamespace(kc, ns) {
  const k8sApi = kc.makeApiClient(_kubernetes_client_node__WEBPACK_IMPORTED_MODULE_0__.AppsV1Api); // Deployments

  await k8sApi.listNamespacedDeployment(ns.name).then(async response => {
    response.body.items.forEach(async item => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects.find(d => d.name === item.metadata?.name && d.kind.toString() === _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.Deployment */ .O.Deployment])?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];

      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling deployment", item.metadata?.name);
        item.metadata.labels = _objectSpread(_objectSpread({}, item.metadata?.labels), {}, {
          [odigosLabelKey]: odigosLabelValue
        });
        await k8sApi.replaceNamespacedDeployment(item.metadata.name, ns.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling deployment", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedDeployment(item.metadata.name, ns.name, item);
      }
    });
  }); // StatefulSets

  await k8sApi.listNamespacedStatefulSet(ns.name).then(async response => {
    response.body.items.forEach(async item => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects.find(d => d.name === item.metadata?.name && d.kind.toString() === _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.StatefulSet */ .O.StatefulSet])?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];

      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling statefulset", item.metadata?.name);
        item.metadata.labels = _objectSpread(_objectSpread({}, item.metadata?.labels), {}, {
          [odigosLabelKey]: odigosLabelValue
        });
        await k8sApi.replaceNamespacedStatefulSet(item.metadata.name, ns.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling statefulset", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedStatefulSet(item.metadata.name, ns.name, item);
      }
    });
  }); // DaemonSets

  await k8sApi.listNamespacedDaemonSet(ns.name).then(async response => {
    response.body.items.forEach(async item => {
      if (!item.metadata?.name) {
        return;
      }

      const labeledReq = ns.objects.find(d => d.name === item.metadata?.name && d.kind.toString() === _types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind */ .O[_types_apps__WEBPACK_IMPORTED_MODULE_1__/* .AppKind.DaemonSet */ .O.DaemonSet])?.labeled;
      const odigosLabeled = item.metadata?.labels?.[odigosLabelKey];

      if (labeledReq && odigosLabeled !== odigosLabelValue) {
        console.log("labeling daemonset", item.metadata?.name);
        item.metadata.labels = _objectSpread(_objectSpread({}, item.metadata?.labels), {}, {
          [odigosLabelKey]: odigosLabelValue
        });
        await k8sApi.replaceNamespacedDaemonSet(item.metadata.name, ns.name, item);
      } else if (!labeledReq && odigosLabeled === odigosLabelValue) {
        console.log("unlabeling daemonset", item.metadata.name);
        delete item.metadata?.labels?.[odigosLabelKey];
        await k8sApi.replaceNamespacedDaemonSet(item.metadata.name, ns.name, item);
      }
    });
  });
}

/***/ }),

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

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = (__webpack_exec__(5794));
module.exports = __webpack_exports__;

})();