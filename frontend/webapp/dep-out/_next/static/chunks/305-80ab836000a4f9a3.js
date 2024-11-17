"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[305],{109:function(e,t,r){/**
 * @license React
 * use-sync-external-store-with-selector.production.min.js
 *
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */var n=r(2265),o="function"==typeof Object.is?Object.is:function(e,t){return e===t&&(0!==e||1/e==1/t)||e!=e&&t!=t},u=n.useSyncExternalStore,c=n.useRef,i=n.useEffect,l=n.useMemo,s=n.useDebugValue;t.useSyncExternalStoreWithSelector=function(e,t,r,n,f){var b=c(null);if(null===b.current){var S={hasValue:!1,value:null};b.current=S}else S=b.current;var y=u(e,(b=l(function(){function a(t){if(!c){if(c=!0,e=t,t=n(t),void 0!==f&&S.hasValue){var r=S.value;if(f(r,t))return u=r}return u=t}if(r=u,o(e,t))return r;var i=n(t);return void 0!==f&&f(r,i)?r:(e=t,u=i)}var e,u,c=!1,i=void 0===r?null:r;return[function(){return a(t())},null===i?void 0:function(){return a(i())}]},[t,r,n,f]))[0],b[1]);return i(function(){S.hasValue=!0,S.value=y},[y]),s(y),y}},9688:function(e,t,r){e.exports=r(109)},4667:function(e,t,r){r.d(t,{Z:function(){return _objectDestructuringEmpty}});function _objectDestructuringEmpty(e){if(null==e)throw TypeError("Cannot destructure undefined")}},1600:function(e,t,r){r.d(t,{Z:function(){return _objectWithoutProperties}});function _objectWithoutProperties(e,t){if(null==e)return{};var r,n,o=function(e,t){if(null==e)return{};var r,n,o={},u=Object.keys(e);for(n=0;n<u.length;n++)r=u[n],t.indexOf(r)>=0||(o[r]=e[r]);return o}(e,t);if(Object.getOwnPropertySymbols){var u=Object.getOwnPropertySymbols(e);for(n=0;n<u.length;n++)r=u[n],!(t.indexOf(r)>=0)&&Object.prototype.propertyIsEnumerable.call(e,r)&&(o[r]=e[r])}return o}},941:function(e,t,r){r.d(t,{Z:function(){return _toConsumableArray}});var n=r(6015),o=r(909);function _toConsumableArray(e){return function(e){if(Array.isArray(e))return(0,n.Z)(e)}(e)||function(e){if("undefined"!=typeof Symbol&&null!=e[Symbol.iterator]||null!=e["@@iterator"])return Array.from(e)}(e)||(0,o.Z)(e)||function(){throw TypeError("Invalid attempt to spread non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method.")}()}},3046:function(e,t,r){r.d(t,{I0:function(){return p},v9:function(){return s},zt:function(){return Provider_default}});var n=r(2265),o=r(9688),u=Symbol.for("react-redux-context"),c="undefined"!=typeof globalThis?globalThis:{},i=function(){if(!n.createContext)return{};let e=c[u]??(c[u]=new Map),t=e.get(n.createContext);return t||(t=n.createContext(null),e.set(n.createContext,t)),t}();function createReduxContextHook(e=i){return function(){let t=n.useContext(e);return t}}var l=createReduxContextHook(),useSyncExternalStoreWithSelector=()=>{throw Error("uSES not initialized!")},refEquality=(e,t)=>e===t,s=function(e=i){let t=e===i?l:createReduxContextHook(e),useSelector2=(e,r={})=>{let{equalityFn:o=refEquality,devModeChecks:u={}}="function"==typeof r?{equalityFn:r}:r,{store:c,subscription:i,getServerState:l,stabilityCheck:s,identityFunctionCheck:f}=t();n.useRef(!0);let b=n.useCallback({[e.name](t){let r=e(t);return r}}[e.name],[e,s,u.stabilityCheck]),S=useSyncExternalStoreWithSelector(i.addNestedSub,c.getState,l||c.getState,b,o);return n.useDebugValue(S),S};return Object.assign(useSelector2,{withTypes:()=>useSelector2}),useSelector2}();Symbol.for("react.element"),Symbol.for("react.portal"),Symbol.for("react.fragment"),Symbol.for("react.strict_mode"),Symbol.for("react.profiler"),Symbol.for("react.provider"),Symbol.for("react.context"),Symbol.for("react.server_context"),Symbol.for("react.forward_ref"),Symbol.for("react.suspense"),Symbol.for("react.suspense_list"),Symbol.for("react.memo"),Symbol.for("react.lazy"),Symbol.for("react.offscreen"),Symbol.for("react.client.reference");var f={notify(){},get:()=>[]},b=!!("undefined"!=typeof window&&void 0!==window.document&&void 0!==window.document.createElement),S="undefined"!=typeof navigator&&"ReactNative"===navigator.product,y=b||S?n.useLayoutEffect:n.useEffect,Provider_default=function({store:e,context:t,children:r,serverState:o,stabilityCheck:u="once",identityFunctionCheck:c="once"}){let l=n.useMemo(()=>{let t=function(e,t){let r;let n=f,o=0,u=!1;function handleChangeWrapper(){c.onStateChange&&c.onStateChange()}function trySubscribe(){if(o++,!r){let o,u;r=t?t.addNestedSub(handleChangeWrapper):e.subscribe(handleChangeWrapper),o=null,u=null,n={clear(){o=null,u=null},notify(){(()=>{let e=o;for(;e;)e.callback(),e=e.next})()},get(){let e=[],t=o;for(;t;)e.push(t),t=t.next;return e},subscribe(e){let t=!0,r=u={callback:e,next:null,prev:u};return r.prev?r.prev.next=r:o=r,function(){t&&null!==o&&(t=!1,r.next?r.next.prev=r.prev:u=r.prev,r.prev?r.prev.next=r.next:o=r.next)}}}}}function tryUnsubscribe(){o--,r&&0===o&&(r(),r=void 0,n.clear(),n=f)}let c={addNestedSub:function(e){trySubscribe();let t=n.subscribe(e),r=!1;return()=>{r||(r=!0,t(),tryUnsubscribe())}},notifyNestedSubs:function(){n.notify()},handleChangeWrapper,isSubscribed:function(){return u},trySubscribe:function(){u||(u=!0,trySubscribe())},tryUnsubscribe:function(){u&&(u=!1,tryUnsubscribe())},getListeners:()=>n};return c}(e);return{store:e,subscription:t,getServerState:o?()=>o:void 0,stabilityCheck:u,identityFunctionCheck:c}},[e,o,u,c]),s=n.useMemo(()=>e.getState(),[e]);return y(()=>{let{subscription:t}=l;return t.onStateChange=t.notifyNestedSubs,t.trySubscribe(),s!==e.getState()&&t.notifyNestedSubs(),()=>{t.tryUnsubscribe(),t.onStateChange=void 0}},[l,s]),n.createElement((t||i).Provider,{value:l},r)};function createStoreHook(e=i){let t=e===i?l:createReduxContextHook(e),useStore2=()=>{let{store:e}=t();return e};return Object.assign(useStore2,{withTypes:()=>useStore2}),useStore2}var d=createStoreHook(),p=function(e=i){let t=e===i?d:createStoreHook(e),useDispatch2=()=>{let e=t();return e.dispatch};return Object.assign(useDispatch2,{withTypes:()=>useDispatch2}),useDispatch2}();useSyncExternalStoreWithSelector=o.useSyncExternalStoreWithSelector,n.useSyncExternalStore}}]);