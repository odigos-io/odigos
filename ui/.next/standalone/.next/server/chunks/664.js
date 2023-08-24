exports.id = 664;
exports.ids = [664];
exports.modules = {

/***/ 598:
/***/ ((module, exports, __webpack_require__) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "addBasePath", ({
  enumerable: true,
  get: function () {
    return addBasePath;
  }
}));

const _addpathprefix = __webpack_require__(1751);

const _normalizetrailingslash = __webpack_require__(1485);

const basePath =  false || "";

function addBasePath(path, required) {
  if (false) {}

  return (0, _normalizetrailingslash.normalizePathTrailingSlash)((0, _addpathprefix.addPathPrefix)(path, basePath));
}

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 8298:
/***/ ((module, exports, __webpack_require__) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "addLocale", ({
  enumerable: true,
  get: function () {
    return addLocale;
  }
}));

const _normalizetrailingslash = __webpack_require__(1485);

const addLocale = function (path) {
  for (var _len = arguments.length, args = new Array(_len > 1 ? _len - 1 : 0), _key = 1; _key < _len; _key++) {
    args[_key - 1] = arguments[_key];
  }

  if (false) {}

  return path;
};

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 6846:
/***/ ((module, exports) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
0 && (0);

function _export(target, all) {
  for (var name in all) Object.defineProperty(target, name, {
    enumerable: true,
    get: all[name]
  });
}

_export(exports, {
  PrefetchKind: function () {
    return PrefetchKind;
  },
  ACTION_REFRESH: function () {
    return ACTION_REFRESH;
  },
  ACTION_NAVIGATE: function () {
    return ACTION_NAVIGATE;
  },
  ACTION_RESTORE: function () {
    return ACTION_RESTORE;
  },
  ACTION_SERVER_PATCH: function () {
    return ACTION_SERVER_PATCH;
  },
  ACTION_PREFETCH: function () {
    return ACTION_PREFETCH;
  },
  ACTION_FAST_REFRESH: function () {
    return ACTION_FAST_REFRESH;
  },
  ACTION_SERVER_ACTION: function () {
    return ACTION_SERVER_ACTION;
  }
});

const ACTION_REFRESH = "refresh";
const ACTION_NAVIGATE = "navigate";
const ACTION_RESTORE = "restore";
const ACTION_SERVER_PATCH = "server-patch";
const ACTION_PREFETCH = "prefetch";
const ACTION_FAST_REFRESH = "fast-refresh";
const ACTION_SERVER_ACTION = "server-action";
var PrefetchKind;

(function (PrefetchKind) {
  PrefetchKind["AUTO"] = "auto";
  PrefetchKind["FULL"] = "full";
  PrefetchKind["TEMPORARY"] = "temporary";
})(PrefetchKind || (PrefetchKind = {}));

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 6612:
/***/ ((module, exports) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "getDomainLocale", ({
  enumerable: true,
  get: function () {
    return getDomainLocale;
  }
}));
const basePath = (/* unused pure expression or super */ null && ( false || ""));

function getDomainLocale(path, locale, locales, domainLocales) {
  if (false) {} else {
    return false;
  }
}

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 9771:
/***/ ((module, exports, __webpack_require__) => {

"use client";
"use strict";

const _excluded = ["href", "as", "children", "prefetch", "passHref", "replace", "shallow", "scroll", "locale", "onClick", "onMouseEnter", "onTouchStart", "legacyBehavior"];

function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }

function _objectWithoutProperties(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }

function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }

Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "default", ({
  enumerable: true,
  get: function () {
    return _default;
  }
}));

const _interop_require_default = __webpack_require__(167);

const _react = /*#__PURE__*/_interop_require_default._(__webpack_require__(6689));

const _resolvehref = __webpack_require__(7782);

const _islocalurl = __webpack_require__(1109);

const _formaturl = __webpack_require__(3938);

const _utils = __webpack_require__(9232);

const _addlocale = __webpack_require__(8298);

const _routercontext = __webpack_require__(4964);

const _approutercontext = __webpack_require__(3280);

const _useintersection = __webpack_require__(4203);

const _getdomainlocale = __webpack_require__(6612);

const _addbasepath = __webpack_require__(598);

const _routerreducertypes = __webpack_require__(6846);

const prefetched = new Set();

function prefetch(router, href, as, options, appOptions, isAppRouter) {
  if (true) {
    return;
  } // app-router supports external urls out of the box so it shouldn't short-circuit here as support for e.g. `replace` is added in the app-router.


  if (!isAppRouter && !(0, _islocalurl.isLocalURL)(href)) {
    return;
  } // We should only dedupe requests when experimental.optimisticClientCache is
  // disabled.


  if (!options.bypassPrefetchedCheck) {
    const locale = // Let the link's locale prop override the default router locale.
    typeof options.locale !== "undefined" ? options.locale : "locale" in router ? router.locale : undefined;
    const prefetchedKey = href + "%" + as + "%" + locale; // If we've already fetched the key, then don't prefetch it again!

    if (prefetched.has(prefetchedKey)) {
      return;
    } // Mark this URL as prefetched.


    prefetched.add(prefetchedKey);
  }

  const prefetchPromise = isAppRouter ? router.prefetch(href, appOptions) : router.prefetch(href, as, options); // Prefetch the JSON page if asked (only in the client)
  // We need to handle a prefetch error here since we may be
  // loading with priority which can reject but we don't
  // want to force navigation since this is only a prefetch

  Promise.resolve(prefetchPromise).catch(err => {
    if (false) {}
  });
}

function isModifiedEvent(event) {
  const eventTarget = event.currentTarget;
  const target = eventTarget.getAttribute("target");
  return target && target !== "_self" || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || // triggers resource download
  event.nativeEvent && event.nativeEvent.which === 2;
}

function linkClicked(e, router, href, as, replace, shallow, scroll, locale, isAppRouter, prefetchEnabled) {
  const {
    nodeName
  } = e.currentTarget; // anchors inside an svg have a lowercase nodeName

  const isAnchorNodeName = nodeName.toUpperCase() === "A";

  if (isAnchorNodeName && (isModifiedEvent(e) || // app-router supports external urls out of the box so it shouldn't short-circuit here as support for e.g. `replace` is added in the app-router.
  !isAppRouter && !(0, _islocalurl.isLocalURL)(href))) {
    // ignore click for browser’s default behavior
    return;
  }

  e.preventDefault();

  const navigate = () => {
    // If the router is an NextRouter instance it will have `beforePopState`
    if ("beforePopState" in router) {
      router[replace ? "replace" : "push"](href, as, {
        shallow,
        locale,
        scroll
      });
    } else {
      router[replace ? "replace" : "push"](as || href, {
        forceOptimisticNavigation: !prefetchEnabled
      });
    }
  };

  if (isAppRouter) {
    _react.default.startTransition(navigate);
  } else {
    navigate();
  }
}

function formatStringOrUrl(urlObjOrString) {
  if (typeof urlObjOrString === "string") {
    return urlObjOrString;
  }

  return (0, _formaturl.formatUrl)(urlObjOrString);
}
/**
 * React Component that enables client-side transitions between routes.
 */


const Link = /*#__PURE__*/_react.default.forwardRef(function LinkComponent(props, forwardedRef) {
  let children;

  const {
    href: hrefProp,
    as: asProp,
    children: childrenProp,
    prefetch: prefetchProp = null,
    passHref,
    replace,
    shallow,
    scroll,
    locale,
    onClick,
    onMouseEnter: onMouseEnterProp,
    onTouchStart: onTouchStartProp,
    // @ts-expect-error this is inlined as a literal boolean not a string
    legacyBehavior = true === false
  } = props,
        restProps = _objectWithoutProperties(props, _excluded);

  children = childrenProp;

  if (legacyBehavior && (typeof children === "string" || typeof children === "number")) {
    children = /*#__PURE__*/_react.default.createElement("a", null, children);
  }

  const prefetchEnabled = prefetchProp !== false;
  /**
   * The possible states for prefetch are:
   * - null: this is the default "auto" mode, where we will prefetch partially if the link is in the viewport
   * - true: we will prefetch if the link is visible and prefetch the full page, not just partially
   * - false: we will not prefetch if in the viewport at all
   */

  const appPrefetchKind = prefetchProp === null ? _routerreducertypes.PrefetchKind.AUTO : _routerreducertypes.PrefetchKind.FULL;

  const pagesRouter = _react.default.useContext(_routercontext.RouterContext);

  const appRouter = _react.default.useContext(_approutercontext.AppRouterContext);

  const router = pagesRouter != null ? pagesRouter : appRouter; // We're in the app directory if there is no pages router.

  const isAppRouter = !pagesRouter;

  if (false) { var createPropError; }

  if (false) {}

  const {
    href,
    as
  } = _react.default.useMemo(() => {
    if (!pagesRouter) {
      const resolvedHref = formatStringOrUrl(hrefProp);
      return {
        href: resolvedHref,
        as: asProp ? formatStringOrUrl(asProp) : resolvedHref
      };
    }

    const [resolvedHref, resolvedAs] = (0, _resolvehref.resolveHref)(pagesRouter, hrefProp, true);
    return {
      href: resolvedHref,
      as: asProp ? (0, _resolvehref.resolveHref)(pagesRouter, asProp) : resolvedAs || resolvedHref
    };
  }, [pagesRouter, hrefProp, asProp]);

  const previousHref = _react.default.useRef(href);

  const previousAs = _react.default.useRef(as); // This will return the first child, if multiple are provided it will throw an error


  let child;

  if (legacyBehavior) {
    if (false) {} else {
      child = _react.default.Children.only(children);
    }
  } else {
    if (false) {}
  }

  const childRef = legacyBehavior ? child && typeof child === "object" && child.ref : forwardedRef;
  const [setIntersectionRef, isVisible, resetVisible] = (0, _useintersection.useIntersection)({
    rootMargin: "200px"
  });

  const setRef = _react.default.useCallback(el => {
    // Before the link getting observed, check if visible state need to be reset
    if (previousAs.current !== as || previousHref.current !== href) {
      resetVisible();
      previousAs.current = as;
      previousHref.current = href;
    }

    setIntersectionRef(el);

    if (childRef) {
      if (typeof childRef === "function") childRef(el);else if (typeof childRef === "object") {
        childRef.current = el;
      }
    }
  }, [as, childRef, href, resetVisible, setIntersectionRef]); // Prefetch the URL if we haven't already and it's visible.


  _react.default.useEffect(() => {
    // in dev, we only prefetch on hover to avoid wasting resources as the prefetch will trigger compiling the page.
    if (false) {}

    if (!router) {
      return;
    } // If we don't need to prefetch the URL, don't do prefetch.


    if (!isVisible || !prefetchEnabled) {
      return;
    } // Prefetch the URL.


    prefetch(router, href, as, {
      locale
    }, {
      kind: appPrefetchKind
    }, isAppRouter);
  }, [as, href, isVisible, locale, prefetchEnabled, pagesRouter == null ? void 0 : pagesRouter.locale, router, isAppRouter, appPrefetchKind]);

  const childProps = {
    ref: setRef,

    onClick(e) {
      if (false) {}

      if (!legacyBehavior && typeof onClick === "function") {
        onClick(e);
      }

      if (legacyBehavior && child.props && typeof child.props.onClick === "function") {
        child.props.onClick(e);
      }

      if (!router) {
        return;
      }

      if (e.defaultPrevented) {
        return;
      }

      linkClicked(e, router, href, as, replace, shallow, scroll, locale, isAppRouter, prefetchEnabled);
    },

    onMouseEnter(e) {
      if (!legacyBehavior && typeof onMouseEnterProp === "function") {
        onMouseEnterProp(e);
      }

      if (legacyBehavior && child.props && typeof child.props.onMouseEnter === "function") {
        child.props.onMouseEnter(e);
      }

      if (!router) {
        return;
      }

      if (!prefetchEnabled && isAppRouter) {
        return;
      }

      prefetch(router, href, as, {
        locale,
        priority: true,
        // @see {https://github.com/vercel/next.js/discussions/40268?sort=top#discussioncomment-3572642}
        bypassPrefetchedCheck: true
      }, {
        kind: appPrefetchKind
      }, isAppRouter);
    },

    onTouchStart(e) {
      if (!legacyBehavior && typeof onTouchStartProp === "function") {
        onTouchStartProp(e);
      }

      if (legacyBehavior && child.props && typeof child.props.onTouchStart === "function") {
        child.props.onTouchStart(e);
      }

      if (!router) {
        return;
      }

      if (!prefetchEnabled && isAppRouter) {
        return;
      }

      prefetch(router, href, as, {
        locale,
        priority: true,
        // @see {https://github.com/vercel/next.js/discussions/40268?sort=top#discussioncomment-3572642}
        bypassPrefetchedCheck: true
      }, {
        kind: appPrefetchKind
      }, isAppRouter);
    }

  }; // If child is an <a> tag and doesn't have a href attribute, or if the 'passHref' property is
  // defined, we specify the current 'href', so that repetition is not needed by the user.
  // If the url is absolute, we can bypass the logic to prepend the domain and locale.

  if ((0, _utils.isAbsoluteUrl)(as)) {
    childProps.href = as;
  } else if (!legacyBehavior || passHref || child.type === "a" && !("href" in child.props)) {
    const curLocale = typeof locale !== "undefined" ? locale : pagesRouter == null ? void 0 : pagesRouter.locale; // we only render domain locales if we are currently on a domain locale
    // so that locale links are still visitable in development/preview envs

    const localeDomain = (pagesRouter == null ? void 0 : pagesRouter.isLocaleDomain) && (0, _getdomainlocale.getDomainLocale)(as, curLocale, pagesRouter == null ? void 0 : pagesRouter.locales, pagesRouter == null ? void 0 : pagesRouter.domainLocales);
    childProps.href = localeDomain || (0, _addbasepath.addBasePath)((0, _addlocale.addLocale)(as, curLocale, pagesRouter == null ? void 0 : pagesRouter.defaultLocale));
  }

  return legacyBehavior ? /*#__PURE__*/_react.default.cloneElement(child, childProps) : /*#__PURE__*/_react.default.createElement("a", _objectSpread(_objectSpread({}, restProps), childProps), children);
});

const _default = Link;

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 1485:
/***/ ((module, exports, __webpack_require__) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "normalizePathTrailingSlash", ({
  enumerable: true,
  get: function () {
    return normalizePathTrailingSlash;
  }
}));

const _removetrailingslash = __webpack_require__(3297);

const _parsepath = __webpack_require__(8854);

const normalizePathTrailingSlash = path => {
  if (!path.startsWith("/") || undefined) {
    return path;
  }

  const {
    pathname,
    query,
    hash
  } = (0, _parsepath.parsePath)(path);

  if (false) {}

  return "" + (0, _removetrailingslash.removeTrailingSlash)(pathname) + query + hash;
};

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 4818:
/***/ ((module, exports) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
0 && (0);

function _export(target, all) {
  for (var name in all) Object.defineProperty(target, name, {
    enumerable: true,
    get: all[name]
  });
}

_export(exports, {
  requestIdleCallback: function () {
    return requestIdleCallback;
  },
  cancelIdleCallback: function () {
    return cancelIdleCallback;
  }
});

const requestIdleCallback = typeof self !== "undefined" && self.requestIdleCallback && self.requestIdleCallback.bind(window) || function (cb) {
  let start = Date.now();
  return self.setTimeout(function () {
    cb({
      didTimeout: false,
      timeRemaining: function () {
        return Math.max(0, 50 - (Date.now() - start));
      }
    });
  }, 1);
};

const cancelIdleCallback = typeof self !== "undefined" && self.cancelIdleCallback && self.cancelIdleCallback.bind(window) || function (id) {
  return clearTimeout(id);
};

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 4203:
/***/ ((module, exports, __webpack_require__) => {

"use strict";


Object.defineProperty(exports, "__esModule", ({
  value: true
}));
Object.defineProperty(exports, "useIntersection", ({
  enumerable: true,
  get: function () {
    return useIntersection;
  }
}));

const _react = __webpack_require__(6689);

const _requestidlecallback = __webpack_require__(4818);

const hasIntersectionObserver = typeof IntersectionObserver === "function";
const observers = new Map();
const idList = [];

function createObserver(options) {
  const id = {
    root: options.root || null,
    margin: options.rootMargin || ""
  };
  const existing = idList.find(obj => obj.root === id.root && obj.margin === id.margin);
  let instance;

  if (existing) {
    instance = observers.get(existing);

    if (instance) {
      return instance;
    }
  }

  const elements = new Map();
  const observer = new IntersectionObserver(entries => {
    entries.forEach(entry => {
      const callback = elements.get(entry.target);
      const isVisible = entry.isIntersecting || entry.intersectionRatio > 0;

      if (callback && isVisible) {
        callback(isVisible);
      }
    });
  }, options);
  instance = {
    id,
    observer,
    elements
  };
  idList.push(id);
  observers.set(id, instance);
  return instance;
}

function observe(element, callback, options) {
  const {
    id,
    observer,
    elements
  } = createObserver(options);
  elements.set(element, callback);
  observer.observe(element);
  return function unobserve() {
    elements.delete(element);
    observer.unobserve(element); // Destroy observer when there's nothing left to watch:

    if (elements.size === 0) {
      observer.disconnect();
      observers.delete(id);
      const index = idList.findIndex(obj => obj.root === id.root && obj.margin === id.margin);

      if (index > -1) {
        idList.splice(index, 1);
      }
    }
  };
}

function useIntersection(param) {
  let {
    rootRef,
    rootMargin,
    disabled
  } = param;
  const isDisabled = disabled || !hasIntersectionObserver;
  const [visible, setVisible] = (0, _react.useState)(false);
  const elementRef = (0, _react.useRef)(null);
  const setElement = (0, _react.useCallback)(element => {
    elementRef.current = element;
  }, []);
  (0, _react.useEffect)(() => {
    if (hasIntersectionObserver) {
      if (isDisabled || visible) return;
      const element = elementRef.current;

      if (element && element.tagName) {
        const unobserve = observe(element, isVisible => isVisible && setVisible(isVisible), {
          root: rootRef == null ? void 0 : rootRef.current,
          rootMargin
        });
        return unobserve;
      }
    } else {
      if (!visible) {
        const idleCallback = (0, _requestidlecallback.requestIdleCallback)(() => setVisible(true));
        return () => (0, _requestidlecallback.cancelIdleCallback)(idleCallback);
      }
    } // eslint-disable-next-line react-hooks/exhaustive-deps

  }, [isDisabled, rootMargin, rootRef, visible, elementRef.current]);
  const resetVisible = (0, _react.useCallback)(() => {
    setVisible(false);
  }, []);
  return [setElement, visible, resetVisible];
}

if ((typeof exports.default === 'function' || typeof exports.default === 'object' && exports.default !== null) && typeof exports.default.__esModule === 'undefined') {
  Object.defineProperty(exports.default, '__esModule', {
    value: true
  });
  Object.assign(exports.default, exports);
  module.exports = exports.default;
}

/***/ }),

/***/ 1664:
/***/ ((module, __unused_webpack_exports, __webpack_require__) => {

module.exports = __webpack_require__(9771)


/***/ }),

/***/ 167:
/***/ ((__unused_webpack_module, exports) => {

"use strict";


exports._ = exports._interop_require_default = _interop_require_default;
function _interop_require_default(obj) {
    return obj && obj.__esModule ? obj : { default: obj };
}


/***/ })

};
;