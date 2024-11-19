"use strict";(self.webpackChunk_N_E=self.webpackChunk_N_E||[]).push([[727],{8727:function(e,t,r){r.d(t,{AD:function(){return RightArrowIcon},BT:function(){return ActionsGroup},C_:function(){return TraceIcon},Dk:function(){return BellIcon},EU:function(){return RadioButton},Ex:function(){return ErrorSamplerIcon},Hj:function(){return RedErrorIcon},I$:function(){return LogsIcon},II:function(){return Input},If:function(){return UnFocusOverviewIcon},Ik:function(){return ChargeIcon},Kl:function(){return UnFocusSourcesIcon},Kx:function(){return TextArea},Mg:function(){return MetricsFocusIcon},Mj:function(){return SearchInput},Pc:function(){return FocusOverviewIcon},Pf:function(){return PiiMaskingIcon},Ph:function(){return ConnectIcon},Qm:function(){return UnFocusDestinationsIcon},Rg:function(){return Steps},Rk:function(){return LatencySamplerIcon},Uw:function(){return Tap},Vp:function(){return Tag},XZ:function(){return Checkbox},Y8:function(){return WhiteArrowIcon},Zb:function(){return Card},aN:function(){return Loader},b0:function(){return TraceFocusIcon},cM:function(){return KeyvalDataFlow},cN:function(){return DangerZone},cl:function(){return LogsFocusIcon},ct:function(){return DeleteAttributeIcon},dr:function(){return MetricsIcon},e8:function(){return BlueInfoIcon},iA:function(){return Table3},iM:function(){return GreenCheckIcon},nH:function(){return ThemeProviderWrapper},oq:function(){return MultiInputTable},pO:function(){return PlusIcon},rU:function(){return Link},rs:function(){return Switch},t4:function(){return RenameAttributeIcon},u:function(){return Tooltip},uA:function(){return FocusActionIcon},uY:function(){return FocusSourcesIcon},u_:function(){return Modal},vb:function(){return DropDown},vu:function(){return KeyValueTable},wM:function(){return FocusDestinationsIcon},wZ:function(){return UnFocusActionIcon},xC:function(){return BackIcon},xP:function(){return LinkIcon},xq:function(){return buildFlowNodesAndEdges},xv:function(){return Text},yI:function(){return ProbabilisticSamplerIcon},yr:function(){return AddClusterInfoIcon},zx:function(){return Button}});var n=r(2265),l=r(1369);r(6691);var i=r(4867),a=r(9673);r(715);var o=r(8967),c=r(4887),s=r(4033);r(5424);var d=l.ZP.label`
  height: 24px;
  color: #303030;
  font-size: 14px;
  font-weight: 400;
  margin-right: 7px;
  -webkit-tap-highlight-color: transparent;
  display: flex;
  align-items: center;

  gap: 10px;
  cursor: pointer;
`,p=l.ZP.span`
  cursor: pointer;
  width: 23px;
  height: 23px;
  border: ${({theme:e})=>`solid 2px ${e.colors.light_grey}`};
  border-radius: 50%;
  display: inline-block;
  position: relative;
`,u=l.ZP.p`
  color: ${({theme:e})=>e.text.white};
  margin: 0;
  font-family: ${({theme:e})=>e.font_family.primary}, sans-serif;
  font-size: 16px;
  font-weight: 400;
`;function Text({children:e,color:t,style:r,weight:l,size:i,...a}){return n.createElement(u,{style:{fontWeight:l,color:t,fontSize:i,...r},...a},e)}var checked_radio_default=e=>n.createElement("svg",{xmlns:"http://www.w3.org/2000/svg",width:15,height:15,viewBox:"0 0 18 18",fill:"none",...e},n.createElement("rect",{x:.5,y:.5,width:17,height:17,rx:8.5,fill:"#96F2FF",stroke:"#96F2FF"}),n.createElement("path",{d:"M13.7727 6L7.39773 12.375L4.5 9.47727",stroke:"#132330",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"})),RadioButton=({label:e="",onChange:t,value:r,size:l=25,textStyles:i={}})=>n.createElement(d,null,n.createElement("div",{onClick:function(){t&&t({})},style:{display:"flex",alignItems:"center"}},r?n.createElement(checked_radio_default,{width:l,height:l}):n.createElement(p,{style:{width:l,height:l}})),n.createElement(Text,{...i},e)),C=l.ZP.div`
  :hover {
    background: ${({theme:e,disabled:t,variant:r})=>t?e.colors.blue_grey:"primary"===r?e.colors.torquiz_light:"transparent"};
  }
  p {
    cursor: ${({disabled:e})=>e?"not-allowed !important":"pointer !important"};
  }
`,m=l.ZP.button`
  display: flex;
  padding: 8px 16px;
  align-items: center;
  border-radius: 8px;
  border: none;
  width: 100%;
  height: 100%;
  border: 1px solid
    ${({theme:e,variant:t})=>"primary"===t?"transparent":e.colors.secondary};
  cursor: ${({disabled:e})=>e?"not-allowed !important":"pointer !important"};
  background: ${({theme:e,disabled:t,variant:r})=>t?"primary"===r?e.colors.blue_grey:"transparent":"primary"===r?e.colors.secondary:"transparent"};
  justify-content: center;
  align-items: center;
  opacity: ${({disabled:e,variant:t})=>"primary"!==t&&e?.5:1};
`,Button=({variant:e="primary",children:t,style:r,disabled:l,type:i="button",...a})=>n.createElement(C,{variant:e,disabled:l},n.createElement(m,{type:i,variant:e,disabled:l,style:{...r},...a},t)),g=l.ZP.div`
  background: radial-gradient(
        circle at 100% 100%,
        #ffffff 0,
        #ffffff 3px,
        transparent 3px
      )
      0% 0%/8px 8px no-repeat,
    radial-gradient(circle at 0 100%, #ffffff 0, #ffffff 3px, transparent 3px)
      100% 0%/8px 8px no-repeat,
    radial-gradient(circle at 100% 0, #ffffff 0, #ffffff 3px, transparent 3px)
      0% 100%/8px 8px no-repeat,
    radial-gradient(circle at 0 0, #ffffff 0, #ffffff 3px, transparent 3px) 100%
      100%/8px 8px no-repeat,
    linear-gradient(#ffffff, #ffffff) 50% 50% / calc(100% - 10px)
      calc(100% - 16px) no-repeat,
    linear-gradient(#ffffff, #ffffff) 50% 50% / calc(100% - 16px)
      calc(100% - 10px) no-repeat,
    linear-gradient(0deg, transparent 0%, #0ee6f3 100%),
    radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
  border-radius: 8px;
  padding: 1px;
  width: 32px;
  height: 32px;
`,h=l.ZP.div`
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  justify-content: center;
  align-items: center;
  background: radial-gradient(
      78.09% 72.18% at 100% -0%,
      rgba(150, 242, 255, 0.4) 0%,
      rgba(150, 242, 255, 0) 61.91%
    ),
    linear-gradient(180deg, #2e4c55 0%, #303355 100%);
`;function FloatBox({children:e,style:t={}}){return n.createElement(g,null,n.createElement(h,{style:{...t}},e))}var trash_default=e=>n.createElement("svg",{width:"14px",height:"14px",viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M20.5001 6H3.5",stroke:"#0EE6F3",strokeWidth:1.5,strokeLinecap:"round"}),n.createElement("path",{d:"M18.8332 8.5L18.3732 15.3991C18.1962 18.054 18.1077 19.3815 17.2427 20.1907C16.3777 21 15.0473 21 12.3865 21H11.6132C8.95235 21 7.62195 21 6.75694 20.1907C5.89194 19.3815 5.80344 18.054 5.62644 15.3991L5.1665 8.5",stroke:"#0EE6F3",strokeWidth:1.5,strokeLinecap:"round"}),n.createElement("path",{d:"M9.5 11L10 16",stroke:"#0EE6F3",strokeWidth:1.5,strokeLinecap:"round"}),n.createElement("path",{d:"M14.5 11L14 16",stroke:"#0EE6F3",strokeWidth:1.5,strokeLinecap:"round"}),n.createElement("path",{d:"M6.5 6C6.55588 6 6.58382 6 6.60915 5.99936C7.43259 5.97849 8.15902 5.45491 8.43922 4.68032C8.44784 4.65649 8.45667 4.62999 8.47434 4.57697L8.57143 4.28571C8.65431 4.03708 8.69575 3.91276 8.75071 3.8072C8.97001 3.38607 9.37574 3.09364 9.84461 3.01877C9.96213 3 10.0932 3 10.3553 3H13.6447C13.9068 3 14.0379 3 14.1554 3.01877C14.6243 3.09364 15.03 3.38607 15.2493 3.8072C15.3043 3.91276 15.3457 4.03708 15.4286 4.28571L15.5257 4.57697C15.5433 4.62992 15.5522 4.65651 15.5608 4.68032C15.841 5.45491 16.5674 5.97849 17.3909 5.99936C17.4162 6 17.4441 6 17.5 6",stroke:"#0EE6F3",strokeWidth:1.5}))),check_default=e=>n.createElement("svg",{width:10,height:10,viewBox:"0 0 10 10",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{clipPath:"url(#clip0_48_7283)"},n.createElement("path",{d:"M1.5625 5.625L3.75 7.8125L8.75 2.8125",stroke:"#96F2FF",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"})),n.createElement("defs",null,n.createElement("clipPath",{id:"clip0_48_7283"},n.createElement("rect",{width:10,height:10,fill:"white"})))),expand_arrow_default=e=>n.createElement("svg",{width:12,height:13,viewBox:"0 0 12 13",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.0155 5.26528L6.26552 9.01528C6.23069 9.05014 6.18934 9.0778 6.14381 9.09667C6.09829 9.11555 6.04949 9.12526 6.00021 9.12526C5.95093 9.12526 5.90213 9.11555 5.85661 9.09667C5.81108 9.0778 5.76972 9.05014 5.7349 9.01528L1.9849 5.26528C1.91453 5.19491 1.875 5.09948 1.875 4.99996C1.875 4.90045 1.91453 4.80502 1.9849 4.73465C2.05526 4.66429 2.1507 4.62476 2.25021 4.62476C2.34972 4.62476 2.44516 4.66429 2.51552 4.73465L6.00021 8.21981L9.4849 4.73465C9.51974 4.69981 9.5611 4.67217 9.60662 4.65332C9.65214 4.63446 9.70094 4.62476 9.75021 4.62476C9.79948 4.62476 9.84827 4.63446 9.8938 4.65332C9.93932 4.67217 9.98068 4.69981 10.0155 4.73465C10.0504 4.76949 10.078 4.81086 10.0969 4.85638C10.1157 4.9019 10.1254 4.95069 10.1254 4.99996C10.1254 5.04924 10.1157 5.09803 10.0969 5.14355C10.078 5.18907 10.0504 5.23044 10.0155 5.26528Z",fill:"#CCD0D2"})),cluster_attr_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M4.97883 9.68508C2.99294 8.89073 2 8.49355 2 8C2 7.50645 2.99294 7.10927 4.97883 6.31492L7.7873 5.19153C9.77318 4.39718 10.7661 4 12 4C13.2339 4 14.2268 4.39718 16.2127 5.19153L19.0212 6.31492C21.0071 7.10927 22 7.50645 22 8C22 8.49355 21.0071 8.89073 19.0212 9.68508L16.2127 10.8085C14.2268 11.6028 13.2339 12 12 12C10.7661 12 9.77318 11.6028 7.7873 10.8085L4.97883 9.68508Z",stroke:"#8b92a6",strokeWidth:1.5}),n.createElement("path",{d:"M22 12C22 12 21.0071 12.8907 19.0212 13.6851L16.2127 14.8085C14.2268 15.6028 13.2339 16 12 16C10.7661 16 9.77318 15.6028 7.7873 14.8085L4.97883 13.6851C2.99294 12.8907 2 12 2 12",stroke:"#8b92a6",strokeWidth:1.5,strokeLinecap:"round"}),n.createElement("path",{d:"M22 16C22 16 21.0071 16.8907 19.0212 17.6851L16.2127 18.8085C14.2268 19.6028 13.2339 20 12 20C10.7661 20 9.77318 19.6028 7.7873 18.8085L4.97883 17.6851C2.99294 16.8907 2 16 2 16",stroke:"#8b92a6",strokeWidth:1.5,strokeLinecap:"round"}))),delete_attr_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M7 9.5L12 14.5M12 9.5L7 14.5M19.4922 13.9546L16.5608 17.7546C16.2082 18.2115 16.032 18.44 15.8107 18.6047C15.6146 18.7505 15.3935 18.8592 15.1583 18.9253C14.8928 19 14.6042 19 14.0271 19H6.2C5.07989 19 4.51984 19 4.09202 18.782C3.71569 18.5903 3.40973 18.2843 3.21799 17.908C3 17.4802 3 16.9201 3 15.8V8.2C3 7.0799 3 6.51984 3.21799 6.09202C3.40973 5.71569 3.71569 5.40973 4.09202 5.21799C4.51984 5 5.07989 5 6.2 5H14.0271C14.6042 5 14.8928 5 15.1583 5.07467C15.3935 5.14081 15.6146 5.2495 15.8107 5.39534C16.032 5.55998 16.2082 5.78846 16.5608 6.24543L19.4922 10.0454C20.0318 10.7449 20.3016 11.0947 20.4054 11.4804C20.4969 11.8207 20.4969 12.1793 20.4054 12.5196C20.3016 12.9053 20.0318 13.2551 19.4922 13.9546Z",stroke:"#8b92a7",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"}))),rename_attr_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M20 7H9.00001C6.23858 7 4 9.23857 4 12C4 14.7614 6.23858 17 9 17H16M20 7L17 4M20 7L17 10",stroke:"#8b92a7",strokeWidth:1.5,strokeLinecap:"round",strokeLinejoin:"round"}))),error_sampler_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",role:"img",xmlns:"http://www.w3.org/2000/svg","aria-labelledby":"errorIconTitle",stroke:"#8b92a7",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round",fill:"none",color:"#000000",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("title",{id:"errorIconTitle"},"Error"),n.createElement("path",{d:"M12 8L12 13"}),n.createElement("line",{x1:12,y1:16,x2:12,y2:16}),n.createElement("circle",{cx:12,cy:12,r:10}))),pii_masking_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M3 7C3 5.11438 3 4.17157 3.58579 3.58579C4.17157 3 5.11438 3 7 3H12H17C18.8856 3 19.8284 3 20.4142 3.58579C21 4.17157 21 5.11438 21 7V15V17C21 18.8856 21 19.8284 20.4142 20.4142C19.8284 21 18.8856 21 17 21H12H7C5.11438 21 4.17157 21 3.58579 20.4142C3 19.8284 3 18.8856 3 17V15V7Z",stroke:"#8b92a7",strokeWidth:2,strokeLinejoin:"round"}),n.createElement("path",{d:"M16 12C16 14.2091 14.2091 16 12 16C9.79086 16 8 14.2091 8 12C8 9.79086 9.79086 8 12 8C14.2091 8 16 9.79086 16 12Z",stroke:"#8b92a7",strokeWidth:2}))),latency_sampler_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M23 12C23 18.0751 18.0751 23 12 23C5.92487 23 1 18.0751 1 12C1 5.92487 5.92487 1 12 1C18.0751 1 23 5.92487 23 12ZM3.00683 12C3.00683 16.9668 7.03321 20.9932 12 20.9932C16.9668 20.9932 20.9932 16.9668 20.9932 12C20.9932 7.03321 16.9668 3.00683 12 3.00683C7.03321 3.00683 3.00683 7.03321 3.00683 12Z",fill:"#8b92a7"}),n.createElement("path",{d:"M12 5C11.4477 5 11 5.44771 11 6V12.4667C11 12.4667 11 12.7274 11.1267 12.9235C11.2115 13.0898 11.3437 13.2343 11.5174 13.3346L16.1372 16.0019C16.6155 16.278 17.2271 16.1141 17.5032 15.6358C17.7793 15.1575 17.6155 14.5459 17.1372 14.2698L13 11.8812V6C13 5.44772 12.5523 5 12 5Z",fill:"#8b92a7"}))),probabilistic_sampler_default=e=>n.createElement("svg",{fill:"#8b92a7",id:"Capa_1",xmlns:"http://www.w3.org/2000/svg",xmlnsXlink:"http://www.w3.org/1999/xlink",viewBox:"0 0 320.281 320.281",xmlSpace:"preserve",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("g",null,n.createElement("path",{d:"M260.727,115.941l-97.891,53.473V57.89c0-4.971-4.029-9-9-9c-74.823,0-135.695,60.873-135.695,135.695 s60.873,135.696,135.695,135.696s135.696-60.873,135.696-135.696c0-22.735-5.739-45.234-16.596-65.067 C270.551,115.161,265.087,113.561,260.727,115.941z M153.836,302.281c-64.897,0-117.695-52.798-117.695-117.696 c0-61.871,47.984-112.745,108.695-117.354v117.354c0,3.177,1.675,6.119,4.408,7.741c2.733,1.622,6.119,1.682,8.906,0.158 l103.007-56.267c6.807,15.117,10.375,31.667,10.375,48.369C271.531,249.482,218.733,302.281,153.836,302.281z"}),n.createElement("path",{d:"M301.035,70.59c-23.221-42.42-67.63-69.468-115.896-70.588c-4.974-0.1-9.089,3.817-9.207,8.785l-2.995,126.658 c-0.076,3.215,1.569,6.226,4.314,7.898c1.436,0.875,3.058,1.314,4.684,1.314c1.482,0,2.968-0.366,4.314-1.102L297.455,82.81 c2.096-1.145,3.651-3.076,4.322-5.368C302.449,75.15,302.182,72.685,301.035,70.59z M191.3,120.286l2.406-101.733 c35.355,3.565,67.468,23.126,86.91,52.944L191.3,120.286z"})))),f={AddClusterInfo:cluster_attr_default,RenameAttribute:rename_attr_default,DeleteAttribute:delete_attr_default,ErrorSampler:error_sampler_default,PiiMasking:pii_masking_default,LatencySampler:latency_sampler_default,ProbabilisticSampler:probabilistic_sampler_default},x="https://d1n7d4xz7fr8b4.cloudfront.net/",E={java:`${x}java.png`,go:`${x}go.png`,javascript:`${x}nodejs.png`,python:`${x}python.png`,dotnet:`${x}dotnet.png`,default:`${x}default.png`,mysql:`${x}mysql.png`,unknown:`${x}default.svg`,processing:`${x}default.svg`,"no containers":`${x}default.svg`,"no running pods":`${x}default.svg`,nginx:`${x}nginx.svg`,postgres:`${x}postgres.svg`},logs_grey_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M2 4C2 3.86739 2.05268 3.74021 2.14645 3.64645C2.24021 3.55268 2.36739 3.5 2.5 3.5H13.5C13.6326 3.5 13.7598 3.55268 13.8536 3.64645C13.9473 3.74021 14 3.86739 14 4C14 4.13261 13.9473 4.25979 13.8536 4.35355C13.7598 4.44732 13.6326 4.5 13.5 4.5H2.5C2.36739 4.5 2.24021 4.44732 2.14645 4.35355C2.05268 4.25979 2 4.13261 2 4ZM2.5 7H10.5C10.6326 7 10.7598 6.94732 10.8536 6.85355C10.9473 6.75979 11 6.63261 11 6.5C11 6.36739 10.9473 6.24021 10.8536 6.14645C10.7598 6.05268 10.6326 6 10.5 6H2.5C2.36739 6 2.24021 6.05268 2.14645 6.14645C2.05268 6.24021 2 6.36739 2 6.5C2 6.63261 2.05268 6.75979 2.14645 6.85355C2.24021 6.94732 2.36739 7 2.5 7ZM13.5 8.5H2.5C2.36739 8.5 2.24021 8.55268 2.14645 8.64645C2.05268 8.74021 2 8.86739 2 9C2 9.13261 2.05268 9.25979 2.14645 9.35355C2.24021 9.44732 2.36739 9.5 2.5 9.5H13.5C13.6326 9.5 13.7598 9.44732 13.8536 9.35355C13.9473 9.25979 14 9.13261 14 9C14 8.86739 13.9473 8.74021 13.8536 8.64645C13.7598 8.55268 13.6326 8.5 13.5 8.5ZM10.5 11H2.5C2.36739 11 2.24021 11.0527 2.14645 11.1464C2.05268 11.2402 2 11.3674 2 11.5C2 11.6326 2.05268 11.7598 2.14645 11.8536C2.24021 11.9473 2.36739 12 2.5 12H10.5C10.6326 12 10.7598 11.9473 10.8536 11.8536C10.9473 11.7598 11 11.6326 11 11.5C11 11.3674 10.9473 11.2402 10.8536 11.1464C10.7598 11.0527 10.6326 11 10.5 11Z",fill:"#8B92A5"})),logs_blue_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M2 4C2 3.86739 2.05268 3.74021 2.14645 3.64645C2.24021 3.55268 2.36739 3.5 2.5 3.5H13.5C13.6326 3.5 13.7598 3.55268 13.8536 3.64645C13.9473 3.74021 14 3.86739 14 4C14 4.13261 13.9473 4.25979 13.8536 4.35355C13.7598 4.44732 13.6326 4.5 13.5 4.5H2.5C2.36739 4.5 2.24021 4.44732 2.14645 4.35355C2.05268 4.25979 2 4.13261 2 4ZM2.5 7H10.5C10.6326 7 10.7598 6.94732 10.8536 6.85355C10.9473 6.75979 11 6.63261 11 6.5C11 6.36739 10.9473 6.24021 10.8536 6.14645C10.7598 6.05268 10.6326 6 10.5 6H2.5C2.36739 6 2.24021 6.05268 2.14645 6.14645C2.05268 6.24021 2 6.36739 2 6.5C2 6.63261 2.05268 6.75979 2.14645 6.85355C2.24021 6.94732 2.36739 7 2.5 7ZM13.5 8.5H2.5C2.36739 8.5 2.24021 8.55268 2.14645 8.64645C2.05268 8.74021 2 8.86739 2 9C2 9.13261 2.05268 9.25979 2.14645 9.35355C2.24021 9.44732 2.36739 9.5 2.5 9.5H13.5C13.6326 9.5 13.7598 9.44732 13.8536 9.35355C13.9473 9.25979 14 9.13261 14 9C14 8.86739 13.9473 8.74021 13.8536 8.64645C13.7598 8.55268 13.6326 8.5 13.5 8.5ZM10.5 11H2.5C2.36739 11 2.24021 11.0527 2.14645 11.1464C2.05268 11.2402 2 11.3674 2 11.5C2 11.6326 2.05268 11.7598 2.14645 11.8536C2.24021 11.9473 2.36739 12 2.5 12H10.5C10.6326 12 10.7598 11.9473 10.8536 11.8536C10.9473 11.7598 11 11.6326 11 11.5C11 11.3674 10.9473 11.2402 10.8536 11.1464C10.7598 11.0527 10.6326 11 10.5 11Z",fill:"#96F2FF"})),chart_line_grey_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M14.5 13C14.5 13.1326 14.4473 13.2598 14.3536 13.3536C14.2598 13.4473 14.1326 13.5 14 13.5H2C1.86739 13.5 1.74021 13.4473 1.64645 13.3536C1.55268 13.2598 1.5 13.1326 1.5 13V3C1.5 2.86739 1.55268 2.74021 1.64645 2.64645C1.74021 2.55268 1.86739 2.5 2 2.5C2.13261 2.5 2.25979 2.55268 2.35355 2.64645C2.44732 2.74021 2.5 2.86739 2.5 3V8.89812L5.67063 6.125C5.7569 6.04947 5.86652 6.0059 5.9811 6.00157C6.09569 5.99725 6.20828 6.03244 6.3 6.10125L9.97563 8.85812L13.6706 5.625C13.7191 5.57704 13.7768 5.5395 13.8403 5.51467C13.9038 5.48985 13.9717 5.47827 14.0398 5.48065C14.1079 5.48303 14.1749 5.49931 14.2365 5.5285C14.2981 5.55769 14.3531 5.59917 14.398 5.65038C14.443 5.7016 14.4771 5.76148 14.4981 5.82633C14.5191 5.89119 14.5266 5.95965 14.5201 6.02752C14.5137 6.09538 14.4935 6.16122 14.4607 6.22097C14.4279 6.28073 14.3832 6.33314 14.3294 6.375L10.3294 9.875C10.2431 9.95053 10.1335 9.9941 10.0189 9.99843C9.90431 10.0028 9.79172 9.96756 9.7 9.89875L6.02437 7.14313L2.5 10.2269V12.5H14C14.1326 12.5 14.2598 12.5527 14.3536 12.6464C14.4473 12.7402 14.5 12.8674 14.5 13Z",fill:"#8B92A5"})),chart_line_blue_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M14.5 13C14.5 13.1326 14.4473 13.2598 14.3536 13.3536C14.2598 13.4473 14.1326 13.5 14 13.5H2C1.86739 13.5 1.74021 13.4473 1.64645 13.3536C1.55268 13.2598 1.5 13.1326 1.5 13V3C1.5 2.86739 1.55268 2.74021 1.64645 2.64645C1.74021 2.55268 1.86739 2.5 2 2.5C2.13261 2.5 2.25979 2.55268 2.35355 2.64645C2.44732 2.74021 2.5 2.86739 2.5 3V8.89812L5.67063 6.125C5.7569 6.04947 5.86652 6.0059 5.9811 6.00157C6.09569 5.99725 6.20828 6.03244 6.3 6.10125L9.97563 8.85812L13.6706 5.625C13.7191 5.57704 13.7768 5.5395 13.8403 5.51467C13.9038 5.48985 13.9717 5.47827 14.0398 5.48065C14.1079 5.48303 14.1749 5.49931 14.2365 5.5285C14.2981 5.55769 14.3531 5.59917 14.398 5.65038C14.443 5.7016 14.4771 5.76148 14.4981 5.82633C14.5191 5.89119 14.5266 5.95965 14.5201 6.02752C14.5137 6.09538 14.4935 6.16122 14.4607 6.22097C14.4279 6.28073 14.3832 6.33314 14.3294 6.375L10.3294 9.875C10.2431 9.95053 10.1335 9.9941 10.0189 9.99843C9.90431 10.0028 9.79172 9.96756 9.7 9.89875L6.02437 7.14313L2.5 10.2269V12.5H14C14.1326 12.5 14.2598 12.5527 14.3536 12.6464C14.4473 12.7402 14.5 12.8674 14.5 13Z",fill:"#96F2FF"})),tree_structure_grey_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.5 7H13.5C13.7652 7 14.0196 6.89464 14.2071 6.70711C14.3946 6.51957 14.5 6.26522 14.5 6V3C14.5 2.73478 14.3946 2.48043 14.2071 2.29289C14.0196 2.10536 13.7652 2 13.5 2H10.5C10.2348 2 9.98043 2.10536 9.79289 2.29289C9.60536 2.48043 9.5 2.73478 9.5 3V4H9C8.46957 4 7.96086 4.21071 7.58579 4.58579C7.21071 4.96086 7 5.46957 7 6V7.5H5V7C5 6.73478 4.89464 6.48043 4.70711 6.29289C4.51957 6.10536 4.26522 6 4 6H2C1.73478 6 1.48043 6.10536 1.29289 6.29289C1.10536 6.48043 1 6.73478 1 7V9C1 9.26522 1.10536 9.51957 1.29289 9.70711C1.48043 9.89464 1.73478 10 2 10H4C4.26522 10 4.51957 9.89464 4.70711 9.70711C4.89464 9.51957 5 9.26522 5 9V8.5H7V10C7 10.5304 7.21071 11.0391 7.58579 11.4142C7.96086 11.7893 8.46957 12 9 12H9.5V13C9.5 13.2652 9.60536 13.5196 9.79289 13.7071C9.98043 13.8946 10.2348 14 10.5 14H13.5C13.7652 14 14.0196 13.8946 14.2071 13.7071C14.3946 13.5196 14.5 13.2652 14.5 13V10C14.5 9.73478 14.3946 9.48043 14.2071 9.29289C14.0196 9.10536 13.7652 9 13.5 9H10.5C10.2348 9 9.98043 9.10536 9.79289 9.29289C9.60536 9.48043 9.5 9.73478 9.5 10V11H9C8.73478 11 8.48043 10.8946 8.29289 10.7071C8.10536 10.5196 8 10.2652 8 10V6C8 5.73478 8.10536 5.48043 8.29289 5.29289C8.48043 5.10536 8.73478 5 9 5H9.5V6C9.5 6.26522 9.60536 6.51957 9.79289 6.70711C9.98043 6.89464 10.2348 7 10.5 7ZM4 9H2V7H4V9ZM10.5 10H13.5V13H10.5V10ZM10.5 3H13.5V6H10.5V3Z",fill:"#8B92A5"})),tree_structure_blue_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.5 7H13.5C13.7652 7 14.0196 6.89464 14.2071 6.70711C14.3946 6.51957 14.5 6.26522 14.5 6V3C14.5 2.73478 14.3946 2.48043 14.2071 2.29289C14.0196 2.10536 13.7652 2 13.5 2H10.5C10.2348 2 9.98043 2.10536 9.79289 2.29289C9.60536 2.48043 9.5 2.73478 9.5 3V4H9C8.46957 4 7.96086 4.21071 7.58579 4.58579C7.21071 4.96086 7 5.46957 7 6V7.5H5V7C5 6.73478 4.89464 6.48043 4.70711 6.29289C4.51957 6.10536 4.26522 6 4 6H2C1.73478 6 1.48043 6.10536 1.29289 6.29289C1.10536 6.48043 1 6.73478 1 7V9C1 9.26522 1.10536 9.51957 1.29289 9.70711C1.48043 9.89464 1.73478 10 2 10H4C4.26522 10 4.51957 9.89464 4.70711 9.70711C4.89464 9.51957 5 9.26522 5 9V8.5H7V10C7 10.5304 7.21071 11.0391 7.58579 11.4142C7.96086 11.7893 8.46957 12 9 12H9.5V13C9.5 13.2652 9.60536 13.5196 9.79289 13.7071C9.98043 13.8946 10.2348 14 10.5 14H13.5C13.7652 14 14.0196 13.8946 14.2071 13.7071C14.3946 13.5196 14.5 13.2652 14.5 13V10C14.5 9.73478 14.3946 9.48043 14.2071 9.29289C14.0196 9.10536 13.7652 9 13.5 9H10.5C10.2348 9 9.98043 9.10536 9.79289 9.29289C9.60536 9.48043 9.5 9.73478 9.5 10V11H9C8.73478 11 8.48043 10.8946 8.29289 10.7071C8.10536 10.5196 8 10.2652 8 10V6C8 5.73478 8.10536 5.48043 8.29289 5.29289C8.48043 5.10536 8.73478 5 9 5H9.5V6C9.5 6.26522 9.60536 6.51957 9.79289 6.70711C9.98043 6.89464 10.2348 7 10.5 7ZM4 9H2V7H4V9ZM10.5 10H13.5V13H10.5V10ZM10.5 3H13.5V6H10.5V3Z",fill:"#96F2FF"})),arrow_right_default=e=>n.createElement("svg",{width:32,height:32,viewBox:"0 0 32 32",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M27.7075 16.7076L18.7075 25.7076C18.5199 25.8952 18.2654 26.0006 18 26.0006C17.7346 26.0006 17.4801 25.8952 17.2925 25.7076C17.1049 25.5199 16.9994 25.2654 16.9994 25.0001C16.9994 24.7347 17.1049 24.4802 17.2925 24.2926L24.5863 17.0001H5C4.73478 17.0001 4.48043 16.8947 4.29289 16.7072C4.10536 16.5196 4 16.2653 4 16.0001C4 15.7349 4.10536 15.4805 4.29289 15.293C4.48043 15.1054 4.73478 15.0001 5 15.0001H24.5863L17.2925 7.70757C17.1049 7.51993 16.9994 7.26543 16.9994 7.00007C16.9994 6.7347 17.1049 6.48021 17.2925 6.29257C17.4801 6.10493 17.7346 5.99951 18 5.99951C18.2654 5.99951 18.5199 6.10493 18.7075 6.29257L27.7075 15.2926C27.8005 15.3854 27.8742 15.4957 27.9246 15.6171C27.9749 15.7385 28.0008 15.8687 28.0008 16.0001C28.0008 16.1315 27.9749 16.2616 27.9246 16.383C27.8742 16.5044 27.8005 16.6147 27.7075 16.7076Z",fill:"#0A1824"})),charge_rect_default=e=>n.createElement("svg",{width:48,height:48,viewBox:"0 0 48 48",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("rect",{width:48,height:48,rx:7.5,fill:"url(#paint0_linear_48_4503)"}),n.createElement("rect",{width:48,height:48,rx:7.5,fill:"url(#paint1_radial_48_4503)",fillOpacity:.4}),n.createElement("rect",{x:.375,y:.375,width:47.25,height:47.25,rx:7.125,stroke:"url(#paint2_linear_48_4503)",strokeOpacity:.5,strokeWidth:.75}),n.createElement("path",{d:"M34.6033 19.3484L31.1561 22.7946L33.8004 25.4379C33.875 25.5125 33.9342 25.6011 33.9746 25.6985C34.0149 25.796 34.0357 25.9005 34.0357 26.006C34.0357 26.1114 34.0149 26.2159 33.9746 26.3134C33.9342 26.4108 33.875 26.4994 33.8004 26.574C33.7259 26.6486 33.6373 26.7077 33.5398 26.7481C33.4424 26.7885 33.3379 26.8092 33.2324 26.8092C33.1269 26.8092 33.0225 26.7885 32.925 26.7481C32.8276 26.7077 32.739 26.6486 32.6644 26.574L32.0282 25.9367L26.7094 31.2555C26.3367 31.6283 25.8941 31.924 25.4071 32.1257C24.9201 32.3274 24.3981 32.4313 23.8709 32.4313C23.3437 32.4313 22.8217 32.3274 22.3347 32.1257C21.8476 31.924 21.4051 31.6283 21.0324 31.2555L19.4588 29.6759L14.5324 34.6023C14.4578 34.6769 14.3693 34.7361 14.2718 34.7764C14.1744 34.8168 14.0699 34.8376 13.9644 34.8376C13.8589 34.8376 13.7545 34.8168 13.657 34.7764C13.5596 34.7361 13.471 34.6769 13.3964 34.6023C13.3218 34.5277 13.2626 34.4392 13.2223 34.3417C13.1819 34.2442 13.1611 34.1398 13.1611 34.0343C13.1611 33.9288 13.1819 33.8244 13.2223 33.7269C13.2626 33.6294 13.3218 33.5409 13.3964 33.4663L18.3228 28.5399L16.7462 26.9633C16.3735 26.5906 16.0778 26.1481 15.876 25.661C15.6743 25.174 15.5704 24.652 15.5704 24.1248C15.5704 23.5976 15.6743 23.0756 15.876 22.5886C16.0778 22.1016 16.3735 21.659 16.7462 21.2863L22.065 15.9675L21.4278 15.3313C21.2771 15.1806 21.1925 14.9763 21.1925 14.7633C21.1925 14.5502 21.2771 14.3459 21.4278 14.1953C21.5784 14.0446 21.7827 13.96 21.9958 13.96C22.2088 13.96 22.4131 14.0446 22.5638 14.1953L25.2041 16.8426L28.6503 13.3954C28.8009 13.2448 29.0052 13.1602 29.2183 13.1602C29.4313 13.1602 29.6356 13.2448 29.7863 13.3954C29.9369 13.5461 30.0215 13.7504 30.0215 13.9634C30.0215 14.1765 29.9369 14.3808 29.7863 14.5314L26.3391 17.9776L30.0211 21.6596L33.4673 18.2124C33.6179 18.0618 33.8222 17.9772 34.0353 17.9772C34.2483 17.9772 34.4526 18.0618 34.6033 18.2124C34.7539 18.3631 34.8386 18.5674 34.8386 18.7804C34.8386 18.9935 34.7539 19.1978 34.6033 19.3484Z",fill:"#96F2FF"}),n.createElement("defs",null,n.createElement("linearGradient",{id:"paint0_linear_48_4503",x1:24,y1:0,x2:24,y2:48,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#2E4C55"}),n.createElement("stop",{offset:1,stopColor:"#303355"})),n.createElement("radialGradient",{id:"paint1_radial_48_4503",cx:0,cy:0,r:1,gradientUnits:"userSpaceOnUse",gradientTransform:"translate(48 -1.78814e-06) rotate(120.009) scale(34.6442 37.2185)"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:.619146,stopColor:"#96F2FF",stopOpacity:0})),n.createElement("linearGradient",{id:"paint2_linear_48_4503",x1:24,y1:0,x2:24,y2:48,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:1,stopColor:"#96F2FF",stopOpacity:0})))),connect_default=e=>n.createElement("svg",{width:48,height:48,viewBox:"0 0 48 48",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("rect",{width:48,height:48,rx:7.5,fill:"url(#paint0_linear_48_6367)"}),n.createElement("rect",{width:48,height:48,rx:7.5,fill:"url(#paint1_radial_48_6367)",fillOpacity:.4}),n.createElement("rect",{x:.375,y:.375,width:47.25,height:47.25,rx:7.125,stroke:"url(#paint2_linear_48_6367)",strokeOpacity:.5,strokeWidth:.75}),n.createElement("path",{d:"M35.5352 22.3778L25.6222 12.4648C25.4748 12.3175 25.2999 12.2006 25.1073 12.1208C24.9148 12.0411 24.7084 12 24.5 12C24.2916 12 24.0852 12.0411 23.8927 12.1208C23.7001 12.2006 23.5252 12.3175 23.3778 12.4648L13.4648 22.3778C13.3175 22.5252 13.2006 22.7001 13.1208 22.8927C13.0411 23.0852 13 23.2916 13 23.5C13 23.7084 13.0411 23.9148 13.1208 24.1073C13.2006 24.2999 13.3175 24.4748 13.4648 24.6222L23.3778 34.5352C23.5252 34.6825 23.7001 34.7994 23.8927 34.8792C24.0852 34.9589 24.2916 35 24.5 35C24.7084 35 24.9148 34.9589 25.1073 34.8792C25.2999 34.7994 25.4748 34.6825 25.6222 34.5352L35.5352 24.6222C35.6825 24.4748 35.7994 24.2999 35.8792 24.1073C35.9589 23.9148 36 23.7084 36 23.5C36 23.2916 35.9589 23.0852 35.8792 22.8927C35.7994 22.7001 35.6825 22.5252 35.5352 22.3778ZM28.8757 23.2758L26.5757 25.5757C26.4319 25.7196 26.2368 25.8004 26.0333 25.8004C25.8299 25.8004 25.6348 25.7196 25.4909 25.5757C25.3471 25.4319 25.2662 25.2368 25.2662 25.0333C25.2662 24.8299 25.3471 24.6348 25.4909 24.4909L26.4828 23.5H22.9667C22.56 23.5 22.17 23.6615 21.8825 23.9491C21.5949 24.2367 21.4333 24.6267 21.4333 25.0333V25.8C21.4333 26.0033 21.3526 26.1983 21.2088 26.3421C21.065 26.4859 20.87 26.5667 20.6667 26.5667C20.4634 26.5667 20.2684 26.4859 20.1246 26.3421C19.9808 26.1983 19.9 26.0033 19.9 25.8V25.0333C19.9 24.22 20.2231 23.44 20.7982 22.8649C21.3733 22.2898 22.1533 21.9667 22.9667 21.9667H26.4828L25.4909 20.9758C25.3471 20.8319 25.2662 20.6368 25.2662 20.4333C25.2662 20.2299 25.3471 20.0348 25.4909 19.8909C25.6348 19.7471 25.8299 19.6663 26.0333 19.6663C26.2368 19.6663 26.4319 19.7471 26.5757 19.8909L28.8757 22.1909C28.947 22.2621 29.0036 22.3467 29.0421 22.4398C29.0807 22.5328 29.1006 22.6326 29.1006 22.7333C29.1006 22.8341 29.0807 22.9339 29.0421 23.0269C29.0036 23.12 28.947 23.2045 28.8757 23.2758Z",fill:"#96F2FF"}),n.createElement("defs",null,n.createElement("linearGradient",{id:"paint0_linear_48_6367",x1:24,y1:0,x2:24,y2:48,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#2E4C55"}),n.createElement("stop",{offset:1,stopColor:"#303355"})),n.createElement("radialGradient",{id:"paint1_radial_48_6367",cx:0,cy:0,r:1,gradientUnits:"userSpaceOnUse",gradientTransform:"translate(48 -1.78814e-06) rotate(120.009) scale(34.6442 37.2185)"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:.619146,stopColor:"#96F2FF",stopOpacity:0})),n.createElement("linearGradient",{id:"paint2_linear_48_6367",x1:24,y1:0,x2:24,y2:48,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:1,stopColor:"#96F2FF",stopOpacity:0})))),white_arrow_right_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M3.21986 11.4694L9.96986 4.71944C10.1106 4.57871 10.3015 4.49965 10.5005 4.49965C10.6995 4.49965 10.8904 4.57871 11.0311 4.71944C11.1718 4.86017 11.2509 5.05105 11.2509 5.25007C11.2509 5.44909 11.1718 5.63996 11.0311 5.7807L5.5608 11.2501L20.2505 11.2501C20.4494 11.2501 20.6402 11.3291 20.7808 11.4697C20.9215 11.6104 21.0005 11.8012 21.0005 12.0001C21.0005 12.199 20.9215 12.3897 20.7808 12.5304C20.6402 12.6711 20.4494 12.7501 20.2505 12.7501L5.5608 12.7501L11.0311 18.2194C11.1718 18.3602 11.2509 18.551 11.2509 18.7501C11.2509 18.9491 11.1718 19.14 11.0311 19.2807C10.8904 19.4214 10.6995 19.5005 10.5005 19.5005C10.3015 19.5005 10.1106 19.4214 9.96986 19.2807L3.21986 12.5307C3.15013 12.461 3.09481 12.3783 3.05707 12.2873C3.01933 12.1962 2.9999 12.0986 2.9999 12.0001C2.9999 11.9015 3.01933 11.8039 3.05707 11.7129C3.09481 11.6218 3.15013 11.5391 3.21986 11.4694Z",fill:"white"})),link_default=e=>n.createElement("svg",{width:24,height:25,viewBox:"0 0 24 25",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{filter:"url(#filter0_d_48_6611)"},n.createElement("path",{d:"M18 6.92102C18 7.05363 17.9473 7.18081 17.8536 7.27457C17.7598 7.36834 17.6326 7.42102 17.5 7.42102C17.3674 7.42102 17.2402 7.36834 17.1464 7.27457C17.0527 7.18081 17 7.05363 17 6.92102V4.12852L12.8544 8.27477C12.7606 8.36859 12.6333 8.4213 12.5006 8.4213C12.3679 8.4213 12.2407 8.36859 12.1469 8.27477C12.0531 8.18095 12.0003 8.0537 12.0003 7.92102C12.0003 7.78834 12.0531 7.66109 12.1469 7.56727L16.2925 3.42102H13.5C13.3674 3.42102 13.2402 3.36834 13.1464 3.27457C13.0527 3.18081 13 3.05363 13 2.92102C13 2.78841 13.0527 2.66124 13.1464 2.56747C13.2402 2.4737 13.3674 2.42102 13.5 2.42102H17.5C17.6326 2.42102 17.7598 2.4737 17.8536 2.56747C17.9473 2.66124 18 2.78841 18 2.92102V6.92102ZM15.5 8.42102C15.3674 8.42102 15.2402 8.4737 15.1464 8.56747C15.0527 8.66123 15 8.78841 15 8.92102V13.421H7V5.42102H11.5C11.6326 5.42102 11.7598 5.36834 11.8536 5.27457C11.9473 5.18081 12 5.05363 12 4.92102C12 4.78841 11.9473 4.66124 11.8536 4.56747C11.7598 4.4737 11.6326 4.42102 11.5 4.42102H7C6.73478 4.42102 6.48043 4.52638 6.29289 4.71391C6.10536 4.90145 6 5.1558 6 5.42102V13.421C6 13.6862 6.10536 13.9406 6.29289 14.1281C6.48043 14.3157 6.73478 14.421 7 14.421H15C15.2652 14.421 15.5196 14.3157 15.7071 14.1281C15.8946 13.9406 16 13.6862 16 13.421V8.92102C16 8.78841 15.9473 8.66123 15.8536 8.56747C15.7598 8.4737 15.6326 8.42102 15.5 8.42102Z",fill:"#96F2FF"})),n.createElement("defs",null,n.createElement("filter",{id:"filter0_d_48_6611",x:0,y:.421021,width:24,height:24,filterUnits:"userSpaceOnUse",colorInterpolationFilters:"sRGB"},n.createElement("feFlood",{floodOpacity:0,result:"BackgroundImageFix"}),n.createElement("feColorMatrix",{in:"SourceAlpha",type:"matrix",values:"0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 127 0",result:"hardAlpha"}),n.createElement("feOffset",{dy:4}),n.createElement("feGaussianBlur",{stdDeviation:2}),n.createElement("feComposite",{in2:"hardAlpha",operator:"out"}),n.createElement("feColorMatrix",{type:"matrix",values:"0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0.25 0"}),n.createElement("feBlend",{mode:"normal",in2:"BackgroundImageFix",result:"effect1_dropShadow_48_6611"}),n.createElement("feBlend",{mode:"normal",in:"SourceGraphic",in2:"effect1_dropShadow_48_6611",result:"shape"})))),green_check_default=e=>n.createElement("svg",{height:16,viewBox:"0 0 16 16",width:16,className:"octicon octicon-check v-align-middle",...e},n.createElement("path",{fill:"green",d:"M13.78 4.22a.75.75 0 0 1 0 1.06l-7.25 7.25a.75.75 0 0 1-1.06 0L2.22 9.28a.751.751 0 0 1 .018-1.042.751.751 0 0 1 1.042-.018L6 10.94l6.72-6.72a.75.75 0 0 1 1.06 0Z"})),red_error_default=e=>n.createElement("svg",{fill:"#f85249",height:16,viewBox:"0 0 16 16",width:16,className:"octicon octicon-check v-align-middle",...e},n.createElement("path",{className:"icon-path",d:"M11.383 13.644A1.03 1.03 0 0 1 9.928 15.1L6 11.172 2.072 15.1a1.03 1.03 0 1 1-1.455-1.456l3.928-3.928L.617 5.79a1.03 1.03 0 1 1 1.455-1.456L6 8.261l3.928-3.928a1.03 1.03 0 0 1 1.455 1.456L7.455 9.716z"})),blue_info_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M12 7.01002L12 7.00003M12 17L12 10",stroke:"#2196F3",strokeWidth:1.5,strokeLinecap:"round",strokeLinejoin:"round"}))),bell_default=e=>n.createElement("svg",{viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{fillRule:"evenodd",clipRule:"evenodd",d:"M12 1.25C7.71983 1.25 4.25004 4.71979 4.25004 9V9.7041C4.25004 10.401 4.04375 11.0824 3.65717 11.6622L2.50856 13.3851C1.17547 15.3848 2.19318 18.1028 4.51177 18.7351C5.26738 18.9412 6.02937 19.1155 6.79578 19.2581L6.79768 19.2632C7.56667 21.3151 9.62198 22.75 12 22.75C14.378 22.75 16.4333 21.3151 17.2023 19.2632L17.2042 19.2581C17.9706 19.1155 18.7327 18.9412 19.4883 18.7351C21.8069 18.1028 22.8246 15.3848 21.4915 13.3851L20.3429 11.6622C19.9563 11.0824 19.75 10.401 19.75 9.7041V9C19.75 4.71979 16.2802 1.25 12 1.25ZM15.3764 19.537C13.1335 19.805 10.8664 19.8049 8.62349 19.5369C9.33444 20.5585 10.571 21.25 12 21.25C13.4289 21.25 14.6655 20.5585 15.3764 19.537ZM5.75004 9C5.75004 5.54822 8.54826 2.75 12 2.75C15.4518 2.75 18.25 5.54822 18.25 9V9.7041C18.25 10.6972 18.544 11.668 19.0948 12.4943L20.2434 14.2172C21.0086 15.3649 20.4245 16.925 19.0936 17.288C14.4494 18.5546 9.5507 18.5546 4.90644 17.288C3.57561 16.925 2.99147 15.3649 3.75664 14.2172L4.90524 12.4943C5.45609 11.668 5.75004 10.6972 5.75004 9.7041V9Z",fill:"#ffffff"}))),plus_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M14 8C14 8.13261 13.9473 8.25979 13.8536 8.35355C13.7598 8.44732 13.6326 8.5 13.5 8.5H8.5V13.5C8.5 13.6326 8.44732 13.7598 8.35355 13.8536C8.25979 13.9473 8.13261 14 8 14C7.86739 14 7.74021 13.9473 7.64645 13.8536C7.55268 13.7598 7.5 13.6326 7.5 13.5V8.5H2.5C2.36739 8.5 2.24021 8.44732 2.14645 8.35355C2.05268 8.25979 2 8.13261 2 8C2 7.86739 2.05268 7.74021 2.14645 7.64645C2.24021 7.55268 2.36739 7.5 2.5 7.5H7.5V2.5C7.5 2.36739 7.55268 2.24021 7.64645 2.14645C7.74021 2.05268 7.86739 2 8 2C8.13261 2 8.25979 2.05268 8.35355 2.14645C8.44732 2.24021 8.5 2.36739 8.5 2.5V7.5H13.5C13.6326 7.5 13.7598 7.55268 13.8536 7.64645C13.9473 7.74021 14 7.86739 14 8Z",fill:"#203548"})),back_default=e=>n.createElement("svg",{width:16,height:17,viewBox:"0 0 16 17",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{clipPath:"url(#clip0_106_437)"},n.createElement("path",{d:"M10 13.5L5 8.5L10 3.5",stroke:"white",strokeLinecap:"round",strokeLinejoin:"round"})),n.createElement("defs",null,n.createElement("clipPath",{id:"clip0_106_437"},n.createElement("rect",{width:16,height:16,fill:"white",transform:"translate(0 0.5)"})))),focus_overview_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M22.3725 7.37097C22.0941 7.65023 21.7633 7.87169 21.399 8.0226C21.0348 8.17352 20.6443 8.25092 20.25 8.25034C20.0086 8.25021 19.7682 8.22126 19.5337 8.16409L17.0371 12.801C17.0653 12.8272 17.0934 12.8535 17.1206 12.8807C17.5173 13.2774 17.7947 13.7775 17.9215 14.324C18.0482 14.8706 18.0192 15.4417 17.8376 15.9726C17.656 16.5034 17.3292 16.9728 16.8943 17.3272C16.4594 17.6817 15.9338 17.9071 15.3773 17.9778C14.8207 18.0485 14.2554 17.9618 13.7457 17.7274C13.2359 17.493 12.8021 17.1203 12.4936 16.6518C12.185 16.1832 12.014 15.6375 12 15.0766C11.986 14.5157 12.1295 13.9622 12.4143 13.4788L10.5225 11.5869C10.0609 11.8581 9.53526 12.0008 8.99996 12.0003C8.75836 12.0005 8.51759 11.9719 8.28277 11.915L5.78621 16.55C5.81434 16.5763 5.8434 16.6025 5.87059 16.6297C6.29008 17.0493 6.57573 17.5839 6.69143 18.1658C6.80713 18.7477 6.74768 19.3509 6.5206 19.899C6.29351 20.4471 5.909 20.9156 5.41566 21.2453C4.92233 21.5749 4.34234 21.7508 3.74902 21.7508C3.15571 21.7508 2.57572 21.5749 2.08239 21.2453C1.58905 20.9156 1.20453 20.4471 0.977452 19.899C0.750369 19.3509 0.690919 18.7477 0.806619 18.1658C0.922319 17.5839 1.20797 17.0493 1.62746 16.6297C1.99247 16.2649 2.44543 16.0004 2.94255 15.8618C3.43967 15.7231 3.96415 15.7151 4.46527 15.8385L6.96184 11.2016C6.93371 11.1753 6.90559 11.1491 6.8784 11.1219C6.59975 10.8433 6.37871 10.5126 6.2279 10.1486C6.0771 9.78453 5.99948 9.39437 5.99948 9.00034C5.99948 8.60632 6.0771 8.21616 6.2279 7.85213C6.37871 7.48811 6.59975 7.15736 6.8784 6.87878C7.39453 6.3612 8.08186 6.04985 8.81131 6.00321C9.54077 5.95658 10.2622 6.17787 10.84 6.62551C11.4178 7.07316 11.8123 7.71637 11.9495 8.43434C12.0866 9.15231 11.9569 9.89563 11.5847 10.5247L13.4765 12.4166C14.1525 12.0205 14.956 11.9029 15.7171 12.0885L18.2137 7.45159C18.1856 7.42534 18.1565 7.39909 18.1293 7.3719C17.8507 7.09332 17.6297 6.76257 17.4788 6.39855C17.328 6.03453 17.2504 5.64437 17.2504 5.25034C17.2504 4.85632 17.328 4.46616 17.4788 4.10213C17.6297 3.73811 17.8507 3.40736 18.1293 3.12878C18.692 2.56611 19.4552 2.25 20.2509 2.25C21.0466 2.25 21.8098 2.56611 22.3725 3.12878C22.9351 3.69145 23.2512 4.4546 23.2512 5.25034C23.2512 6.04608 22.9351 6.80923 22.3725 7.3719V7.37097Z",fill:"#0EE6F3"})),unfocus_overview_default=e=>n.createElement("svg",{width:24,height:20,viewBox:"0 0 24 20",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M22.3725 5.37048C22.0941 5.64974 21.7633 5.8712 21.399 6.02211C21.0348 6.17303 20.6443 6.25043 20.25 6.24985C20.0086 6.24972 19.7682 6.22077 19.5337 6.1636L17.0371 10.8005C17.0653 10.8267 17.0934 10.853 17.1206 10.8802C17.5173 11.2769 17.7947 11.777 17.9215 12.3235C18.0482 12.8701 18.0192 13.4412 17.8376 13.9721C17.656 14.503 17.3292 14.9723 16.8943 15.3267C16.4594 15.6812 15.9338 15.9066 15.3773 15.9773C14.8207 16.048 14.2554 15.9613 13.7457 15.7269C13.2359 15.4925 12.8021 15.1198 12.4936 14.6513C12.185 14.1827 12.014 13.637 12 13.0761C11.986 12.5152 12.1295 11.9617 12.4143 11.4783L10.5225 9.58642C10.0609 9.85759 9.53526 10.0003 8.99996 9.99985C8.75836 10 8.51759 9.9714 8.28277 9.91454L5.78621 14.5495C5.81434 14.5758 5.8434 14.602 5.87059 14.6292C6.29008 15.0488 6.57573 15.5834 6.69143 16.1653C6.80713 16.7472 6.74768 17.3504 6.5206 17.8985C6.29351 18.4467 5.909 18.9151 5.41566 19.2448C4.92233 19.5744 4.34234 19.7503 3.74902 19.7503C3.15571 19.7503 2.57572 19.5744 2.08239 19.2448C1.58905 18.9151 1.20453 18.4467 0.977452 17.8985C0.750369 17.3504 0.690919 16.7472 0.806619 16.1653C0.922319 15.5834 1.20797 15.0488 1.62746 14.6292C1.99247 14.2644 2.44543 13.9999 2.94255 13.8613C3.43967 13.7227 3.96415 13.7146 4.46527 13.838L6.96184 9.20111C6.93371 9.17485 6.90559 9.1486 6.8784 9.12142C6.59975 8.84283 6.37871 8.51208 6.2279 8.14806C6.0771 7.78404 5.99948 7.39388 5.99948 6.99985C5.99948 6.60583 6.0771 6.21567 6.2279 5.85165C6.37871 5.48762 6.59975 5.15688 6.8784 4.87829C7.39453 4.36071 8.08186 4.04936 8.81131 4.00273C9.54077 3.95609 10.2622 4.17738 10.84 4.62503C11.4178 5.07267 11.8123 5.71588 11.9495 6.43385C12.0866 7.15182 11.9569 7.89515 11.5847 8.52423L13.4765 10.4161C14.1525 10.02 14.956 9.90236 15.7171 10.088L18.2137 5.4511C18.1856 5.42485 18.1565 5.3986 18.1293 5.37142C17.8507 5.09283 17.6297 4.76208 17.4788 4.39806C17.328 4.03404 17.2504 3.64388 17.2504 3.24985C17.2504 2.85583 17.328 2.46567 17.4788 2.10165C17.6297 1.73762 17.8507 1.40688 18.1293 1.12829C18.692 0.565618 19.4552 0.249512 20.2509 0.249512C21.0466 0.249512 21.8098 0.565618 22.3725 1.12829C22.9351 1.69096 23.2512 2.45411 23.2512 3.24985C23.2512 4.04559 22.9351 4.80874 22.3725 5.37142V5.37048Z",fill:"#8B92A5"})),sources_focus_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M22.0302 7.7808L18.81 11L21.2802 13.4693C21.3499 13.539 21.4052 13.6217 21.4429 13.7128C21.4806 13.8038 21.5 13.9014 21.5 13.9999C21.5 14.0985 21.4806 14.196 21.4429 14.2871C21.4052 14.3781 21.3499 14.4608 21.2802 14.5305C21.2106 14.6002 21.1278 14.6555 21.0368 14.6932C20.9458 14.7309 20.8482 14.7503 20.7496 14.7503C20.6511 14.7503 20.5535 14.7309 20.4625 14.6932C20.3714 14.6555 20.2887 14.6002 20.219 14.5305L19.6247 13.9352L14.6561 18.9038C14.3079 19.252 13.8945 19.5282 13.4396 19.7167C12.9846 19.9052 12.497 20.0022 12.0045 20.0022C11.5121 20.0022 11.0245 19.9052 10.5695 19.7167C10.1145 19.5282 9.70113 19.252 9.35293 18.9038L7.88299 17.4282L3.28099 22.0302C3.21131 22.0999 3.12859 22.1552 3.03755 22.1929C2.94651 22.2306 2.84893 22.25 2.75039 22.25C2.65184 22.25 2.55427 22.2306 2.46323 22.1929C2.37218 22.1552 2.28946 22.0999 2.21978 22.0302C2.1501 21.9605 2.09483 21.8778 2.05712 21.7868C2.01941 21.6957 2 21.5982 2 21.4996C2 21.4011 2.01941 21.3035 2.05712 21.2125C2.09483 21.1214 2.1501 21.0387 2.21978 20.969L6.82178 16.367L5.34903 14.8943C5.0008 14.5461 4.72457 14.1327 4.53611 13.6777C4.34766 13.2227 4.25066 12.7351 4.25066 12.2427C4.25066 11.7502 4.34766 11.2626 4.53611 10.8076C4.72457 10.3526 5.0008 9.93925 5.34903 9.59104L10.3176 4.6225L9.72229 4.02815C9.58156 3.88742 9.5025 3.69656 9.5025 3.49754C9.5025 3.29853 9.58156 3.10766 9.72229 2.96694C9.86301 2.82621 10.0539 2.74716 10.2529 2.74716C10.4519 2.74716 10.6428 2.82621 10.7835 2.96694L13.25 5.43996L16.4692 2.21978C16.6099 2.07906 16.8008 2 16.9998 2C17.1988 2 17.3897 2.07906 17.5304 2.21978C17.6711 2.36051 17.7502 2.55137 17.7502 2.75039C17.7502 2.9494 17.6711 3.14026 17.5304 3.28099L14.3102 6.50023L17.7498 9.93978L20.969 6.7196C21.1097 6.57887 21.3006 6.49981 21.4996 6.49981C21.6986 6.49981 21.8895 6.57887 22.0302 6.7196C22.1709 6.86032 22.25 7.05119 22.25 7.2502C22.25 7.44922 22.1709 7.64008 22.0302 7.7808Z",fill:"#96F2FF"})),sources_unfocus_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M22.0302 7.7808L18.81 11L21.2802 13.4693C21.3499 13.539 21.4052 13.6217 21.4429 13.7128C21.4806 13.8038 21.5 13.9014 21.5 13.9999C21.5 14.0985 21.4806 14.196 21.4429 14.2871C21.4052 14.3781 21.3499 14.4608 21.2802 14.5305C21.2106 14.6002 21.1278 14.6555 21.0368 14.6932C20.9458 14.7309 20.8482 14.7503 20.7496 14.7503C20.6511 14.7503 20.5535 14.7309 20.4625 14.6932C20.3714 14.6555 20.2887 14.6002 20.219 14.5305L19.6247 13.9352L14.6561 18.9038C14.3079 19.252 13.8945 19.5282 13.4396 19.7167C12.9846 19.9052 12.497 20.0022 12.0045 20.0022C11.5121 20.0022 11.0245 19.9052 10.5695 19.7167C10.1145 19.5282 9.70113 19.252 9.35293 18.9038L7.88299 17.4282L3.28099 22.0302C3.21131 22.0999 3.12859 22.1552 3.03755 22.1929C2.94651 22.2306 2.84893 22.25 2.75039 22.25C2.65184 22.25 2.55427 22.2306 2.46323 22.1929C2.37218 22.1552 2.28946 22.0999 2.21978 22.0302C2.1501 21.9605 2.09483 21.8778 2.05712 21.7868C2.01941 21.6957 2 21.5982 2 21.4996C2 21.4011 2.01941 21.3035 2.05712 21.2125C2.09483 21.1214 2.1501 21.0387 2.21978 20.969L6.82178 16.367L5.34903 14.8943C5.0008 14.5461 4.72457 14.1327 4.53611 13.6777C4.34766 13.2227 4.25066 12.7351 4.25066 12.2427C4.25066 11.7502 4.34766 11.2626 4.53611 10.8076C4.72457 10.3526 5.0008 9.93925 5.34903 9.59104L10.3176 4.6225L9.72229 4.02815C9.58156 3.88742 9.5025 3.69656 9.5025 3.49754C9.5025 3.29853 9.58156 3.10766 9.72229 2.96694C9.86301 2.82621 10.0539 2.74716 10.2529 2.74716C10.4519 2.74716 10.6428 2.82621 10.7835 2.96694L13.25 5.43996L16.4692 2.21978C16.6099 2.07906 16.8008 2 16.9998 2C17.1988 2 17.3897 2.07906 17.5304 2.21978C17.6711 2.36051 17.7502 2.55137 17.7502 2.75039C17.7502 2.9494 17.6711 3.14026 17.5304 3.28099L14.3102 6.50023L17.7498 9.93978L20.969 6.7196C21.1097 6.57887 21.3006 6.49981 21.4996 6.49981C21.6986 6.49981 21.8895 6.57887 22.0302 6.7196C22.1709 6.86032 22.25 7.05119 22.25 7.2502C22.25 7.44922 22.1709 7.64008 22.0302 7.7808Z",fill:"#8B92A5"})),destinations_focus_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M23.0453 11.1522L13.3478 1.45474C13.2036 1.31057 13.0325 1.19621 12.8441 1.11818C12.6558 1.04016 12.4539 1 12.25 1C12.0461 1 11.8442 1.04016 11.6559 1.11818C11.4675 1.19621 11.2964 1.31057 11.1522 1.45474L1.45474 11.1522C1.31057 11.2964 1.19621 11.4675 1.11818 11.6559C1.04016 11.8442 1 12.0461 1 12.25C1 12.4539 1.04016 12.6558 1.11818 12.8441C1.19621 13.0325 1.31057 13.2036 1.45474 13.3478L11.1522 23.0453C11.2964 23.1894 11.4675 23.3038 11.6559 23.3818C11.8442 23.4598 12.0461 23.5 12.25 23.5C12.4539 23.5 12.6558 23.4598 12.8441 23.3818C13.0325 23.3038 13.2036 23.1894 13.3478 23.0453L23.0453 13.3478C23.1894 13.2036 23.3038 13.0325 23.3818 12.8441C23.4598 12.6558 23.5 12.4539 23.5 12.25C23.5 12.0461 23.4598 11.8442 23.3818 11.6559C23.3038 11.4675 23.1894 11.2964 23.0453 11.1522ZM16.5306 12.0306L14.2806 14.2806C14.1399 14.4213 13.949 14.5004 13.75 14.5004C13.551 14.5004 13.3601 14.4213 13.2194 14.2806C13.0786 14.1399 12.9996 13.949 12.9996 13.75C12.9996 13.551 13.0786 13.3601 13.2194 13.2194L14.1897 12.25H10.75C10.3522 12.25 9.97066 12.408 9.68935 12.6893C9.40805 12.9706 9.25002 13.3522 9.25002 13.75V14.5C9.25002 14.6989 9.171 14.8897 9.03035 15.0303C8.8897 15.171 8.69893 15.25 8.50002 15.25C8.30111 15.25 8.11034 15.171 7.96969 15.0303C7.82904 14.8897 7.75002 14.6989 7.75002 14.5V13.75C7.75002 12.9543 8.06609 12.1913 8.6287 11.6287C9.1913 11.0661 9.95436 10.75 10.75 10.75H14.1897L13.2194 9.78064C13.0786 9.63991 12.9996 9.44904 12.9996 9.25002C12.9996 9.05099 13.0786 8.86012 13.2194 8.71939C13.3601 8.57866 13.551 8.4996 13.75 8.4996C13.949 8.4996 14.1399 8.57866 14.2806 8.71939L16.5306 10.9694C16.6003 11.039 16.6557 11.1218 16.6934 11.2128C16.7311 11.3038 16.7506 11.4014 16.7506 11.5C16.7506 11.5986 16.7311 11.6962 16.6934 11.7872C16.6557 11.8783 16.6003 11.961 16.5306 12.0306Z",fill:"#0EE6F3"})),destinations_unfocus_default=e=>n.createElement("svg",{width:24,height:24,viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M23.0453 11.1522L13.3478 1.45474C13.2036 1.31057 13.0325 1.19621 12.8441 1.11818C12.6558 1.04016 12.4539 1 12.25 1C12.0461 1 11.8442 1.04016 11.6559 1.11818C11.4675 1.19621 11.2964 1.31057 11.1522 1.45474L1.45474 11.1522C1.31057 11.2964 1.19621 11.4675 1.11818 11.6559C1.04016 11.8442 1 12.0461 1 12.25C1 12.4539 1.04016 12.6558 1.11818 12.8441C1.19621 13.0325 1.31057 13.2036 1.45474 13.3478L11.1522 23.0453C11.2964 23.1894 11.4675 23.3038 11.6559 23.3818C11.8442 23.4598 12.0461 23.5 12.25 23.5C12.4539 23.5 12.6558 23.4598 12.8441 23.3818C13.0325 23.3038 13.2036 23.1894 13.3478 23.0453L23.0453 13.3478C23.1894 13.2036 23.3038 13.0325 23.3818 12.8441C23.4598 12.6558 23.5 12.4539 23.5 12.25C23.5 12.0461 23.4598 11.8442 23.3818 11.6559C23.3038 11.4675 23.1894 11.2964 23.0453 11.1522ZM16.5306 12.0306L14.2806 14.2806C14.1399 14.4213 13.949 14.5004 13.75 14.5004C13.551 14.5004 13.3601 14.4213 13.2194 14.2806C13.0786 14.1399 12.9996 13.949 12.9996 13.75C12.9996 13.551 13.0786 13.3601 13.2194 13.2194L14.1897 12.25H10.75C10.3522 12.25 9.97066 12.408 9.68935 12.6893C9.40805 12.9706 9.25002 13.3522 9.25002 13.75V14.5C9.25002 14.6989 9.171 14.8897 9.03035 15.0303C8.8897 15.171 8.69893 15.25 8.50002 15.25C8.30111 15.25 8.11034 15.171 7.96969 15.0303C7.82904 14.8897 7.75002 14.6989 7.75002 14.5V13.75C7.75002 12.9543 8.06609 12.1913 8.6287 11.6287C9.1913 11.0661 9.95436 10.75 10.75 10.75H14.1897L13.2194 9.78064C13.0786 9.63991 12.9996 9.44904 12.9996 9.25002C12.9996 9.05099 13.0786 8.86012 13.2194 8.71939C13.3601 8.57866 13.551 8.4996 13.75 8.4996C13.949 8.4996 14.1399 8.57866 14.2806 8.71939L16.5306 10.9694C16.6003 11.039 16.6557 11.1218 16.6934 11.2128C16.7311 11.3038 16.7506 11.4014 16.7506 11.5C16.7506 11.5986 16.7311 11.6962 16.6934 11.7872C16.6557 11.8783 16.6003 11.961 16.5306 12.0306Z",fill:"#8B92A5"})),transform_focus_default=e=>n.createElement("svg",{width:"24px",height:"24px",viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.25 4.00003C10.25 3.69074 10.0602 3.41317 9.77191 3.30105C9.48366 3.18892 9.15614 3.26524 8.94715 3.49324L3.44715 9.49324C3.24617 9.71248 3.19374 10.0298 3.3135 10.302C3.43327 10.5743 3.70259 10.75 4.00002 10.75H20C20.4142 10.75 20.75 10.4142 20.75 10C20.75 9.58582 20.4142 9.25003 20 9.25003L10.25 9.25003V4.00003Z",fill:"#0ce6f3"}),n.createElement("path",{d:"M13.75 20L13.75 14.75H4.00002C3.5858 14.75 3.25002 14.4142 3.25002 14C3.25002 13.5858 3.5858 13.25 4.00002 13.25L20 13.25C20.2974 13.25 20.5668 13.4258 20.6865 13.698C20.8063 13.9703 20.7539 14.2876 20.5529 14.5068L15.0529 20.5068C14.8439 20.7348 14.5164 20.8111 14.2281 20.699C13.9399 20.5869 13.75 20.3093 13.75 20Z",fill:"#0ce6f3"})),transform_unfocus_default=e=>n.createElement("svg",{width:"24px",height:"24px",viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.25 4.00003C10.25 3.69074 10.0602 3.41317 9.77191 3.30105C9.48366 3.18892 9.15614 3.26524 8.94715 3.49324L3.44715 9.49324C3.24617 9.71248 3.19374 10.0298 3.3135 10.302C3.43327 10.5743 3.70259 10.75 4.00002 10.75H20C20.4142 10.75 20.75 10.4142 20.75 10C20.75 9.58582 20.4142 9.25003 20 9.25003L10.25 9.25003V4.00003Z",fill:"#8b92a6"}),n.createElement("path",{d:"M13.75 20L13.75 14.75H4.00002C3.5858 14.75 3.25002 14.4142 3.25002 14C3.25002 13.5858 3.5858 13.25 4.00002 13.25L20 13.25C20.2974 13.25 20.5668 13.4258 20.6865 13.698C20.8063 13.9703 20.7539 14.2876 20.5529 14.5068L15.0529 20.5068C14.8439 20.7348 14.5164 20.8111 14.2281 20.699C13.9399 20.5869 13.75 20.3093 13.75 20Z",fill:"#8b92a6"}));function useOnClickOutside(e,t){(0,n.useEffect)(()=>{let listener=r=>{let n=e?.current;if(n?.contains(r?.target))return null;t(r)};return document.addEventListener("mousedown",listener),document.addEventListener("touchstart",listener),()=>{document.removeEventListener("mousedown",listener),document.removeEventListener("touchstart",listener)}},[e,t])}var w={colors:{primary:"#07111A",secondary:"#0EE6F3",torquiz_light:"#96F2FF",dark:"#07111A",data_flow_bg:"#0E1C28",light_dark:"#132330",dark_blue:"#203548",light_grey:"#CCD0D2",blue_grey:"#374A5B",white:"#fff",error:"#FD3F3F",traces:"#4CAF50",logs:"#8B4513",metrics:"#FFD700"},text:{primary:"#07111A",secondary:"#0EE6F3",white:"#fff",light_grey:"#CCD0D2",grey:"#8b92a5",dark_button:"#0A1824"},font_family:{primary:"Inter"}};l.zo.div`
  border-radius: 12px;
  width: 100%;
  border: ${({theme:e})=>`1px solid ${e.colors.dark_blue}`};
  background: ${({theme:e})=>e.colors.dark};
  padding: 16px;
  text-align: start;
  gap: 10px;
  position: relative;
`,l.zo.p`
  font-family: 'IBM Plex Mono', monospace;
  width: 90%;
`,l.zo.span`
  position: absolute;
  right: 16px;
  top: 16px;
  cursor: pointer;
`,l.zo.div`
  display: flex;
  flex-direction: column;
  text-align: start;
  gap: 6px;
  width: 100%;
`;var y=l.ZP.div`
  display: inline-flex;
  position: relative;
  height: fit-content;
  flex-direction: column;
  border-radius: 24px;
  height: 100%;
  border: ${({selected:e,theme:t,type:r})=>`1px solid ${e?t.colors.secondary:"primary"===r?t.colors.dark_blue:"#374a5b"}`};
  background: ${({theme:e,type:t})=>"primary"===t?e.colors.dark:"#0E1C28"};
  box-shadow: ${({type:e})=>"primary"===e?"none":"0px -6px 16px 0px rgba(0, 0, 0, 0.25),4px 4px 16px 0px rgba(71, 231, 241, 0.05),-4px 4px 16px 0px rgba(71, 231, 241, 0.05)"};
`,b=(0,l.ZP)(y)`
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: fit-content;
  gap: 16px;
  padding: 24px 0px;
  box-shadow: none;
`;function Card({children:e,focus:t=!1,type:r="primary",header:l,...i}){return n.createElement(y,{selected:t||void 0,type:r,...i},l&&n.createElement(b,null,l?.body?l?.body():n.createElement(n.Fragment,null,n.createElement(Text,{size:20,weight:600},l?.title),n.createElement(Text,{size:14,color:"#CCD0D2"},l?.subtitle))),e)}var _=l.ZP.div`
  display: flex;
  padding: 4px 8px;
  align-items: flex-start;
  gap: 10px;
  border-radius: 10px;
  width: fit-content;
`;function Tag({title:e="",color:t="#033869"}){return n.createElement(_,{style:{backgroundColor:t}},n.createElement(Text,{weight:500,size:13,color:"#CCD0D2"},e))}var v=l.ZP.div`
  display: flex;
  padding: 8px 14px;
  align-items: flex-end;
  gap: 10px;
  border-radius: 16px;
  border: ${({theme:e,selected:t})=>`1px solid ${t?"transparent":e.colors.dark_blue}`};
  background: ${({theme:e,selected:t})=>t?e.colors.dark_blue:"transparent"};
`;function Tap({title:e="",tapped:t,children:r,style:l,onClick:i}){return n.createElement(v,{onClick:i,selected:t,style:{...l,cursor:i?"pointer":"auto"}},r,n.createElement(Text,{weight:400,size:14,color:t?"#CCD0D2":"#8B92A5",style:{cursor:i?"pointer":"auto"}},e))}var k=l.ZP.div`
  position: relative;
  z-index: 999;
  width: 100%;
  padding: 11px 4px;
  border-radius: 8px;
  cursor: pointer;
  border: ${({selected:e,theme:t})=>`1px solid  ${e?t.colors.white:t.colors.blue_grey}`};
  /* background: ${({theme:e})=>e.colors.dark}; */

  .dropdown-arrow {
    transform: rotate(0deg);
    transition: all 0.2s ease-in-out;
  }

  .dropdown-arrow.open {
    transform: rotate(180deg);
  }
`,L=l.ZP.div`
  padding: 0 12px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: ${({theme:e})=>e.text.white};
  font-size: 14px;
  font-family: ${({theme:e})=>e.font_family.primary}, sans-serif;
  font-weight: 400;
`,$=l.ZP.div`
  position: relative;
  z-index: 1000;
  display: flex;
  width: 100%;
  padding: 11px 4px;
  flex-direction: column;
  border-radius: 8px;
  border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  margin-top: 5px;
`,H=l.ZP.div`
  position: relative;

  z-index: 1000;
  width: 100%;
  max-height: 270px;
  overflow-y: scroll;
  scrollbar-width: none;
  :hover {
    background: ${({theme:e})=>e.colors.dark_blue};
  }
`,Z=l.ZP.div`
  display: flex;
  padding: 7px 12px;
  justify-content: space-between;
  align-items: center;
  border-radius: 8px;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`,V=l.ZP.div`
  margin-bottom: 8px;
  text-align: start;
`,F=l.ZP.div`
  position: relative;
  display: flex;
  width: 340px;
  padding: 9px 13px;
  gap: 10px;
  border-radius: 8px;
  border: ${({active:e,theme:t})=>`1px solid ${e?t.colors.white:t.colors.blue_grey}`};
  background: ${({active:e,theme:t})=>`${e?t.colors.dark:t.colors.light_dark}`};
  &:hover {
    border: ${({theme:e})=>`solid 1px ${e.colors.white}`};
  }
`,M=l.ZP.input`
  width: 85%;
  background: ${({active:e,theme:t})=>`${e?t.colors.dark:"transparent"}`};
  border: none;
  outline: none;
  color: ${({active:e,theme:t})=>`${e?t.colors.white:t.text.grey}`};
  font-size: 14px;
  font-family: ${({theme:e})=>e.font_family.primary}, sans-serif;
  font-weight: 400;
  &:focus {
    color: ${({theme:e})=>`solid 1px ${e.colors.white}`};
  }
`,glass_default=e=>n.createElement("svg",{width:18,height:18,viewBox:"0 0 18 18",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M16.1479 15.3519L12.6273 11.8321C13.6477 10.6071 14.1566 9.03577 14.048 7.44512C13.9394 5.85447 13.2217 4.36692 12.0443 3.29193C10.8668 2.21693 9.32029 1.63725 7.72635 1.67348C6.13241 1.7097 4.6138 2.35904 3.48642 3.48642C2.35904 4.6138 1.7097 6.13241 1.67348 7.72635C1.63725 9.32029 2.21693 10.8668 3.29193 12.0443C4.36692 13.2217 5.85447 13.9394 7.44512 14.048C9.03577 14.1566 10.6071 13.6477 11.8321 12.6273L15.3519 16.1479C15.4042 16.2001 15.4663 16.2416 15.5345 16.2699C15.6028 16.2982 15.676 16.3127 15.7499 16.3127C15.8238 16.3127 15.897 16.2982 15.9653 16.2699C16.0336 16.2416 16.0956 16.2001 16.1479 16.1479C16.2001 16.0956 16.2416 16.0336 16.2699 15.9653C16.2982 15.897 16.3127 15.8238 16.3127 15.7499C16.3127 15.676 16.2982 15.6028 16.2699 15.5345C16.2416 15.4663 16.2001 15.4042 16.1479 15.3519ZM2.81242 7.87492C2.81242 6.87365 3.10933 5.89487 3.6656 5.06234C4.22188 4.22982 5.01253 3.58094 5.93758 3.19778C6.86263 2.81461 7.88053 2.71435 8.86256 2.90969C9.84459 3.10503 10.7466 3.58718 11.4546 4.29519C12.1626 5.00319 12.6448 5.90524 12.8401 6.88727C13.0355 7.8693 12.9352 8.8872 12.5521 9.81225C12.1689 10.7373 11.52 11.528 10.6875 12.0842C9.85497 12.6405 8.87618 12.9374 7.87492 12.9374C6.53271 12.9359 5.24591 12.4021 4.29683 11.453C3.34775 10.5039 2.81391 9.21712 2.81242 7.87492Z",fill:"#8B92A5"})),X_default=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M12.8535 12.146C12.9 12.1925 12.9368 12.2476 12.962 12.3083C12.9871 12.369 13.0001 12.4341 13.0001 12.4998C13.0001 12.5655 12.9871 12.6305 12.962 12.6912C12.9368 12.7519 12.9 12.8071 12.8535 12.8535C12.8071 12.9 12.7519 12.9368 12.6912 12.962C12.6305 12.9871 12.5655 13.0001 12.4998 13.0001C12.4341 13.0001 12.369 12.9871 12.3083 12.962C12.2476 12.9368 12.1925 12.9 12.146 12.8535L7.99979 8.70666L3.85354 12.8535C3.75972 12.9474 3.63247 13.0001 3.49979 13.0001C3.36711 13.0001 3.23986 12.9474 3.14604 12.8535C3.05222 12.7597 2.99951 12.6325 2.99951 12.4998C2.99951 12.3671 3.05222 12.2399 3.14604 12.146L7.29291 7.99979L3.14604 3.85354C3.05222 3.75972 2.99951 3.63247 2.99951 3.49979C2.99951 3.36711 3.05222 3.23986 3.14604 3.14604C3.23986 3.05222 3.36711 2.99951 3.49979 2.99951C3.63247 2.99951 3.75972 3.05222 3.85354 3.14604L7.99979 7.29291L12.146 3.14604C12.2399 3.05222 12.3671 2.99951 12.4998 2.99951C12.6325 2.99951 12.7597 3.05222 12.8535 3.14604C12.9474 3.23986 13.0001 3.36711 13.0001 3.49979C13.0001 3.63247 12.9474 3.75972 12.8535 3.85354L8.70666 7.99979L12.8535 12.146Z",fill:"white"}));function SearchInput({placeholder:e="Search",value:t="",onChange:r=()=>{},loading:l=!1,containerStyle:i={},inputStyle:a={},showClear:o=!0}){let c=t?()=>r({target:{value:""}}):()=>{};return n.createElement(F,{active:!!t||void 0,style:{...i}},n.createElement(glass_default,null),n.createElement(M,{style:{...a},value:t,active:!!t||void 0,placeholder:e,onChange:r}),o&&n.createElement("div",{onClick:c}," ",n.createElement(X_default,{style:{cursor:"pointer"}})))}var question_default=e=>n.createElement("svg",{width:14,height:14,viewBox:"0 0 14 14",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M7.65625 9.84375C7.65625 9.97354 7.61776 10.1004 7.54565 10.2083C7.47354 10.3163 7.37105 10.4004 7.25114 10.45C7.13122 10.4997 6.99927 10.5127 6.87197 10.4874C6.74467 10.4621 6.62774 10.3996 6.53596 10.3078C6.44419 10.216 6.38168 10.0991 6.35636 9.97178C6.33104 9.84448 6.34404 9.71253 6.39371 9.59261C6.44338 9.4727 6.52749 9.37021 6.63541 9.2981C6.74333 9.22599 6.87021 9.1875 7 9.1875C7.17405 9.1875 7.34097 9.25664 7.46404 9.37971C7.58711 9.50278 7.65625 9.6697 7.65625 9.84375ZM7 3.9375C5.7936 3.9375 4.8125 4.8207 4.8125 5.90625V6.125C4.8125 6.24103 4.8586 6.35231 4.94064 6.43436C5.02269 6.51641 5.13397 6.5625 5.25 6.5625C5.36603 6.5625 5.47731 6.51641 5.55936 6.43436C5.64141 6.35231 5.6875 6.24103 5.6875 6.125V5.90625C5.6875 5.30469 6.27649 4.8125 7 4.8125C7.72352 4.8125 8.3125 5.30469 8.3125 5.90625C8.3125 6.50781 7.72352 7 7 7C6.88397 7 6.77269 7.04609 6.69064 7.12814C6.6086 7.21019 6.5625 7.32147 6.5625 7.4375V7.875C6.5625 7.99103 6.6086 8.10231 6.69064 8.18436C6.77269 8.26641 6.88397 8.3125 7 8.3125C7.11603 8.3125 7.22731 8.26641 7.30936 8.18436C7.39141 8.10231 7.4375 7.99103 7.4375 7.875V7.83562C8.435 7.65242 9.1875 6.85672 9.1875 5.90625C9.1875 4.8207 8.20641 3.9375 7 3.9375ZM12.6875 7C12.6875 8.12488 12.3539 9.2245 11.729 10.1598C11.104 11.0951 10.2158 11.8241 9.17651 12.2546C8.13726 12.685 6.99369 12.7977 5.89043 12.5782C4.78716 12.3588 3.77374 11.8171 2.97833 11.0217C2.18292 10.2263 1.64124 9.21284 1.42179 8.10958C1.20233 7.00631 1.31496 5.86274 1.74544 4.82349C2.17591 3.78423 2.90489 2.89597 3.8402 2.27102C4.7755 1.64607 5.87512 1.3125 7 1.3125C8.50793 1.31409 9.95365 1.91382 11.0199 2.98009C12.0862 4.04636 12.6859 5.49207 12.6875 7ZM11.8125 7C11.8125 6.04818 11.5303 5.11773 11.0014 4.32632C10.4726 3.53491 9.72104 2.91808 8.84167 2.55383C7.9623 2.18958 6.99466 2.09428 6.06113 2.27997C5.1276 2.46566 4.27009 2.92401 3.59705 3.59705C2.92401 4.27009 2.46566 5.12759 2.27997 6.06113C2.09428 6.99466 2.18959 7.9623 2.55383 8.84166C2.91808 9.72103 3.53491 10.4726 4.32632 11.0014C5.11773 11.5303 6.04818 11.8125 7 11.8125C8.27591 11.8111 9.49915 11.3036 10.4014 10.4014C11.3036 9.49915 11.8111 8.27591 11.8125 7Z",fill:"#96F2FF"})),S=l.ZP.div`
  display: inline-flex;
  align-items: center; // Align children and icon vertically
  position: relative;
`,P=l.ZP.div`
  margin-left: 8px;
  display: flex;
  align-items: center;
`,A=l.ZP.div`
  visibility: ${({isVisible:e})=>e?"visible":"hidden"};
  background-color: black;
  color: white;
  text-align: center;
  border-radius: 6px;
  padding: 5px 10px;
  max-width: 300px;
  width: 100%;
  text-align: left;
  /* Position the tooltip above the icon */
  position: absolute;
  z-index: 1;
  bottom: 100%;
  left: 50%;
  transform: translateX(-0%);
  margin-bottom: 5px; // Space between the tooltip and the icon

  /* Fade in animation */
  opacity: ${({isVisible:e})=>e?1:0};
  transition: opacity 0.3s;
`,Tooltip=({children:e,text:t,icon:r="?",showIcon:l=!0})=>{let[i,a]=(0,n.useState)(!1);return t?n.createElement(S,{onMouseEnter:()=>a(!0),onMouseLeave:()=>a(!1)},e,n.createElement(A,{isVisible:i},n.createElement(Text,{size:12,weight:600},t)),l&&n.createElement(P,null,n.createElement(question_default,null))):n.createElement(n.Fragment,null,e)},I={width:"90%",border:"none",background:"transparent"},T={background:"transparent"};function DropDown({data:e=[],onChange:t,width:r=260,value:l,label:i,tooltip:a,required:o,...c}){let[s,d]=(0,n.useState)(!1),[p,u]=(0,n.useState)(l||null),[C,m]=(0,n.useState)(!1),[g,h]=(0,n.useState)(""),f=(0,n.useRef)(null);(0,n.useEffect)(()=>{l&&u(l)},[l]),useOnClickOutside(f,()=>d(!1));let handleItemClick=e=>{t(e),u(e),h(""),d(!1)};return n.createElement(n.Fragment,null,i&&n.createElement(V,null,n.createElement(Tooltip,{text:a||""},n.createElement("div",{style:{display:"flex",gap:4}},n.createElement(Text,{size:14,weight:600},i),o&&n.createElement(Text,{size:14,weight:600},"*")))),n.createElement("div",{style:{height:37,width:r},ref:f},n.createElement(k,{selected:C,onMouseEnter:()=>m(!0),onMouseLeave:()=>m(!1),onClick:()=>d(!s),...c},n.createElement(L,null,p?p.label:"Select item",n.createElement(expand_arrow_default,{className:`dropdown-arrow ${s&&"open"}`}))),s&&n.createElement($,null,n.createElement(SearchInput,{value:g,onChange:e=>h(e.target.value),placeholder:"Search",containerStyle:I,inputStyle:T,showClear:!1}),n.createElement(H,null,(g?e?.filter(e=>e?.label.toLowerCase().includes(g.toLowerCase())):e).map(e=>n.createElement(Z,{key:e.id,onClick:t=>handleItemClick(e)},n.createElement(Text,null,e.label)))))))}var B=l.ZP.div`
  display: flex;
  align-items: center;
  gap: 8px;
`,R=l.ZP.div`
  position: relative;
  width: 30px;
  height: 16px;
  background-color: ${({active:e,theme:t})=>e?t.colors.secondary:t.text.grey};
  cursor: pointer;
  user-select: none;
  border-radius: 20px;
  padding: 2px;
  display: flex;
  justify-content: center;
  align-items: center;
`,z=l.ZP.span`
  display: flex;
  justify-content: center;
  align-items: center;
  box-sizing: border-box;
  width: 14px;
  height: 14px;
  cursor: pointer;
  color: #fff;
  background-color: ${({disabled:e,theme:t})=>e?t.text.white:t.text.light_grey};
  box-shadow: 0 2px 4px rgb(0, 0, 0, 0.25);
  border-radius: 100%;
  position: absolute;
  transition: all 0.2s ease;
  left: ${({disabled:e})=>e?18:2}px;
`;function Switch({toggle:e,handleToggleChange:t,style:r,label:l="Select All"}){return n.createElement(B,null,n.createElement(R,{active:e||void 0,onClick:t},n.createElement(z,{disabled:e||void 0})),l&&n.createElement(Text,{size:14},l))}var j=l.zo.div`
  display: flex;
  gap: 8px;
  align-items: center;
  cursor: ${({disabled:e})=>e?"not-allowed":"pointer"};
  pointer-events: ${({disabled:e})=>e?"none":"auto"};
  opacity: ${({disabled:e})=>e?"0.5":"1"};
`,G=l.zo.span`
  width: 16px;
  height: 16px;
  border: ${({theme:e})=>`solid 1px ${e.colors.light_grey}`};
  border-radius: 4px;
`,checkbox_rect_default=e=>n.createElement("svg",{width:18,height:18,viewBox:"0 0 18 18",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("rect",{x:.5,y:.5,width:17,height:17,rx:3.5,fill:"#96F2FF",stroke:"#96F2FF"}),n.createElement("path",{d:"M13.7727 6L7.39773 12.375L4.5 9.47727",stroke:"#132330",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"}));function Checkbox({onChange:e,value:t,label:r="",disabled:l=!1,...i}){return n.createElement(j,{disabled:l||void 0,onClick:e,...i},t?n.createElement(checkbox_rect_default,null):n.createElement(G,null),n.createElement(Text,{size:14},r))}l.ZP.div`
  display: flex;
  padding: 4px;
  align-items: center;
  gap: 4px;
  border-radius: 14px;
  background: ${({theme:e})=>e.colors.dark_blue};
`;var D=l.zo.div`
  cursor: pointer;
  .p {
    cursor: pointer !important;
  }
`;function Link({value:e,onClick:t,fontSize:r=16,color:l=w.colors.secondary}){return n.createElement(D,{onClick:t},n.createElement(Text,{size:r,color:l},e))}var O=l.zo.div`
  position: relative;
  display: flex;
  width: 100%;
  padding-left: 13px;
  height: 100%;
  min-height: 37px;
  align-items: center;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  gap: 10px;
  border-radius: 8px;
  border: ${({theme:e,error:t,active:r})=>`1px solid ${t?e.colors.error:r?e.text.grey:e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.light_dark};

  &:hover {
    border: ${({theme:e})=>`solid 1px ${e.text.grey}`};
  }
`;l.zo.div`
  position: relative;
  display: flex;
  width: 100%;
  padding: 0px 12px;
  height: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-radius: 4px;
  border: ${({theme:e})=>`1px solid ${e.colors.secondary}`};
`;var U=l.zo.input`
  background: transparent;
  border: none;
  outline: none;
  width: 96%;
  color: ${({theme:e})=>e.text.white};
`;(0,l.zo)(U)`
  color: var(--dark-mode-white, #fff);
  font-family: Inter, sans-serif;
  font-size: 24px;
`;var N=l.zo.div`
  margin-bottom: 8px;
  text-align: start;
`,W=l.zo.div`
  margin-top: 4px;
`,K=l.zo.div`
  position: absolute;
  right: 10px;
  cursor: pointer;
`,eye_open_default=e=>n.createElement("svg",{width:"800px",height:"800px",viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M3 14C3 9.02944 7.02944 5 12 5C16.9706 5 21 9.02944 21 14M17 14C17 16.7614 14.7614 19 12 19C9.23858 19 7 16.7614 7 14C7 11.2386 9.23858 9 12 9C14.7614 9 17 11.2386 17 14Z",stroke:"#fff",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"}))),eye_close_default=e=>n.createElement("svg",{width:"800px",height:"800px",viewBox:"0 0 24 24",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("g",{id:"SVGRepo_bgCarrier",strokeWidth:0}),n.createElement("g",{id:"SVGRepo_tracerCarrier",strokeLinecap:"round",strokeLinejoin:"round"}),n.createElement("g",{id:"SVGRepo_iconCarrier"},n.createElement("path",{d:"M9.60997 9.60714C8.05503 10.4549 7 12.1043 7 14C7 16.7614 9.23858 19 12 19C13.8966 19 15.5466 17.944 16.3941 16.3878M21 14C21 9.02944 16.9706 5 12 5C11.5582 5 11.1238 5.03184 10.699 5.09334M3 14C3 11.0069 4.46104 8.35513 6.70883 6.71886M3 3L21 21",stroke:"#fff",strokeWidth:2,strokeLinecap:"round",strokeLinejoin:"round"})));function Input({label:e,value:t,onChange:r,type:l="text",error:i="",style:a={},onKeyDown:o,tooltip:c,required:s,autoComplete:d="off",...p}){let[u,C]=(0,n.useState)(!1);return n.createElement("div",{style:{...a}},e&&n.createElement(N,null,n.createElement(Tooltip,{text:c||""},n.createElement("div",{style:{display:"flex",gap:4}},n.createElement(Text,{size:14,weight:600},e),s&&n.createElement(Text,{size:14,weight:600},"*")))),n.createElement(O,{active:!!t||void 0,error:!!i||void 0},n.createElement(U,{type:u?"text":l,value:t,onChange:function(e){r(e.target.value)},autoComplete:d,onKeyDown:o,...p}),"password"===l&&n.createElement(K,{onClick:()=>C(!u)},u?n.createElement(eye_close_default,{width:16,height:16}):n.createElement(eye_open_default,{width:16,height:16}))),i&&n.createElement(W,null,n.createElement(Text,{size:14,color:"#FD3F3F"},i)))}l.zo.div`
  position: relative;
  margin-top: 8px;
  border-radius: 8px;
  width: 240px;
  height: 140px;
  cursor: pointer;
  background: ${({url:e})=>`linear-gradient(
      0deg,
      rgba(2, 20, 30, 0.2) 0%,
      rgba(2, 20, 30, 0.2) 100%
    ),
    url(${e}),
    lightgray 50%`};
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
`;var Q=l.zo.div`
  position: absolute;
  margin-left: auto;
  margin-right: auto;
  left: 0;
  right: 0;
  top: 30px;
  text-align: center;
`;(0,l.zo)(Q)`
  top: 40%;
`,l.zo.video`
  width: 980px;
  border-radius: 8px;
`,l.zo.div`
  width: 980px;

  display: flex;
  justify-content: space-between;
  margin-bottom: 21px;
`,l.zo.div`
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background: rgba(0, 0, 0, 0.65);
  z-index: 9999;
`;var q=l.ZP.div`
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
`,Y=l.ZP.div`
  width: ${({width:e})=>e||48}px;
  height: ${({height:e})=>e||48}px;
  border: 4px solid;
  border-color: ${({theme:e})=>`${e.colors.secondary} ${e.colors.secondary} ${e.colors.secondary}  transparent`};
  border-radius: 50%;
  animation: spin-anim 1.2s linear infinite;

  @keyframes spin-anim {
    0% {
      transform: rotate(0deg);
    }
    100% {
      transform: rotate(360deg);
    }
  }
`;function Loader({width:e,height:t}){return n.createElement(q,null,n.createElement(Y,{width:e,height:t}))}l.ZP.div`
  position: fixed;
  top: 3%;
  right: 3%;
`,l.ZP.div`
  display: flex;
  padding: 6px 16px 6px 8px;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-radius: 8px;
  border: ${({theme:e})=>`1px solid ${e.colors.secondary}`};
  background: ${({theme:e})=>e.colors.dark_blue};
  svg {
    cursor: pointer;
  }
`;var X=l.ZP.div`
  display: flex;
  flex-direction: column;
  padding: 8px;
  border-radius: 12px;
  border: ${({theme:e})=>`solid 1px ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  align-items: center;
  gap: 4px;
  min-width: 80px;
`,J=l.ZP.div`
  max-width: 72px;
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
`,ee=l.ZP.span`
  background-color: ${({backgroundColor:e})=>e};
  width: 8px;
  height: 8px;
  border-radius: 8px;
`,et=l.ZP.div`
  width: 24px;
  height: 24px;
`,er=(0,n.memo)(({data:e,isConnectable:t})=>{let r=f[e.type]?f[e.type]:null;return n.createElement(X,null,n.createElement(i.HH,{type:"target",position:i.Ly.Left,id:"b",isConnectable:t,style:{visibility:"hidden"}}),r&&n.createElement(et,null,n.createElement(r,null)),n.createElement(J,null,n.createElement(Text,{size:14,weight:600},e?.spec?.actionName||"Action")),n.createElement("div",{style:{display:"flex",justifyContent:"center",alignItems:"center",gap:4,width:"100%"}},e.spec?.signals.map(e=>n.createElement(ee,{key:e,backgroundColor:w.colors[e.toLowerCase()]}))),n.createElement(i.HH,{type:"source",position:i.Ly.Right,id:"a",isConnectable:t,style:{visibility:"hidden"}}))}),en=l.F4`
  0% {
    opacity: 1;

  }
  100% {
    opacity: 0.5;
  }
`,el=l.ZP.div`
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 60px;
  position: relative;
  z-index: 90;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    border-radius: 60px;
    background: #110c1f55;
    z-index: -1;
    animation: ${en} 1s infinite alternate;
  }
`,ei=l.ZP.div`
  width: 100px;
  height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50px;
  border: solid 1px #3a3a3a76;
  background: #110c1f7d;
`,ea=l.ZP.div`
  width: 80px;
  height: 80px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 40px;
  border: solid 1px #3a3a3aa9;
  background: #110c1f;
  position: relative;
  z-index: 99;
`,folder_default=e=>n.createElement("svg",{width:32,height:32,viewBox:"0 0 32 32",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("rect",{width:32,height:32,rx:4,fill:"url(#paint0_linear_280_5350)"}),n.createElement("rect",{width:32,height:32,rx:4,fill:"url(#paint1_radial_280_5350)",fillOpacity:.4}),n.createElement("path",{opacity:.2,d:"M25.75 11.5V19.8334C25.75 20.0102 25.6798 20.1798 25.5548 20.3048C25.4298 20.4298 25.2602 20.5 25.0834 20.5H22.75V14.5C22.75 14.3011 22.671 14.1103 22.5303 13.9697C22.3897 13.829 22.1989 13.75 22 13.75H15.5003C15.338 13.75 15.1801 13.6974 15.0503 13.6L12.4497 11.65C12.3199 11.5526 12.162 11.5 11.9997 11.5H10V9.25C10 9.05109 10.079 8.86032 10.2197 8.71967C10.3603 8.57902 10.5511 8.5 10.75 8.5H14.9997C15.162 8.5 15.3199 8.55263 15.4497 8.65L18.0503 10.6C18.1801 10.6974 18.338 10.75 18.5003 10.75H25C25.1989 10.75 25.3897 10.829 25.5303 10.9697C25.671 11.1103 25.75 11.3011 25.75 11.5Z",fill:"#96F2FF"}),n.createElement("path",{d:"M25 10H18.5003L15.8997 8.05C15.6397 7.85599 15.3241 7.7508 14.9997 7.75H10.75C10.3522 7.75 9.97064 7.90804 9.68934 8.18934C9.40804 8.47064 9.25 8.85218 9.25 9.25V10.75H7.75C7.35218 10.75 6.97064 10.908 6.68934 11.1893C6.40804 11.4706 6.25 11.8522 6.25 12.25V22.75C6.25 23.1478 6.40804 23.5294 6.68934 23.8107C6.97064 24.092 7.35218 24.25 7.75 24.25H22.0834C22.459 24.2495 22.819 24.1001 23.0846 23.8346C23.3501 23.569 23.4995 23.209 23.5 22.8334V21.25H25.0834C25.459 21.2495 25.819 21.1001 26.0846 20.8346C26.3501 20.569 26.4995 20.209 26.5 19.8334V11.5C26.5 11.1022 26.342 10.7206 26.0607 10.4393C25.7794 10.158 25.3978 10 25 10ZM22 22.75H7.75V12.25H11.9997L14.6003 14.2C14.8603 14.394 15.1759 14.4992 15.5003 14.5H22V22.75ZM25 19.75H23.5V14.5C23.5 14.1022 23.342 13.7206 23.0607 13.4393C22.7794 13.158 22.3978 13 22 13H15.5003L12.8997 11.05C12.6397 10.856 12.3241 10.7508 11.9997 10.75H10.75V9.25H14.9997L17.6003 11.2C17.8603 11.394 18.1759 11.4992 18.5003 11.5H25V19.75Z",fill:"#96F2FF"}),n.createElement("rect",{x:.375,y:.375,width:31.25,height:31.25,rx:3.625,stroke:"url(#paint2_linear_280_5350)",strokeOpacity:.5,strokeWidth:.75}),n.createElement("defs",null,n.createElement("linearGradient",{id:"paint0_linear_280_5350",x1:16,y1:0,x2:16,y2:32,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#2E4C55"}),n.createElement("stop",{offset:1,stopColor:"#303355"})),n.createElement("radialGradient",{id:"paint1_radial_280_5350",cx:0,cy:0,r:1,gradientUnits:"userSpaceOnUse",gradientTransform:"translate(32 -1.19209e-06) rotate(120.009) scale(23.0961 24.8123)"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:.619146,stopColor:"#96F2FF",stopOpacity:0})),n.createElement("linearGradient",{id:"paint2_linear_280_5350",x1:16,y1:0,x2:16,y2:32,gradientUnits:"userSpaceOnUse"},n.createElement("stop",{stopColor:"#96F2FF"}),n.createElement("stop",{offset:1,stopColor:"#96F2FF",stopOpacity:0})))),eo=l.zo.div`
  display: flex;
  padding: 16px;
  border-radius: 12px;
  border: ${({theme:e})=>`solid 1px ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  align-items: center;
  width: 272px;
  gap: 8px;
`,ec=l.zo.div`
  gap: 10px;
`,es=(0,n.memo)(({data:e,isConnectable:t})=>n.createElement(eo,null,n.createElement(folder_default,{width:32}),n.createElement(ec,null,n.createElement(Text,{size:14,weight:600},e?.name),e?.totalAppsInstrumented&&n.createElement(Text,{color:"#8b92a5"},`${e.totalAppsInstrumented} Apps Instrumented`)),n.createElement(i.HH,{type:"source",position:i.Ly.Right,id:"a",isConnectable:t,style:{visibility:"hidden"}}))),logs_grey_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M2 4C2 3.86739 2.05268 3.74021 2.14645 3.64645C2.24021 3.55268 2.36739 3.5 2.5 3.5H13.5C13.6326 3.5 13.7598 3.55268 13.8536 3.64645C13.9473 3.74021 14 3.86739 14 4C14 4.13261 13.9473 4.25979 13.8536 4.35355C13.7598 4.44732 13.6326 4.5 13.5 4.5H2.5C2.36739 4.5 2.24021 4.44732 2.14645 4.35355C2.05268 4.25979 2 4.13261 2 4ZM2.5 7H10.5C10.6326 7 10.7598 6.94732 10.8536 6.85355C10.9473 6.75979 11 6.63261 11 6.5C11 6.36739 10.9473 6.24021 10.8536 6.14645C10.7598 6.05268 10.6326 6 10.5 6H2.5C2.36739 6 2.24021 6.05268 2.14645 6.14645C2.05268 6.24021 2 6.36739 2 6.5C2 6.63261 2.05268 6.75979 2.14645 6.85355C2.24021 6.94732 2.36739 7 2.5 7ZM13.5 8.5H2.5C2.36739 8.5 2.24021 8.55268 2.14645 8.64645C2.05268 8.74021 2 8.86739 2 9C2 9.13261 2.05268 9.25979 2.14645 9.35355C2.24021 9.44732 2.36739 9.5 2.5 9.5H13.5C13.6326 9.5 13.7598 9.44732 13.8536 9.35355C13.9473 9.25979 14 9.13261 14 9C14 8.86739 13.9473 8.74021 13.8536 8.64645C13.7598 8.55268 13.6326 8.5 13.5 8.5ZM10.5 11H2.5C2.36739 11 2.24021 11.0527 2.14645 11.1464C2.05268 11.2402 2 11.3674 2 11.5C2 11.6326 2.05268 11.7598 2.14645 11.8536C2.24021 11.9473 2.36739 12 2.5 12H10.5C10.6326 12 10.7598 11.9473 10.8536 11.8536C10.9473 11.7598 11 11.6326 11 11.5C11 11.3674 10.9473 11.2402 10.8536 11.1464C10.7598 11.0527 10.6326 11 10.5 11Z",fill:"#8B92A5"})),logs_blue_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M2 4C2 3.86739 2.05268 3.74021 2.14645 3.64645C2.24021 3.55268 2.36739 3.5 2.5 3.5H13.5C13.6326 3.5 13.7598 3.55268 13.8536 3.64645C13.9473 3.74021 14 3.86739 14 4C14 4.13261 13.9473 4.25979 13.8536 4.35355C13.7598 4.44732 13.6326 4.5 13.5 4.5H2.5C2.36739 4.5 2.24021 4.44732 2.14645 4.35355C2.05268 4.25979 2 4.13261 2 4ZM2.5 7H10.5C10.6326 7 10.7598 6.94732 10.8536 6.85355C10.9473 6.75979 11 6.63261 11 6.5C11 6.36739 10.9473 6.24021 10.8536 6.14645C10.7598 6.05268 10.6326 6 10.5 6H2.5C2.36739 6 2.24021 6.05268 2.14645 6.14645C2.05268 6.24021 2 6.36739 2 6.5C2 6.63261 2.05268 6.75979 2.14645 6.85355C2.24021 6.94732 2.36739 7 2.5 7ZM13.5 8.5H2.5C2.36739 8.5 2.24021 8.55268 2.14645 8.64645C2.05268 8.74021 2 8.86739 2 9C2 9.13261 2.05268 9.25979 2.14645 9.35355C2.24021 9.44732 2.36739 9.5 2.5 9.5H13.5C13.6326 9.5 13.7598 9.44732 13.8536 9.35355C13.9473 9.25979 14 9.13261 14 9C14 8.86739 13.9473 8.74021 13.8536 8.64645C13.7598 8.55268 13.6326 8.5 13.5 8.5ZM10.5 11H2.5C2.36739 11 2.24021 11.0527 2.14645 11.1464C2.05268 11.2402 2 11.3674 2 11.5C2 11.6326 2.05268 11.7598 2.14645 11.8536C2.24021 11.9473 2.36739 12 2.5 12H10.5C10.6326 12 10.7598 11.9473 10.8536 11.8536C10.9473 11.7598 11 11.6326 11 11.5C11 11.3674 10.9473 11.2402 10.8536 11.1464C10.7598 11.0527 10.6326 11 10.5 11Z",fill:"#96F2FF"})),chart_line_grey_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M14.5 13C14.5 13.1326 14.4473 13.2598 14.3536 13.3536C14.2598 13.4473 14.1326 13.5 14 13.5H2C1.86739 13.5 1.74021 13.4473 1.64645 13.3536C1.55268 13.2598 1.5 13.1326 1.5 13V3C1.5 2.86739 1.55268 2.74021 1.64645 2.64645C1.74021 2.55268 1.86739 2.5 2 2.5C2.13261 2.5 2.25979 2.55268 2.35355 2.64645C2.44732 2.74021 2.5 2.86739 2.5 3V8.89812L5.67063 6.125C5.7569 6.04947 5.86652 6.0059 5.9811 6.00157C6.09569 5.99725 6.20828 6.03244 6.3 6.10125L9.97563 8.85812L13.6706 5.625C13.7191 5.57704 13.7768 5.5395 13.8403 5.51467C13.9038 5.48985 13.9717 5.47827 14.0398 5.48065C14.1079 5.48303 14.1749 5.49931 14.2365 5.5285C14.2981 5.55769 14.3531 5.59917 14.398 5.65038C14.443 5.7016 14.4771 5.76148 14.4981 5.82633C14.5191 5.89119 14.5266 5.95965 14.5201 6.02752C14.5137 6.09538 14.4935 6.16122 14.4607 6.22097C14.4279 6.28073 14.3832 6.33314 14.3294 6.375L10.3294 9.875C10.2431 9.95053 10.1335 9.9941 10.0189 9.99843C9.90431 10.0028 9.79172 9.96756 9.7 9.89875L6.02437 7.14313L2.5 10.2269V12.5H14C14.1326 12.5 14.2598 12.5527 14.3536 12.6464C14.4473 12.7402 14.5 12.8674 14.5 13Z",fill:"#8B92A5"})),chart_line_blue_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M14.5 13C14.5 13.1326 14.4473 13.2598 14.3536 13.3536C14.2598 13.4473 14.1326 13.5 14 13.5H2C1.86739 13.5 1.74021 13.4473 1.64645 13.3536C1.55268 13.2598 1.5 13.1326 1.5 13V3C1.5 2.86739 1.55268 2.74021 1.64645 2.64645C1.74021 2.55268 1.86739 2.5 2 2.5C2.13261 2.5 2.25979 2.55268 2.35355 2.64645C2.44732 2.74021 2.5 2.86739 2.5 3V8.89812L5.67063 6.125C5.7569 6.04947 5.86652 6.0059 5.9811 6.00157C6.09569 5.99725 6.20828 6.03244 6.3 6.10125L9.97563 8.85812L13.6706 5.625C13.7191 5.57704 13.7768 5.5395 13.8403 5.51467C13.9038 5.48985 13.9717 5.47827 14.0398 5.48065C14.1079 5.48303 14.1749 5.49931 14.2365 5.5285C14.2981 5.55769 14.3531 5.59917 14.398 5.65038C14.443 5.7016 14.4771 5.76148 14.4981 5.82633C14.5191 5.89119 14.5266 5.95965 14.5201 6.02752C14.5137 6.09538 14.4935 6.16122 14.4607 6.22097C14.4279 6.28073 14.3832 6.33314 14.3294 6.375L10.3294 9.875C10.2431 9.95053 10.1335 9.9941 10.0189 9.99843C9.90431 10.0028 9.79172 9.96756 9.7 9.89875L6.02437 7.14313L2.5 10.2269V12.5H14C14.1326 12.5 14.2598 12.5527 14.3536 12.6464C14.4473 12.7402 14.5 12.8674 14.5 13Z",fill:"#96F2FF"})),tree_structure_grey_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.5 7H13.5C13.7652 7 14.0196 6.89464 14.2071 6.70711C14.3946 6.51957 14.5 6.26522 14.5 6V3C14.5 2.73478 14.3946 2.48043 14.2071 2.29289C14.0196 2.10536 13.7652 2 13.5 2H10.5C10.2348 2 9.98043 2.10536 9.79289 2.29289C9.60536 2.48043 9.5 2.73478 9.5 3V4H9C8.46957 4 7.96086 4.21071 7.58579 4.58579C7.21071 4.96086 7 5.46957 7 6V7.5H5V7C5 6.73478 4.89464 6.48043 4.70711 6.29289C4.51957 6.10536 4.26522 6 4 6H2C1.73478 6 1.48043 6.10536 1.29289 6.29289C1.10536 6.48043 1 6.73478 1 7V9C1 9.26522 1.10536 9.51957 1.29289 9.70711C1.48043 9.89464 1.73478 10 2 10H4C4.26522 10 4.51957 9.89464 4.70711 9.70711C4.89464 9.51957 5 9.26522 5 9V8.5H7V10C7 10.5304 7.21071 11.0391 7.58579 11.4142C7.96086 11.7893 8.46957 12 9 12H9.5V13C9.5 13.2652 9.60536 13.5196 9.79289 13.7071C9.98043 13.8946 10.2348 14 10.5 14H13.5C13.7652 14 14.0196 13.8946 14.2071 13.7071C14.3946 13.5196 14.5 13.2652 14.5 13V10C14.5 9.73478 14.3946 9.48043 14.2071 9.29289C14.0196 9.10536 13.7652 9 13.5 9H10.5C10.2348 9 9.98043 9.10536 9.79289 9.29289C9.60536 9.48043 9.5 9.73478 9.5 10V11H9C8.73478 11 8.48043 10.8946 8.29289 10.7071C8.10536 10.5196 8 10.2652 8 10V6C8 5.73478 8.10536 5.48043 8.29289 5.29289C8.48043 5.10536 8.73478 5 9 5H9.5V6C9.5 6.26522 9.60536 6.51957 9.79289 6.70711C9.98043 6.89464 10.2348 7 10.5 7ZM4 9H2V7H4V9ZM10.5 10H13.5V13H10.5V10ZM10.5 3H13.5V6H10.5V3Z",fill:"#8B92A5"})),tree_structure_blue_default2=e=>n.createElement("svg",{width:16,height:16,viewBox:"0 0 16 16",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M10.5 7H13.5C13.7652 7 14.0196 6.89464 14.2071 6.70711C14.3946 6.51957 14.5 6.26522 14.5 6V3C14.5 2.73478 14.3946 2.48043 14.2071 2.29289C14.0196 2.10536 13.7652 2 13.5 2H10.5C10.2348 2 9.98043 2.10536 9.79289 2.29289C9.60536 2.48043 9.5 2.73478 9.5 3V4H9C8.46957 4 7.96086 4.21071 7.58579 4.58579C7.21071 4.96086 7 5.46957 7 6V7.5H5V7C5 6.73478 4.89464 6.48043 4.70711 6.29289C4.51957 6.10536 4.26522 6 4 6H2C1.73478 6 1.48043 6.10536 1.29289 6.29289C1.10536 6.48043 1 6.73478 1 7V9C1 9.26522 1.10536 9.51957 1.29289 9.70711C1.48043 9.89464 1.73478 10 2 10H4C4.26522 10 4.51957 9.89464 4.70711 9.70711C4.89464 9.51957 5 9.26522 5 9V8.5H7V10C7 10.5304 7.21071 11.0391 7.58579 11.4142C7.96086 11.7893 8.46957 12 9 12H9.5V13C9.5 13.2652 9.60536 13.5196 9.79289 13.7071C9.98043 13.8946 10.2348 14 10.5 14H13.5C13.7652 14 14.0196 13.8946 14.2071 13.7071C14.3946 13.5196 14.5 13.2652 14.5 13V10C14.5 9.73478 14.3946 9.48043 14.2071 9.29289C14.0196 9.10536 13.7652 9 13.5 9H10.5C10.2348 9 9.98043 9.10536 9.79289 9.29289C9.60536 9.48043 9.5 9.73478 9.5 10V11H9C8.73478 11 8.48043 10.8946 8.29289 10.7071C8.10536 10.5196 8 10.2652 8 10V6C8 5.73478 8.10536 5.48043 8.29289 5.29289C8.48043 5.10536 8.73478 5 9 5H9.5V6C9.5 6.26522 9.60536 6.51957 9.79289 6.70711C9.98043 6.89464 10.2348 7 10.5 7ZM4 9H2V7H4V9ZM10.5 10H13.5V13H10.5V10ZM10.5 3H13.5V6H10.5V3Z",fill:"#96F2FF"})),ed={LOGS:"Logs",METRICS:"Metrics",TRACES:"Traces"},ep=[{id:1,icons:{notFocus:()=>logs_grey_default2(),focus:()=>logs_blue_default2()},title:ed.LOGS,type:"logs",tapped:!0},{id:2,icons:{notFocus:()=>chart_line_grey_default2(),focus:()=>chart_line_blue_default2()},title:ed.METRICS,type:"metrics",tapped:!0},{id:3,icons:{notFocus:()=>tree_structure_grey_default2(),focus:()=>tree_structure_blue_default2()},title:ed.TRACES,type:"traces",tapped:!0}],eu=l.zo.div`
  padding: 16px 24px;
  display: flex;
  border-radius: 12px;
  gap: 8px;
  border: ${({theme:e})=>`solid 1px ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  align-items: center;
  justify-content: space-between;
  width: 430px;
`,eC=l.zo.div`
  display: flex;
  align-items: center;
  gap: 8px;
`,em=l.zo.div`
  gap: 8px;
  display: flex;
  flex-direction: column;
  width: 100%;
`,eg={backgroundColor:"#fff",padding:4,borderRadius:10},eh=l.zo.div`
  padding: 4px;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 100%;
  opacity: ${({tapped:e})=>e?1:.4};
`,ef=l.zo.div`
  display: flex;
  gap: 8px;
`,ex=l.ZP.div`
  display: flex;
  padding: 16px;
  border-radius: 12px;
  border: ${({theme:e})=>`solid 1px ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  align-items: center;
  width: 272px;
  gap: 8px;
`,eE=l.ZP.div`
  gap: 4px;
  display: flex;
  flex-direction: column;
`,ew=l.ZP.div`
  padding: 4px;
  background-color: #fff;
  border-radius: 8px;
  display: flex;
  justify-content: center;
  align-items: center;
`,ey=(0,n.memo)(({data:e,isConnectable:t})=>{let r=e?.languages?.[0]?.language||"default",l=E[r];return n.createElement(ex,null,n.createElement(ew,null,n.createElement("img",{src:l,alt:"",width:32,height:32})),n.createElement(eE,null,n.createElement(Text,{color:"#8b92a5"},e.namespace),n.createElement(Text,{size:18,weight:600},e?.name)),n.createElement(i.HH,{type:"source",position:i.Ly.Right,id:"a",isConnectable:t,style:{visibility:"hidden"}}))}),eb=l.ZP.div`
  display: inline-flex;
  height: 20px;
  padding: 4px 8px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 32px;
  border: ${({theme:e})=>`solid 1px ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
`,e_=l.ZP.div`
  text-align: center;
  display: flex;
  align-items: center;
  justify-content: center;
`,ev=(0,n.memo)(({isConnectable:e,data:t})=>n.createElement(eb,null,n.createElement(i.HH,{type:"target",position:i.Ly.Left,id:"b",isConnectable:e,style:{visibility:"hidden"}}),n.createElement(e_,null,n.createElement(Text,{color:w.colors.light_grey},t.metrics?.data_transfer)),n.createElement(i.HH,{type:"source",position:i.Ly.Right,id:"a",isConnectable:e,style:{visibility:"hidden"}}))),ek=l.ZP.div`
  width: 100%;
  height: 100%;
`,eL=l.ZP.div`
  button {
    display: flex;
    padding: 8px;
    align-items: center;
    gap: 10px;
    border-radius: 8px;
    border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
    background: #0e1c28 !important;
    margin-bottom: 8px;
  }

  .react-flow__controls button path {
    fill: #fff;
  }
`,e$=l.ZP.div`
  position: absolute;
  z-index: 999;
  top: 15px;
  left: 60px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  background-color: ${w.colors.dark};
  padding: 10px;
  border-radius: 8px;
  border: 1px solid ${w.colors.blue_grey};
  button {
    display: flex;
    padding: 8px;
    align-items: center;
    gap: 10px;
    border-radius: 8px;
    border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
    background: #0e1c28 !important;
    margin-bottom: 8px;
  }

  .react-flow__controls button path {
    fill: #fff;
  }
`,eH=l.ZP.div`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14px;
`,eZ=l.ZP.span`
  background-color: ${({color:e})=>e};
  width: 10px;
  height: 10px;
  border-radius: 8px;
  margin-right: 6px;
`,eV=l.ZP.div`
  display: flex;
  gap: 10px;
  cursor: pointer;
`;function DataFlowControlPanel(){let[e,t]=(0,n.useState)(!0);(0,n.useEffect)(()=>{setTimeout(()=>{t(!1)},7e3)},[]);let r=[{name:"Traces",color:w.colors.traces},{name:"Logs",color:w.colors.logs},{name:"Metrics",color:w.colors.metrics}];return n.createElement(n.Fragment,null,n.createElement(e$,null,n.createElement(eV,{onClick:()=>t(!e)},n.createElement(Text,{size:14,weight:600},"Supported Signals"),n.createElement(expand_arrow_default,null)),e&&n.createElement(eH,null,r.map(e=>n.createElement("div",{key:e.name,style:{display:"flex",alignItems:"center",justifyContent:"center"}},n.createElement(eZ,{color:e.color}),n.createElement(Text,{size:12,weight:500},e.name))))),n.createElement(eL,null,n.createElement(o.Z,{position:"top-left",showInteractive:!1})))}var eF=w.colors.data_flow_bg,eM={custom:({isConnectable:e})=>n.createElement(el,null,n.createElement(ei,null,n.createElement(ea,null,n.createElement("img",{src:"https://d1n7d4xz7fr8b4.cloudfront.net/logo.png",alt:"logo",style:{borderRadius:"50%",width:64,height:64}}))),n.createElement(i.HH,{type:"target",position:i.Ly.Left,style:{visibility:"hidden"}}),n.createElement(i.HH,{type:"source",position:i.Ly.Right,id:"a",isConnectable:e,style:{visibility:"hidden"}})),namespace:es,destination:function({data:e,isConnectable:t}){return n.createElement(eu,null,n.createElement(eC,null,n.createElement("img",{src:e?.destination_type?.image_url,width:40,height:40,style:eg,alt:""}),n.createElement(em,null,n.createElement(Text,{color:"#8b92a5"},e?.name),n.createElement(Text,{size:18,weight:600},e?.destination_type?.display_name))),n.createElement(ef,null,ep.map(t=>n.createElement(eh,{key:t?.id,tapped:e?.signals?.[t?.type]?"true":void 0,style:{border:`solid 2px ${w.colors[t.type.toLowerCase()]}`}},e?.signals?.[t?.type]?t.icons.focus():t.icons.notFocus()))),n.createElement(i.HH,{type:"target",position:i.Ly.Left,id:"a",isConnectable:t,style:{visibility:"hidden"}}))},action:er,source:ey,metric:ev};function DataFlow({nodes:e,edges:t,...r}){let{fitView:l}=(0,i._K)();return(0,n.useEffect)(()=>{setTimeout(()=>{l()},100)},[l,e,t]),n.createElement(ek,null,n.createElement(i.x$,{nodes:e,edges:t,nodeTypes:eM,nodesDraggable:!1,nodeOrigin:[.4,.4],...r},n.createElement(DataFlowControlPanel,null),n.createElement(a.A,{gap:12,size:1,style:{backgroundColor:eF}})))}function KeyvalDataFlow(e){return n.createElement(i.tV,null,n.createElement(DataFlow,{...e}))}var eS=l.zo.div`
  padding: 10px;
  border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
  border-radius: 8px;
  width: fit-content;
  width: 344px;
  display: flex;
  flex-direction: column;
  gap: 8px;
`,eP=l.zo.div`
  width: 100%;
  display: flex;
  justify-content: flex-end;
  :hover {
    background: ${({theme:e})=>e.colors.error};
    p {
      color: #fff !important;
    }
  }
`,eA=l.zo.button`
  padding: 8px 12px;
  border-radius: 4px;
  background: transparent;
  border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
  cursor: pointer !important;
`;function DangerZone({title:e,subTitle:t,btnText:r,onClick:l}){return n.createElement(n.Fragment,null,n.createElement(eS,null,n.createElement(Text,{size:14,weight:600},e),n.createElement(Text,{size:12},t),n.createElement(eP,null,n.createElement(eA,{onClick:l},n.createElement(Text,{weight:500,size:14,color:w.colors.error},r)))))}var portal_modal_default=({children:e,wrapperId:t})=>{let[r,l]=(0,n.useState)(null);(0,n.useLayoutEffect)(()=>{let e=document.getElementById(t),r=!1;return e||(e=createWrapperAndAppendToBody(t),r=!0),l(e),()=>{r&&e.parentNode&&e.parentNode.removeChild(e)}},[t]);let createWrapperAndAppendToBody=e=>{let t=document.createElement("div");return t.setAttribute("id",e),document.body.appendChild(t),t};return r?(0,c.createPortal)(e,r):null},eI=l.F4`
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
`;l.ZP.div`
  padding: 40px;
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 20px;
`,l.ZP.button`
  display: block;
  padding: 10px 30px;
  border-radius: 3px;
  color: ${({theme:e})=>e.colors.btnText};
  border: 1px solid ${({theme:e})=>e.colors.main};
  background-color: ${({theme:e})=>e.colors.main};
  font-family: 'Robot', sans-serif;
  font-weight: 500;
  transition: 0.3s ease all;

  &:hover {
    background-color: ${({theme:e})=>e.colors.shadowMain};
  }
`,l.ZP.button`
  display: block;
  padding: 10px 30px;
  border-radius: 3px;
  color: ${({theme:e})=>e.colors.main};
  border: 1px solid ${({theme:e})=>e.colors.main};
  background-color: transparent;
  font-family: 'Robot', sans-serif;
  font-weight: 500;
  transition: 0.3s ease all;

  &:hover {
    background-color: ${({theme:e})=>e.colors.shadowMain};
    color: ${({theme:e})=>e.colors.btnText};
  }
`;var eT=l.ZP.div`
  width: 100vw;
  height: 100vh;
  z-index: 9999;
  position: fixed;
  top: 0;
  left: 0;
  background-color: ${e=>(e.showOverlay,"rgba(255, 255, 255, 0.1)")};
  display: flex;
  align-items: center;
  justify-content: ${e=>e.positionX?e.positionX:"center"};
  align-items: ${e=>e.positionY?e.positionY:"center"};
  padding: 40px;

  @media (prefers-reduced-motion: no-preference) {
    animation-name: ${eI};
    animation-fill-mode: backwards;
  }
`,eB=l.ZP.div`
  min-width: 500px;
  min-height: 50px;
  /* background-color: #ffffff; */
  position: relative;
  /* border-radius: 8px; */
  border-radius: 12px;
  border: 0.95px solid var(--dark-mode-dark-3, #203548);
  background: var(--dark-mode-dark-2, #0e1c28);

  padding: ${e=>e.padding?e.padding:"20px"};
`,eR=l.ZP.header`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 20px;
`,ez=l.ZP.div`
  position: absolute;
  top: 20px;
  right: 15px;
  border: none;
  background-color: transparent;
  transition: 0.3s ease all;
  border-radius: 3px;
  color: '#d1345b';
  cursor: pointer;

  svg {
    width: 24px;
    height: 24px;
    fill: #fff;
  }
`;l.ZP.button`
  background-color: #ededed8b;
  border: 1px solid #d4d2d2;
  width: 100%;
  height: 36px;
  border-radius: 8px;
  cursor: pointer;

  &:hover {
    background-color: #ededed;
  }
`;var ej=l.ZP.div`
  display: flex;
  width: 100%;
  flex-direction: column;
  align-items: center;
  color: ${({theme:e})=>e.text};
`,eG=l.ZP.footer`
  width: 100%;
  display: flex;
  gap: 1rem;
  align-items: center;
  justify-content: flex-end;
  margin-top: 20px;
  padding-top: 20px;
`,close_modal_default=e=>n.createElement("svg",{xmlns:"http://www.w3.org/2000/svg",width:16,height:16,viewBox:"0 0 16 16",fill:"none",...e},n.createElement("rect",{width:16,height:16,rx:2,fill:"#0E1C28"}),n.createElement("path",{d:"M12.8538 12.146C12.9002 12.1925 12.9371 12.2476 12.9622 12.3083C12.9874 12.369 13.0003 12.4341 13.0003 12.4998C13.0003 12.5655 12.9874 12.6305 12.9622 12.6912C12.9371 12.7519 12.9002 12.8071 12.8538 12.8535C12.8073 12.9 12.7522 12.9368 12.6915 12.962C12.6308 12.9871 12.5657 13.0001 12.5 13.0001C12.4343 13.0001 12.3693 12.9871 12.3086 12.962C12.2479 12.9368 12.1927 12.9 12.1463 12.8535L8.00003 8.70666L3.85378 12.8535C3.75996 12.9474 3.63272 13.0001 3.50003 13.0001C3.36735 13.0001 3.2401 12.9474 3.14628 12.8535C3.05246 12.7597 2.99976 12.6325 2.99976 12.4998C2.99976 12.3671 3.05246 12.2399 3.14628 12.146L7.29316 7.99979L3.14628 3.85354C3.05246 3.75972 2.99976 3.63247 2.99976 3.49979C2.99976 3.36711 3.05246 3.23986 3.14628 3.14604C3.2401 3.05222 3.36735 2.99951 3.50003 2.99951C3.63272 2.99951 3.75996 3.05222 3.85378 3.14604L8.00003 7.29291L12.1463 3.14604C12.2401 3.05222 12.3674 2.99951 12.5 2.99951C12.6327 2.99951 12.76 3.05222 12.8538 3.14604C12.9476 3.23986 13.0003 3.36711 13.0003 3.49979C13.0003 3.63247 12.9476 3.75972 12.8538 3.85354L8.70691 7.99979L12.8538 12.146Z",fill:"white"}));function Modal({children:e,closeModal:t,config:r}){let l=(0,n.useRef)(null),i=(0,n.useCallback)(e=>{"Escape"===e.key&&t()},[]);return useOnClickOutside(l,()=>t()),(0,n.useEffect)(()=>(document.addEventListener("keydown",i),()=>{document.removeEventListener("keydown",i)}),[i]),n.createElement(n.Fragment,null,n.createElement(portal_modal_default,{wrapperId:"modal-portal"},n.createElement(eT,{showOverlay:r.showOverlay,positionX:r.positionX,positionY:r.positionY,style:{animationDuration:"400ms",animationDelay:"0"}},n.createElement(eB,{padding:r.padding,ref:l},r.showHeader&&n.createElement(eR,null,n.createElement(Text,{size:24,weight:700},r.title)),n.createElement(ez,{onClick:t},n.createElement(close_modal_default,null)),n.createElement(ej,null,e),r?.footer&&n.createElement(eG,{style:{...r.footer.style}},r.footer.link&&n.createElement(Link,{onClick:r.footer.link.onClick,value:r.footer.link.text}),r.footer.secondaryBtnText&&n.createElement(Button,{variant:"secondary",onClick:r.footer.secondaryBtnAction},n.createElement(Text,{size:16,weight:700},r.footer.secondaryBtnText)),n.createElement(Button,{disabled:r.footer.isDisabled,onClick:r.footer.primaryBtnAction},n.createElement(Text,{size:16,weight:700,color:w.text.dark_button},r.footer.primaryBtnText)))))))}function StyledComponentsRegistry({children:e}){let[t]=(0,n.useState)(()=>new l.qH);return((0,s.useServerInsertedHTML)(()=>{let e=t.getStyleElement();return t.instance.clearTag(),n.createElement(n.Fragment,null,e)}),"undefined"!=typeof window)?n.createElement(n.Fragment,null,e):n.createElement(l.LC,{sheet:t.instance},e)}var ThemeProviderWrapper=({children:e})=>n.createElement(l.f6,{theme:w},n.createElement(StyledComponentsRegistry,null,e)),eD=l.ZP.div`
  display: flex;
`,eO=l.ZP.div`
  display: flex;
  align-items: center;
`,eU=l.ZP.div`
  opacity: ${({disabled:e})=>e?"0.4":"1"};
`,eN=(0,l.ZP)(eU)`
  margin: 0 8px;
`,eW=l.ZP.div`
  width: 54px;
  height: 1px;
  background-color: #8b92a5;
  margin-top: 2px;
  margin-right: 8px;
`,checked_default=e=>n.createElement("svg",{width:20,height:14,viewBox:"0 0 20 14",fill:"none",xmlns:"http://www.w3.org/2000/svg",...e},n.createElement("path",{d:"M19.1767 1.88786L7.48781 13.675C7.386 13.778 7.26503 13.8597 7.13183 13.9155C6.99863 13.9713 6.85583 14 6.7116 14C6.56737 14 6.42456 13.9713 6.29136 13.9155C6.15816 13.8597 6.03719 13.778 5.93539 13.675L0.821518 8.51812C0.719584 8.41532 0.638726 8.29329 0.58356 8.15899C0.528394 8.02469 0.5 7.88074 0.5 7.73538C0.5 7.59001 0.528394 7.44606 0.58356 7.31176C0.638726 7.17746 0.719584 7.05543 0.821518 6.95264C0.923451 6.84985 1.04446 6.76831 1.17765 6.71268C1.31083 6.65705 1.45357 6.62842 1.59773 6.62842C1.74189 6.62842 1.88463 6.65705 2.01781 6.71268C2.151 6.76831 2.27201 6.84985 2.37394 6.95264L6.71251 11.3277L17.6261 0.324221C17.8319 0.116626 18.1111 0 18.4023 0C18.6934 0 18.9726 0.116626 19.1785 0.324221C19.3843 0.531816 19.5 0.813376 19.5 1.10696C19.5 1.40054 19.3843 1.6821 19.1785 1.8897L19.1767 1.88786Z",fill:"white"}));function StepItem({title:e,index:t,status:r,isLast:l}){return n.createElement(eO,null,n.createElement(FloatBox,null,"done"===r?n.createElement(checked_default,null):n.createElement(eU,{disabled:"active"!==r},n.createElement(Text,{weight:700},t))),n.createElement(eN,{disabled:"active"!==r},n.createElement(Text,{weight:600},e)),!l&&n.createElement(eW,null))}function Steps({data:e}){return n.createElement(eD,null,e?.map(({title:t,status:r},l)=>n.createElement(StepItem,{key:`${l}_${t}`,title:t,status:r,index:l+1,isLast:l+1===e.length})))}l.ZP.div`
  width: 100%;
  display: flex;
  align-items: center;
  gap: 23px;
  margin: ${({margin:e})=>e};
`,l.ZP.div`
  width: 100%;
  border-top: 1px solid #8b92a5;
`,l.ZP.div`
  padding: 16px;
  display: flex;
  justify-content: flex-start !important;
  border: 1px solid ${({theme:e})=>e.colors.dark_blue};
  border-radius: 12px;
`,l.ZP.div`
  line-height: 1.6;
  code {
    background-color: ${({theme:e})=>e.colors.dark_blue};
    padding: 2px 4px;
    border-radius: 6px;
  }
`,l.ZP.div`
  display: inline-flex;
  justify-content: space-between;
  border-radius: 10px;
  margin: auto;
  overflow: hidden;
  position: relative;
`,l.ZP.div`
  color: ${({theme:e})=>e.colors.white};
  padding: 8px 12px;
  position: relative;
  text-align: center;
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  z-index: 1;
  border: ${({theme:e})=>`1px solid  ${e.colors.secondary}`};
  background-color: ${({theme:e})=>e.colors.dark};
  filter: brightness(50%);
  &.active {
    filter: brightness(100%);
  }
  &:first-child {
    border-top-left-radius: 10px;
    border-bottom-left-radius: 10px;
    padding-left: 16px;
  }
  &:last-child {
    border-top-right-radius: 10px;
    border-bottom-right-radius: 10px;
    padding-right: 16px;
  }
  label {
    font-family: ${({theme:e})=>e.font_family.primary};
  }
`,l.ZP.input`
  opacity: 0;
  margin: 0;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  position: absolute;
  width: 100%;
  cursor: pointer;
  height: 100%;
`,l.ZP.div`
  width: 100%;
`,l.ZP.div`
  display: flex;
  width: 110%;
  flex-wrap: wrap;
`,l.ZP.div`
  cursor: pointer;
  padding: 2px 8px;
  margin: 3px;
  border-radius: 5px;
  background: ${w.colors.light_grey};
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 5px;
  min-height: 30px;
`,l.ZP.div`
  display: flex;
  gap: 10px;
  width: 100%;
  height: 37px;
`,(0,l.ZP)(Input)`
  width: 100%;
`,(0,l.ZP)(Button)`
  margin-left: 10px;
`,l.ZP.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
`;var eK=l.ZP.div`
  width: 100%;
`,eQ=l.ZP.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
`,eq=l.ZP.table`
  border: solid 1px ${w.text.grey};
  text-align: center;
  border-spacing: 0;
  border-radius: 8px;
  width: 100%;
`,eY=l.ZP.th`
  padding: 4px;
`,eX=l.ZP.td`
  padding: 8px 0;

  border-top: solid 1px ${w.text.grey};
  border-right: ${({right:e})=>e?`solid 1px ${w.text.grey}`:"none"};
  border-left: ${({left:e})=>e?`solid 1px ${w.text.grey}`:"none"};
`,eJ=l.ZP.input`
  background: transparent;
  border: none;
  width: 94%;

  outline: none;
  color: ${w.text.white};
`,e1=l.ZP.td`
  text-align: center;
  border-top: solid 1px ${w.text.grey};
  padding: 4px;
  cursor: pointer;
`,KeyValueTable=({keyValues:e,setKeyValues:t,title:r,titleKey:l,titleValue:i,titleButton:a,tooltip:o,required:c})=>{let[s,d]=(0,n.useState)(1),deleteRow=r=>{let n=e.filter(e=>e.id!==r);t(n)},updateKey=(r,n)=>{let l=e.map(e=>e.id===r?{...e,key:n}:e);t(l)},updateValue=(r,n)=>{let l=e.map(e=>e.id===r?{...e,value:n}:e);t(l)};return n.createElement(eK,null,r&&n.createElement(eQ,null,n.createElement(Tooltip,{text:o||""},n.createElement("div",{style:{display:"flex",gap:4}},n.createElement(Text,{size:14,weight:600},r),c&&n.createElement(Text,{size:14,weight:600},"*")))),n.createElement(eq,null,n.createElement("thead",null,n.createElement("tr",null,n.createElement(eY,null,n.createElement(Text,{color:w.text.grey,size:14,style:{display:"flex"},weight:300},l||"Key")),n.createElement(eY,null,n.createElement(Text,{color:w.text.grey,size:14,style:{display:"flex"},weight:300},i||"Value")))),n.createElement("tbody",null,e.map(e=>n.createElement("tr",{key:e.id},n.createElement(eX,{right:!0},n.createElement(eJ,{type:"text",value:e.key,onChange:t=>updateKey(e.id,t.target.value)})),n.createElement(eX,null,n.createElement(eJ,{type:"text",value:e.value,onChange:t=>updateValue(e.id,t.target.value)})),n.createElement(eX,{style:{cursor:"pointer"},left:!0,onClick:()=>deleteRow(e.id)},n.createElement(trash_default,null))))),n.createElement("tfoot",null,n.createElement("tr",null,n.createElement(e1,{onClick:()=>{t([...e,{id:s,key:"",value:""}]),d(s+1)},colSpan:3},n.createElement(Text,{weight:400,size:14,color:w.colors.torquiz_light},a||"Add Row"))))))},e2=l.ZP.textarea`
  width: 100%;
  padding: 8px 12px;
  border-radius: 8px;
  box-sizing: border-box;
  resize: vertical;

  outline: none;
  color: ${({theme:e})=>e.text.white};
  font-family: ${w.font_family.primary};
  background-color: ${({theme:e})=>e.colors.light_dark};
  border: ${({theme:e,active:t})=>`1px solid ${t?e.text.grey:e.colors.blue_grey}`};
  &:hover {
    border: ${({theme:e})=>`solid 1px ${e.text.grey}`};
  }
`,e5=l.ZP.div`
  margin-bottom: 8px;
  text-align: start;
`,TextArea=({placeholder:e,value:t,onChange:r,rows:l=4,cols:i=50,tooltip:a,label:o,required:c,...s})=>n.createElement(n.Fragment,null,o&&n.createElement(e5,null,n.createElement(Tooltip,{text:a||""},n.createElement("div",{style:{display:"flex",gap:4}},n.createElement(Text,{size:14,weight:600},o),c&&n.createElement(Text,{size:14,weight:600},"*")))),n.createElement(e2,{placeholder:e,value:t,onChange:r,rows:l,cols:i,active:!!t,...s})),e0=l.ZP.div`
  width: 100%;
`,e3=l.ZP.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
`,e4=l.ZP.table`
  border: solid 1px ${w.text.grey};
  text-align: center;
  border-spacing: 0;
  border-radius: 8px;
  width: 100%;
`;l.ZP.th`
  padding: 4px;
`;var e6=l.ZP.td`
  padding: 8px 0;

  border-bottom: solid 1px ${w.text.grey};
  border-right: ${({right:e})=>e?`solid 1px ${w.text.grey}`:"none"};
  border-left: ${({left:e})=>e?`solid 1px ${w.text.grey}`:"none"};
`,e9=l.ZP.input`
  background: transparent;
  border: none;
  width: 94%;

  outline: none;
  color: ${w.text.white};
`,e7=l.ZP.td`
  text-align: center;
  padding: 4px;
  cursor: pointer;
`,MultiInputTable=({values:e,title:t,tooltip:r,onValuesChange:l,required:i,placeholder:a})=>{let deleteRow=t=>{let r=e.filter((e,r)=>r!==t);l(r)},updateValue=(t,r)=>{let n=e.map((e,n)=>n===t?r:e);l(n)};return n.createElement(e0,null,t&&n.createElement(e3,null,n.createElement(Tooltip,{text:r||""},n.createElement("div",{style:{display:"flex",gap:4}},n.createElement(Text,{size:14,weight:600},t),i&&n.createElement(Text,{size:14,weight:600},"*")))),n.createElement(e4,null,n.createElement("tbody",null,e.map((e,t)=>n.createElement("tr",{key:t},n.createElement(e6,{right:!0},n.createElement(e9,{type:"text",value:e,onChange:e=>updateValue(t,e.target.value),placeholder:0===t?a:""})),n.createElement(e6,{onClick:()=>deleteRow(t),style:{cursor:"pointer"}},n.createElement(trash_default,null))))),n.createElement("tfoot",null,n.createElement("tr",null,n.createElement(e7,{onClick:()=>{l([...e,""])},colSpan:2},n.createElement(Text,{weight:400,size:14,color:w.colors.torquiz_light},"Add Row"))))))},e8=l.ZP.label`
  cursor: pointer;
  display: flex;
  gap: 4px;
  p {
    color: ${({theme:e})=>e.colors.light_grey};
    &:hover {
      color: ${({theme:e})=>e.colors.white};
    }
  }
`,te=l.ZP.div`
  display: ${e=>e.isOpen?"block":"none"};
  position: absolute;
  right: 0px;
  box-shadow: 0px 8px 16px 0px rgba(0, 0, 0, 0.2);
  z-index: 9999;
  flex-direction: column;
  border-radius: 8px;
  border: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
  background: ${({theme:e})=>e.colors.dark};
  margin-top: 5px;
`,tt=l.ZP.div`
  display: flex;
  padding: 7px 12px;
  gap: 4px;
  border-top: ${({theme:e})=>`1px solid ${e.colors.blue_grey}`};
  align-items: center;
  opacity: ${({disabled:e})=>e?.5:1};
  pointer-events: ${({disabled:e})=>e?"none":"auto"};
  cursor: pointer;
  p {
    cursor: pointer !important;
  }

  &:hover {
    background: ${({theme:e})=>e.colors.light_dark};
  }
`,ActionItem=({label:e,items:t,subTitle:r})=>{let[l,i]=(0,n.useState)(!1),a=(0,n.useRef)(null);return useOnClickOutside(a,()=>i(!1)),n.createElement("div",{ref:a,style:{position:"relative"}},n.createElement(e8,{onClick:()=>i(!l)},n.createElement(Text,{size:12,weight:600},e),n.createElement(expand_arrow_default,null)),n.createElement(te,{isOpen:l},n.createElement("div",{style:{padding:12,width:120}},n.createElement(Text,{size:12,weight:600},r)),t.map((e,t)=>n.createElement(tt,{key:t,onClick:e.onClick,disabled:!!e.disabled},e.selected?n.createElement(check_default,null):n.createElement("div",{style:{width:10}}),n.createElement(Text,{size:12,weight:600},e.label)))))},ActionsGroup=({actionGroups:e})=>n.createElement(n.Fragment,null,e.map((e,t)=>e.condition&&n.createElement(ActionItem,{key:t,...e}))),tr=l.ZP.div`
  display: flex;
  justify-content: center;
  padding: 20px;
  gap: 2px;
`,tn=l.ZP.button`
  background-color: ${e=>e.isCurrentPage?w.colors.blue_grey:"transparent"};
  color: ${e=>e.isDisabled?w.text.grey:w.text.white};
  border: none;
  border-radius: 4px;
  padding: 4px 8px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;

  &:disabled {
    cursor: default;
  }

  &:hover {
    background-color: ${w.colors.blue_grey};
  }
`,Pagination=({total:e,itemsPerPage:t,currentPage:r,onPageChange:l})=>{let i=Math.ceil(e/t);return n.createElement(tr,null,n.createElement(tn,{onClick:()=>l(r-1),disabled:1===r,isDisabled:1===r},n.createElement(expand_arrow_default,{style:{transform:"rotate(90deg)"}}),"Previous"),Array(i).fill(0).map((e,t)=>n.createElement(tn,{key:t,onClick:()=>l(t+1),isCurrentPage:r===t+1},t+1)),n.createElement(tn,{onClick:()=>l(r+1),disabled:r===i,isDisabled:r===i},"Next",n.createElement(expand_arrow_default,{style:{transform:"rotate(-90deg)"}})))},tl=l.ZP.table`
  width: 100%;
  background-color: ${w.colors.dark};
  border: 1px solid ${w.colors.blue_grey};
  border-radius: 6px;
  border-collapse: separate;
  border-spacing: 0;
`,ti=l.ZP.tbody``,ta=l.ZP.div`
  margin: 10px 0;
  gap: 8px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
`,to=l.ZP.select`
  padding: 5px;
  border-radius: 4px;
  border: 1px solid ${w.colors.blue_grey};
  background-color: ${w.colors.dark};
  color: ${w.colors.white};
  border-radius: 8px;
  cursor: pointer;
  border: ${({theme:e})=>`1px solid  ${e.colors.blue_grey}`};
`,tc=l.ZP.option`
  background-color: ${w.colors.dark};
  color: ${w.colors.white};
`,Table3=({data:e,renderTableRows:t,renderTableHeader:r,renderEmptyResult:l,currentPage:i,itemsPerPage:a,setCurrentPage:o,setItemsPerPage:c})=>{let s=i*a,d=s-a,p=e.slice(d,s);return n.createElement(n.Fragment,null,n.createElement(ta,null,n.createElement(Text,{size:12,color:w.text.light_grey},"Showing ",d+1," to"," ",Math.min(s,e.length)," of ",e.length," items"),n.createElement(to,{id:"itemsPerPage",value:a,onChange:e=>{c(Number(e.target.value)),o(1)}},n.createElement(tc,{value:10},"10"),n.createElement(tc,{value:25},"25"),n.createElement(tc,{value:50},"50"),n.createElement(tc,{value:100},"100"))),n.createElement(tl,null,r(),n.createElement(ti,null,p.map((e,r)=>t(e,r)))),0===e.length?l():n.createElement(Pagination,{total:e.length,itemsPerPage:a,currentPage:i,onPageChange:e=>{o(e)}}))};l.ZP.div`
  position: relative;
  background-color: ${w.colors.blue_grey};
  border-radius: 8px;
  padding: 4px;

  div {
    color: #f5b175;
  }
  .ͼb {
    color: #64a8fd;
  }
  .ͼm {
    color: ${w.colors.white};
  }
  .ͼd {
    color: #f5b175;
  }
  .ͼc {
    color: #f5b175;
  }
  .cm-gutters {
    display: none;
    border-top-left-radius: 8px;
    border-top-right-radius: 8px;
  }
`,l.ZP.div`
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 10; // Ensure this is higher than the editor's z-index
`,l.ZP.div`
  background-color: ${w.colors.dark};
  z-index: 999;
  border-radius: 4px;
  padding: 4px;
  position: absolute;
  top: 5px;
  right: 5px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: all;
`;var buildFlowNodesAndEdges=(e,t,r)=>{let n=[],l=[],i="center-1",a=t.length,o=248-100*(a%2==1?Math.floor(a/2):a/2-1),c=e.length,s=248-100*(c%2==1?Math.floor(c/2):c/2-1),d=r?.length>0?r?.length*150+600:600;return n.push({id:i,type:"custom",position:{x:d,y:248},data:{label:"Center Node"}}),e.forEach((e,t)=>{let a=!1;e?.conditions&&(a=e.conditions.some(e=>"False"===e.status));let o=`namespace-${t}`;if(n.push({id:o,type:"source",position:{x:100,y:s+100*t},data:e}),e.metrics){let c=`metric-${t}`;n.push({id:c,type:"metric",position:{x:400,y:s+100*t},data:{metrics:e.metrics}}),l.push({id:`e${o}-${c}`,source:c,target:r?.length>0?"action-0":i,animated:!0,style:{stroke:"#96f3ff8e"},data:null}),l.push({id:`e${o}-${i}`,source:o,target:c,animated:!0,style:{stroke:a?"#ff0000":"#96f3ff8e"},data:null})}else l.push({id:`e${o}-${i}`,source:o,target:r?.length>0?"action-0":i,animated:!0,style:{stroke:a?"#ff0000":"#96f3ff8e"},data:null})}),t.forEach((e,t)=>{let r=!1;e?.conditions&&(r=e.conditions.some(e=>"False"===e.status));let a=`destination-${t}`;if(n.push({id:a,type:"destination",position:{x:d+600,y:o+100*t},data:e}),e.metrics){let r=`metric-dest-${t}`;n.push({id:r,type:"metric",position:{x:d+250,y:o+100*t},data:{metrics:e.metrics}}),l.push({id:`e${a}-${r}`,source:i,target:r,animated:!0,style:{stroke:"#96f3ff8e"},data:null}),l.push({id:`e${a}-${r}`,source:r,target:a,animated:!0,style:{stroke:"#96f3ff8e"},data:null})}else l.push({id:`e${i}-${a}`,source:i,target:a,animated:!0,style:{stroke:r?"#ff0000":"#96f3ff8e"},data:null})}),r.forEach((e,t)=>{let a=`action-${t}`,o=`action-${t+1}`;n.push({id:a,type:"action",position:{x:620+125*t,y:250},data:e}),l.push({id:`e${i}-${a}`,source:a,target:t+1===r.length?i:o,animated:!0,style:{stroke:"#96f3ff8e"},data:null})}),{nodes:n,edges:l}},{nodes:ts,edges:td}=buildFlowNodesAndEdges([{name:"adservice",kind:"Deployment",namespace:"default",metrics:{data_transfer:"3.8 KB/s",cpu_usage:"3.8%",memory_usage:"3.8%"},languages:[{container_name:"server",language:"java"}]},{name:"cartservice",kind:"Deployment",namespace:"default",metrics:{data_transfer:"2.3 KB/s",cpu_usage:"3.8%",memory_usage:"3.8%"},languages:[{container_name:"server",language:"dotnet"}]},{name:"checkoutservice",kind:"Deployment",namespace:"default",metrics:{data_transfer:"0 Byte transfered",cpu_usage:"3.8%",memory_usage:"3.8%"},languages:[{container_name:"server",language:"go"}]},{name:"coupon",kind:"Deployment",namespace:"default",metrics:{data_transfer:"3.8 KB/s",cpu_usage:"3.8%",memory_usage:"3.8%"},languages:[{container_name:"coupon",language:"javascript"}]}],[{id:"odigos.io.dest.elasticsearch-6qklw",name:"Elasticsearch",type:"elasticsearch",metrics:{data_transfer:"3.8 KB/s",cpu_usage:"3.8%",memory_usage:"3.8%"},signals:{traces:!0,metrics:!1,logs:!0},fields:{ELASTICSEARCH_CA_PEM:"-----BEGIN CERTIFICATE-----\nMIIDIjCCAgqgAwIBAgIRANR/chGx5YexmqgwbVphZR8wDQYJKoZIhvcNAQELBQAw\nGzEZMBcGA1UEAxMQZWxhc3RpY3NlYXJjaC1jYTAeFw0yNDAzMDYxMjUwNTFaFw0y\nNTAzMDYxMjUwNTFaMBsxGTAXBgNVBAMTEGVsYXN0aWNzZWFyY2gtY2EwggEiMA0G\nCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQClNK8WB2C2aRC1xPkT9Vx3t2I8D8vE\nz4/XKi5djhqZx56VclUmnTGwwJSB6t+9eODVGM8HUBeZTw5r5VU3wz5KO34LfX/X\nDgeZf7jRE4JvNti+ufhYeXhX6yWt2y1lisTy89BMZA1/4r6UBamhDZ9zjC7++hNy\n21S+mgul4zrjC1fBfjz8O42jjkamNcq3SoQHn9puWPhsOBOc4SowJMFN6YIRf3Vy\nPvOuG8wP5uCU14dICW7X5M1JqHpcOTW0W7S5JLcVkozrqEQhQ3lc5f4OE0/GYQ5S\np5sesAUyv9Koiipx3gGvip2+E2Rf2nlLNNMYeFDKyRVmbxkOmIy6PVQdAgMBAAGj\nYTBfMA4GA1UdDwEB/wQEAwICpDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUH\nAwIwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUUh7RgBmgPOCGeP3hFqoVC689\nm4kwDQYJKoZIhvcNAQELBQADggEBAJCpewxuRV0s6EftuHI7Q1PJDYImDv54D1GI\n57nJwdhCZbvJ69m5hjtOAb7ZCerzJQKvN4sEcvcWPMJs15nBNXW+9fF0zN5RjBqU\nV8HA22bw8h+00lGUiozgG4DmFkd0GG35qjhPe9PyirOumiaSi2aGMUiWtkOgAFE2\nDKGLleYFdjDbfQjva/ViUJTo6I7b283foWEqkbaa58ju5QOtjpo09GOmyBXeXHoU\nbLnUqBAALo7FDSdKyMjWRLKSo2rc7jRn98jXzPqRaVuYhEGn+77GnkA5d3ea3fHP\nIrj44yKh8na1xqPEHEvryj9LnKL+yCpQILj5L+jIHVopTmQWyLQ=\n-----END CERTIFICATE-----",ELASTICSEARCH_PASSWORD:"Elasticsearch",ELASTICSEARCH_URL:"Elasticsearch",ELASTICSEARCH_USERNAME:"Elasticsearch",ES_LOGS_INDEX:"Elasticsearch",ES_TRACES_INDEX:"Elasticsearch"},destination_type:{type:"elasticsearch",display_name:"Elasticsearch",image_url:"https:/d15jtxgb40qetw.cloudfront.net/elasticsearch.svg",supported_signals:{traces:{supported:!0},metrics:{supported:!1},logs:{supported:!0}}}},{id:"odigos.io.dest.grafanacloudprometheus-2mcbr",name:"Prometheus",type:"grafanacloudprometheus",signals:{traces:!1,metrics:!0,logs:!1},fields:{GRAFANA_CLOUD_PROMETHEUS_PASSWORD:"Prometheus",GRAFANA_CLOUD_PROMETHEUS_RW_ENDPOINT:"Prometheus",GRAFANA_CLOUD_PROMETHEUS_USERNAME:"Prometheus",PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS:'["k8s.container.name","k8s.pod.name","k8s.namespace.name","Prometheus"]',PROMETHEUS_RESOURCE_EXTERNAL_LABELS:'{"dsfd":"fdsfs"}'},destination_type:{type:"grafanacloudprometheus",display_name:"Grafana Cloud Prometheus",image_url:"https:/d15jtxgb40qetw.cloudfront.net/grafana.svg",supported_signals:{traces:{supported:!1},metrics:{supported:!0},logs:{supported:!1}}}},{id:"odigos.io.dest.s3-gk7bn",name:"aws",type:"s3",metrics:{data_transfer:"3.8111 KB/s",cpu_usage:"3.8%",memory_usage:"3.8%"},signals:{traces:!0,metrics:!0,logs:!0},fields:{S3_BUCKET:"aws",S3_MARSHALER:"otlp_proto",S3_PARTITION:"minute",S3_REGION:"aws"},destination_type:{type:"s3",display_name:"AWS S3",image_url:"https:/d15jtxgb40qetw.cloudfront.net/s3.svg",supported_signals:{traces:{supported:!0},metrics:{supported:!0},logs:{supported:!0}}}}],[{id:"aci-f6c9f",type:"AddClusterInfo",spec:{actionName:"Cluster Attributes",notes:"Actions are a way to modify the OpenTelemetry data recorded by Odigos Sources, before it is exported to your Odigos Destinations.",signals:["METRICS","TRACES"],clusterAttributes:[{attributeName:"Attributes",attributeStringValue:"Attributes"}]}},{id:"aci-hfgcb",type:"DeleteAttribute",spec:{actionName:"Link",notes:"Link to docs",signals:["LOGS","METRICS","TRACES"],clusterAttributes:[{attributeName:"sadsad",attributeStringValue:"sadsa"},{attributeName:"asdsa",attributeStringValue:"asdasd"}]}},{id:"aci-r67mp",type:"RenameAttribute",spec:{actionName:"Initialize Initialize",notes:"This is the initialization phase of the cluster.",signals:["LOGS","METRICS","TRACES"],clusterAttributes:[{attributeName:"region",attributeStringValue:"us-east-1"},{attributeName:"instanceType",attributeStringValue:"t2.micro"},{attributeName:"availabilityZones",attributeStringValue:"3"}]}}]),tp=l.ZP.div`
  width: ${({size:e})=>e||24}px;
  height: ${({size:e})=>e||24}px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
`,LogsIcon=({...e})=>n.createElement(tp,{...e},n.createElement(logs_grey_default,null)),LogsFocusIcon=({...e})=>n.createElement(tp,{...e},n.createElement(logs_blue_default,null)),TraceIcon=({...e})=>n.createElement(tp,{...e},n.createElement(tree_structure_grey_default,null)),TraceFocusIcon=({...e})=>n.createElement(tp,{...e},n.createElement(tree_structure_blue_default,null)),MetricsIcon=({...e})=>n.createElement(tp,{...e},n.createElement(chart_line_grey_default,null)),MetricsFocusIcon=({...e})=>n.createElement(tp,{...e},n.createElement(chart_line_blue_default,null)),AddClusterInfoIcon=({...e})=>n.createElement(tp,{...e},n.createElement(cluster_attr_default,null)),DeleteAttributeIcon=({...e})=>n.createElement(tp,{...e},n.createElement(delete_attr_default,null)),RenameAttributeIcon=({...e})=>n.createElement(tp,{...e},n.createElement(rename_attr_default,null)),ErrorSamplerIcon=({...e})=>n.createElement(tp,{...e},n.createElement(error_sampler_default,null)),PiiMaskingIcon=({...e})=>n.createElement(tp,{...e},n.createElement(pii_masking_default,null)),LatencySamplerIcon=({...e})=>n.createElement(tp,{...e},n.createElement(latency_sampler_default,null)),ProbabilisticSamplerIcon=({...e})=>n.createElement(tp,{...e},n.createElement(probabilistic_sampler_default,null)),PlusIcon=({...e})=>n.createElement(tp,{...e},n.createElement(plus_default,null)),BackIcon=({...e})=>n.createElement(tp,{...e},n.createElement(back_default,null)),RightArrowIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(arrow_right_default,null)),ChargeIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(charge_rect_default,null)),ConnectIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(connect_default,null)),WhiteArrowIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(white_arrow_right_default,null)),LinkIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(link_default,null)),GreenCheckIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(green_check_default,null)),RedErrorIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(red_error_default,null)),BlueInfoIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(blue_info_default,null)),BellIcon=({size:e,color:t,...r})=>n.createElement(tp,{size:e,color:t,...r},n.createElement(bell_default,null)),FocusOverviewIcon=({...e})=>n.createElement(tp,{...e},n.createElement(focus_overview_default,null)),UnFocusOverviewIcon=({...e})=>n.createElement(tp,{...e},n.createElement(unfocus_overview_default,null)),FocusSourcesIcon=({...e})=>n.createElement(tp,{...e},n.createElement(sources_focus_default,null)),UnFocusSourcesIcon=({...e})=>n.createElement(tp,{...e},n.createElement(sources_unfocus_default,null)),FocusDestinationsIcon=({...e})=>n.createElement(tp,{...e},n.createElement(destinations_focus_default,null)),UnFocusDestinationsIcon=({...e})=>n.createElement(tp,{...e},n.createElement(destinations_unfocus_default,null)),FocusActionIcon=({...e})=>n.createElement(tp,{...e},n.createElement(transform_focus_default,null)),UnFocusActionIcon=({...e})=>n.createElement(tp,{...e},n.createElement(transform_unfocus_default,null))}}]);