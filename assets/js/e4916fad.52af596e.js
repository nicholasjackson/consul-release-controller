"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[356],{3905:function(e,n,t){t.d(n,{Zo:function(){return c},kt:function(){return d}});var l=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function r(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var l=Object.getOwnPropertySymbols(e);n&&(l=l.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,l)}return t}function a(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?r(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):r(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function s(e,n){if(null==e)return{};var t,l,o=function(e,n){if(null==e)return{};var t,l,o={},r=Object.keys(e);for(l=0;l<r.length;l++)t=r[l],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);for(l=0;l<r.length;l++)t=r[l],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var u=l.createContext({}),i=function(e){var n=l.useContext(u),t=n;return e&&(t="function"==typeof e?e(n):a(a({},n),e)),t},c=function(e){var n=i(e.components);return l.createElement(u.Provider,{value:n},e.children)},p={inlineCode:"code",wrapper:function(e){var n=e.children;return l.createElement(l.Fragment,{},n)}},m=l.forwardRef((function(e,n){var t=e.components,o=e.mdxType,r=e.originalType,u=e.parentName,c=s(e,["components","mdxType","originalType","parentName"]),m=i(t),d=o,f=m["".concat(u,".").concat(d)]||m[d]||p[d]||r;return t?l.createElement(f,a(a({ref:n},c),{},{components:t})):l.createElement(f,a({ref:n},c))}));function d(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var r=t.length,a=new Array(r);a[0]=m;var s={};for(var u in n)hasOwnProperty.call(n,u)&&(s[u]=n[u]);s.originalType=e,s.mdxType="string"==typeof e?e:o,a[1]=s;for(var i=2;i<r;i++)a[i]=t[i];return l.createElement.apply(null,a)}return l.createElement.apply(null,t)}m.displayName="MDXCreateElement"},9830:function(e,n,t){t.r(n),t.d(n,{frontMatter:function(){return s},contentTitle:function(){return u},metadata:function(){return i},toc:function(){return c},default:function(){return h}});var l=t(7462),o=t(3366),r=(t(7294),t(3905)),a=["components"],s={sidebar_position:2},u="Helm values",i={unversionedId:"helm_values",id:"helm_values",title:"Helm values",description:"Depending on the security configuration of your Consul server, you need to configure the Helm values file accordingly, Consul Release Controller",source:"@site/docs/helm_values.md",sourceDirName:".",slug:"/helm_values",permalink:"/consul-release-controller/helm_values",editUrl:"https://github.com/nicholasjackson/consul-release-controller/tree/main/docs/templates/shared/docs/helm_values.md",tags:[],version:"current",sidebarPosition:2,frontMatter:{sidebar_position:2},sidebar:"tutorialSidebar",previous:{title:"Installing the example application",permalink:"/consul-release-controller/example_app"},next:{title:"Prerequisites",permalink:"/consul-release-controller/prerequisites"}},c=[{value:"ACL support",id:"acl-support",children:[],level:4},{value:"TLS with auto encrypt",id:"tls-with-auto-encrypt",children:[],level:4}],p=function(e){return function(n){return console.warn("Component "+e+" was not imported, exported, or provided by MDXProvider as global scope"),(0,r.kt)("div",n)}},m=p("Tabs"),d=p("TabItem"),f={toc:c};function h(e){var n=e.components,t=(0,o.Z)(e,a);return(0,r.kt)("wrapper",(0,l.Z)({},f,t,{components:n,mdxType:"MDXLayout"}),(0,r.kt)("h1",{id:"helm-values"},"Helm values"),(0,r.kt)("p",null,"Depending on the security configuration of your Consul server, you need to configure the Helm values file accordingly, Consul Release Controller\nneeds to communicate to a Consul agent in order to set the various Service Mesh Configuration."),(0,r.kt)("p",null,"Consul release controller can be configured using the same environment variables used for Consul Agent."),(0,r.kt)("p",null,(0,r.kt)("a",{parentName:"p",href:"https://www.consul.io/commands#environment-variables"},"https://www.consul.io/commands#environment-variables")),(0,r.kt)("p",null,"Depending on your security configuration you will need to configure the Helm chart to set these values and to obtain the\nassociated certificates or tokens."),(0,r.kt)(m,{groupId:"helm_values",mdxType:"Tabs"},(0,r.kt)(d,{value:"insecure",label:"Insecure",mdxType:"TabItem"},(0,r.kt)("p",null,"If you are using a Consul setup that does not have ACLs configured or TLS security enabled, the default values\nassume that the Consul Agent for the server is installed as a ",(0,r.kt)("inlineCode",{parentName:"p"},"DaemonSet"),", and is available using the Kubernetes\n",(0,r.kt)("inlineCode",{parentName:"p"},"status.hostIP")),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},"controller:\n\n  container_config:\n    - name: HOST_IP\n      valueFrom:\n        fieldRef:\n          fieldPath: status.hostIP\n    - name: CONSUL_HTTP_ADDR\n      value: http://$(HOST_IP):8501\n")),(0,r.kt)("p",null,"If your cluster is not setup in this way then you will need to change the environment variable ",(0,r.kt)("inlineCode",{parentName:"p"},"CONSUL_HTTP_ADDR")," to the address\nof a Consul Agent that can be used by the cluster. Consul release controller will work if pointed directly at the Consul server,\nhowever this is not recommend. "),(0,r.kt)("p",null,"While fine for local development environments, we do not recommend using Consul without ACL's and TLS.")),(0,r.kt)(d,{value:"secure",label:"ACLS and TLS",default:"true",mdxType:"TabItem"},(0,r.kt)("h4",{id:"acl-support"},"ACL support"),(0,r.kt)("p",null,"If you have setup Consul using the official Helm chart and have enabled ACL and the Kubernetes controller using the following Helm values:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml",metastring:'title="Official Consul Helm Values"',title:'"Official',Consul:!0,Helm:!0,'Values"':!0},"global:\n  acls:\n    manageSystemACLs: true\ncontroller:\n  enabled: true\n")),(0,r.kt)("p",null,"You can simply enable ACL support in the Helm values:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml",metastring:'title="Consul Release Controller Helm Values"',title:'"Consul',Release:!0,Controller:!0,Helm:!0,'Values"':!0},"acls:\n  enabled: true\n")),(0,r.kt)("p",null,"The Helm chart will auto configure the controller to set the environment variable ",(0,r.kt)("inlineCode",{parentName:"p"},"CONSUL_HTTP_TOKEN")," to use the same ACL token stored in the secret\n",(0,r.kt)("inlineCode",{parentName:"p"},"consul-controller-acl-token")," as used by the Kubernetes controller.  Consul Release Controller and the Kubernetes Controller both require the same\npermissions to read and write Consul config.  Should you wish to use a different token or if you are not using the Kubernetes controller, then you can\noverride the Helm values ",(0,r.kt)("inlineCode",{parentName:"p"},"acls.env.CONSUL_HTTP_TOKEN")," to set the name of the Kubernetes secret where your custom ACL token is stored."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml",metastring:'title="Consul Release Controller Helm Values"',title:'"Consul',Release:!0,Controller:!0,Helm:!0,'Values"':!0},"acls:\n  enabled: false\n  env:\n  - name: CONSUL_HTTP_TOKEN\n    valueFrom: \n      secretKeyRef:\n        name: consul-controller-acl-token\n        key: token \n")),(0,r.kt)("h4",{id:"tls-with-auto-encrypt"},"TLS with auto encrypt"),(0,r.kt)("p",null,"If Consul has been installed with the official Helm chart and you TLS enabled via auto encrypt using the following values:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml",metastring:'title="Consul Release Controller Helm Values"',title:'"Consul',Release:!0,Controller:!0,Helm:!0,'Values"':!0},"  tls:\n    enabled: true\n    enableAutoEncrypt: true\n    httpsOnly: false\n")),(0,r.kt)("p",null,"You can automatically configure the controller using the following config:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml",metastring:'title="Consul Release Controller Helm Values"',title:'"Consul',Release:!0,Controller:!0,Helm:!0,'Values"':!0},"autoEncrypt:\n  enabled: false\n")),(0,r.kt)("p",null,"If you are not using the default settings you can use the following Helm values to configure the chart to work correctly with your\ninstallation."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'controller:\n  enabled: "true"\n\n  container_config:\n    # Configure additional environment variables to be added to the controller container \n    env: []\n\n    # Add additional volume mounts to the controller container. \n    additional_volume_mounts: []\n\n    resources: {}\n\n  # Add additional volumes to the controller deployment.\n  additional_volumes: []\n\n  # Add additional init containers to the controller deployment.\n  additional_init_containers: []\n')))))}h.isMDXComponent=!0}}]);