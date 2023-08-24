"use strict";
(() => {
var exports = {};
exports.id = 153;
exports.ids = [153];
exports.modules = {

/***/ 276:
/***/ ((module) => {

module.exports = require("@kubernetes/client-node");

/***/ }),

/***/ 2163:
/***/ ((__unused_webpack_module, __webpack_exports__, __webpack_require__) => {

__webpack_require__.r(__webpack_exports__);
/* harmony export */ __webpack_require__.d(__webpack_exports__, {
/* harmony export */   "default": () => (/* binding */ handler)
/* harmony export */ });
/* harmony import */ var _utils_kube__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(9302);

async function handler(req, res) {
  return res.status(200).json(await (0,_utils_kube__WEBPACK_IMPORTED_MODULE_0__/* .GetAllKubernetesObjects */ .Y)());
}

/***/ })

};
;

// load runtime
var __webpack_require__ = require("../../../webpack-api-runtime.js");
__webpack_require__.C(exports);
var __webpack_exec__ = (moduleId) => (__webpack_require__(__webpack_require__.s = moduleId))
var __webpack_exports__ = __webpack_require__.X(0, [302], () => (__webpack_exec__(2163)));
module.exports = __webpack_exports__;

})();