"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[510],{3905:function(e,t,n){n.d(t,{Zo:function(){return c},kt:function(){return d}});var r=n(7294);function a(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function o(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function i(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?o(Object(n),!0).forEach((function(t){a(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):o(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function l(e,t){if(null==e)return{};var n,r,a=function(e,t){if(null==e)return{};var n,r,a={},o=Object.keys(e);for(r=0;r<o.length;r++)n=o[r],t.indexOf(n)>=0||(a[n]=e[n]);return a}(e,t);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);for(r=0;r<o.length;r++)n=o[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(a[n]=e[n])}return a}var s=r.createContext({}),u=function(e){var t=r.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):i(i({},t),e)),n},c=function(e){var t=u(e.components);return r.createElement(s.Provider,{value:t},e.children)},p={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},m=r.forwardRef((function(e,t){var n=e.components,a=e.mdxType,o=e.originalType,s=e.parentName,c=l(e,["components","mdxType","originalType","parentName"]),m=u(n),d=a,f=m["".concat(s,".").concat(d)]||m[d]||p[d]||o;return n?r.createElement(f,i(i({ref:t},c),{},{components:n})):r.createElement(f,i({ref:t},c))}));function d(e,t){var n=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var o=n.length,i=new Array(o);i[0]=m;var l={};for(var s in t)hasOwnProperty.call(t,s)&&(l[s]=t[s]);l.originalType=e,l.mdxType="string"==typeof e?e:a,i[1]=l;for(var u=2;u<o;u++)i[u]=n[u];return r.createElement.apply(null,i)}return r.createElement.apply(null,n)}m.displayName="MDXCreateElement"},3918:function(e,t,n){n.r(t),n.d(t,{frontMatter:function(){return l},contentTitle:function(){return s},metadata:function(){return u},toc:function(){return c},default:function(){return m}});var r=n(7462),a=n(3366),o=(n(7294),n(3905)),i=["components"],l={sidebar_position:7},s="Metrics",u={unversionedId:"metrics",id:"metrics",title:"Metrics",description:"Before Consul Release Controller increases the traffic to your Candidate deployment it first checks the health of your application",source:"@site/docs/metrics.md",sourceDirName:".",slug:"/metrics",permalink:"/consul-release-controller/metrics",editUrl:"https://github.com/nicholasjackson/consul-release-controller/tree/main/docs/templates/shared/docs/metrics.md",tags:[],version:"current",sidebarPosition:7,frontMatter:{sidebar_position:7},sidebar:"tutorialSidebar",previous:{title:"Webhook notification",permalink:"/consul-release-controller/webhooks"}},c=[{value:"Default Queries",id:"default-queries",children:[{value:"Prometheus / Kubernetes",id:"prometheus--kubernetes",children:[{value:"EnvoyRequestSuccess",id:"envoyrequestsuccess",children:[],level:4},{value:"EnvoyRequestDuration",id:"envoyrequestduration",children:[],level:4}],level:3}],level:2},{value:"Custom Queries",id:"custom-queries",children:[{value:"Parameters",id:"parameters",children:[],level:3}],level:2}],p={toc:c};function m(e){var t=e.components,n=(0,a.Z)(e,i);return(0,o.kt)("wrapper",(0,r.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("h1",{id:"metrics"},"Metrics"),(0,o.kt)("p",null,"Before Consul Release Controller increases the traffic to your Candidate deployment it first checks the health of your application\nby looking at the traffic metrics for the application. By default Consul Release Controller provides default queries for each of the\nsupported platforms."),(0,o.kt)("h2",{id:"default-queries"},"Default Queries"),(0,o.kt)("h3",{id:"prometheus--kubernetes"},"Prometheus / Kubernetes"),(0,o.kt)("h4",{id:"envoyrequestsuccess"},"EnvoyRequestSuccess"),(0,o.kt)("p",null,"This query measures the HTTP response codes emitted from Envoy for your application and returns the percentage of requests (0-100)\nthat do not result in a HTTP 5xx response."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-javascript"},'sum(\n    rate(\n    envoy_cluster_upstream_rq{\n      namespace="{{ .Namespace }}",\n      pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)",\n      envoy_cluster_name="local_app",\n      envoy_response_code!~"5.*"\n    }[{{ .Interval }}]\n  )\n)\n/\nsum(\n  rate(\n    envoy_cluster_upstream_rq{\n      namespace="{{ .Namespace }}",\n      envoy_cluster_name="local_app",\n      pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"\n    }[{{ .Interval }}]\n  )\n)\n* 100\n')),(0,o.kt)("h4",{id:"envoyrequestduration"},"EnvoyRequestDuration"),(0,o.kt)("p",null,"This query measures the 99 percentile duration for application requests in milliseconds."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-javascript"},'histogram_quantile(\n  0.99,\n  sum(\n    rate(\n      envoy_cluster_upstream_rq_time_bucket{\n        namespace="{{ .Namespace }}",\n        envoy_cluster_name="local_app",\n        pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"\n      }[{{ .Interval }}]\n    )\n  ) by (le)\n)\n')),(0,o.kt)("h2",{id:"custom-queries"},"Custom Queries"),(0,o.kt)("p",null,"Custom queries can be defined by specifying the optional ",(0,o.kt)("inlineCode",{parentName:"p"},"query")," parameter instead of the ",(0,o.kt)("inlineCode",{parentName:"p"},"preset")," parameter."),(0,o.kt)("p",null,"Queries are specified as Prometheus queries and must return a numeric value that can be evaluated with the ",(0,o.kt)("inlineCode",{parentName:"p"},"min"),", ",(0,o.kt)("inlineCode",{parentName:"p"},"max"),"\ncriteria. To enable generic queries Go templates can be used to inject values such as the ",(0,o.kt)("inlineCode",{parentName:"p"},"Name")," of the deployment\nor the ",(0,o.kt)("inlineCode",{parentName:"p"},"Namespace")," where the deployment is running."),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},'monitor:\n  pluginName: "prometheus"\n  config:\n    address: "http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090"\n    queries:\n      - name: "mycustom"\n        min: 20\n        max: 200\n        query: |\n          histogram_quantile(\n            0.99,\n            sum(\n              rate(\n                envoy_cluster_upstream_rq_time_bucket{\n                  namespace="{{ .Namespace }}",\n                  envoy_cluster_name="local_app",\n                  pod=~"{{ .Name }}-[0-9a-zA-Z]+(-[0-9a-zA-Z]+)"\n                }[{{ .Interval }}]\n              )\n            ) by (le)\n          )\n')),(0,o.kt)("h3",{id:"parameters"},"Parameters"),(0,o.kt)("table",null,(0,o.kt)("thead",{parentName:"table"},(0,o.kt)("tr",{parentName:"thead"},(0,o.kt)("th",{parentName:"tr",align:null},"Parameter"),(0,o.kt)("th",{parentName:"tr",align:null},"Type"),(0,o.kt)("th",{parentName:"tr",align:null},"Description"))),(0,o.kt)("tbody",{parentName:"table"},(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},"Name"),(0,o.kt)("td",{parentName:"tr",align:null},"string"),(0,o.kt)("td",{parentName:"tr",align:null},"Name of the candidate deployment")),(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},"Namespace"),(0,o.kt)("td",{parentName:"tr",align:null},"string"),(0,o.kt)("td",{parentName:"tr",align:null},"Namespace where the candidate is running")),(0,o.kt)("tr",{parentName:"tbody"},(0,o.kt)("td",{parentName:"tr",align:null},"Interval"),(0,o.kt)("td",{parentName:"tr",align:null},"duration"),(0,o.kt)("td",{parentName:"tr",align:null},"Interval from the Strategy config, specified as a prometheus duration (30s, etc)")))))}m.isMDXComponent=!0}}]);