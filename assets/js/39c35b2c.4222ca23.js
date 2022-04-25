"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[731],{3905:function(t,e,a){a.d(e,{Zo:function(){return m},kt:function(){return c}});var n=a(7294);function r(t,e,a){return e in t?Object.defineProperty(t,e,{value:a,enumerable:!0,configurable:!0,writable:!0}):t[e]=a,t}function l(t,e){var a=Object.keys(t);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(t);e&&(n=n.filter((function(e){return Object.getOwnPropertyDescriptor(t,e).enumerable}))),a.push.apply(a,n)}return a}function o(t){for(var e=1;e<arguments.length;e++){var a=null!=arguments[e]?arguments[e]:{};e%2?l(Object(a),!0).forEach((function(e){r(t,e,a[e])})):Object.getOwnPropertyDescriptors?Object.defineProperties(t,Object.getOwnPropertyDescriptors(a)):l(Object(a)).forEach((function(e){Object.defineProperty(t,e,Object.getOwnPropertyDescriptor(a,e))}))}return t}function i(t,e){if(null==t)return{};var a,n,r=function(t,e){if(null==t)return{};var a,n,r={},l=Object.keys(t);for(n=0;n<l.length;n++)a=l[n],e.indexOf(a)>=0||(r[a]=t[a]);return r}(t,e);if(Object.getOwnPropertySymbols){var l=Object.getOwnPropertySymbols(t);for(n=0;n<l.length;n++)a=l[n],e.indexOf(a)>=0||Object.prototype.propertyIsEnumerable.call(t,a)&&(r[a]=t[a])}return r}var s=n.createContext({}),p=function(t){var e=n.useContext(s),a=e;return t&&(a="function"==typeof t?t(e):o(o({},e),t)),a},m=function(t){var e=p(t.components);return n.createElement(s.Provider,{value:e},t.children)},u={inlineCode:"code",wrapper:function(t){var e=t.children;return n.createElement(n.Fragment,{},e)}},d=n.forwardRef((function(t,e){var a=t.components,r=t.mdxType,l=t.originalType,s=t.parentName,m=i(t,["components","mdxType","originalType","parentName"]),d=p(a),c=r,k=d["".concat(s,".").concat(c)]||d[c]||u[c]||l;return a?n.createElement(k,o(o({ref:e},m),{},{components:a})):n.createElement(k,o({ref:e},m))}));function c(t,e){var a=arguments,r=e&&e.mdxType;if("string"==typeof t||r){var l=a.length,o=new Array(l);o[0]=d;var i={};for(var s in e)hasOwnProperty.call(e,s)&&(i[s]=e[s]);i.originalType=t,i.mdxType="string"==typeof t?t:r,o[1]=i;for(var p=2;p<l;p++)o[p]=a[p];return n.createElement.apply(null,o)}return n.createElement.apply(null,a)}d.displayName="MDXCreateElement"},728:function(t,e,a){a.r(e),a.d(e,{frontMatter:function(){return i},contentTitle:function(){return s},metadata:function(){return p},toc:function(){return m},default:function(){return d}});var n=a(7462),r=a(3366),l=(a(7294),a(3905)),o=["components"],i={sidebar_position:6},s="Webhook notification",p={unversionedId:"webhooks",id:"webhooks",title:"Webhook notification",description:"Consul Release Controller supports Webhooks for notifications, currently Discord and Slack are supported with default",source:"@site/docs/webhooks.md",sourceDirName:".",slug:"/webhooks",permalink:"/consul-release-controller/webhooks",editUrl:"https://github.com/nicholasjackson/consul-release-controller/tree/main/docs/templates/shared/docs/webhooks.md",tags:[],version:"current",sidebarPosition:6,frontMatter:{sidebar_position:6},sidebar:"tutorialSidebar",previous:{title:"post_deployment_test",permalink:"/consul-release-controller/post_deployment_test"},next:{title:"Metrics",permalink:"/consul-release-controller/metrics"}},m=[{value:"Slack Webhooks",id:"slack-webhooks",children:[{value:"Parameters",id:"parameters",children:[],level:3}],level:2},{value:"Discord Webhooks",id:"discord-webhooks",children:[{value:"Parameters",id:"parameters-1",children:[],level:3}],level:2},{value:"Custom Messages",id:"custom-messages",children:[{value:"Template Variables",id:"template-variables",children:[],level:3}],level:2}],u={toc:m};function d(t){var e=t.components,a=(0,r.Z)(t,o);return(0,l.kt)("wrapper",(0,n.Z)({},u,a,{components:e,mdxType:"MDXLayout"}),(0,l.kt)("h1",{id:"webhook-notification"},"Webhook notification"),(0,l.kt)("p",null,"Consul Release Controller supports Webhooks for notifications, currently ",(0,l.kt)("inlineCode",{parentName:"p"},"Discord")," and ",(0,l.kt)("inlineCode",{parentName:"p"},"Slack")," are supported with default\nand custom messages. "),(0,l.kt)("p",null,(0,l.kt)("strong",{parentName:"p"},"States")),(0,l.kt)("p",null,"Webhooks are called when Consul Release Controller enters the following states:"),(0,l.kt)("table",null,(0,l.kt)("thead",{parentName:"table"},(0,l.kt)("tr",{parentName:"thead"},(0,l.kt)("th",{parentName:"tr",align:null},"State"),(0,l.kt)("th",{parentName:"tr",align:null},"Results"),(0,l.kt)("th",{parentName:"tr",align:null},"Description"))),(0,l.kt)("tbody",{parentName:"table"},(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_configure"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_configured"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when a new release is created")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_deploy"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_complete"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when a new deployment is created")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_monitor"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_unhealthy, event_healthy"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when monitoring a deployment")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_scale"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_scaled"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when scaling a deployment")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_promote"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_promoted"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when promoting a candidate to the primary")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_rollback"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_complete"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when rolling back a failed deployment")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"state_destroy"),(0,l.kt)("td",{parentName:"tr",align:null},"event_fail, event_complete"),(0,l.kt)("td",{parentName:"tr",align:null},"Fired when removing a previously configured release")))),(0,l.kt)("p",null,"These states can be used to filter webhooks using the ",(0,l.kt)("inlineCode",{parentName:"p"},"status")," parameter to reduce ChatOps noise."),(0,l.kt)("h2",{id:"slack-webhooks"},"Slack Webhooks"),(0,l.kt)("p",null,"The following example shows how to configure a webhook that can post to Slack channels."),(0,l.kt)("pre",null,(0,l.kt)("code",{parentName:"pre",className:"language-yaml"},'  webhooks:\n    - name: "slack"\n      pluginName: "slack"\n      config:\n        url: "https://hooks.slack.com/services/T9JT4868N/34340Q02/h9N1ry9x29quExF3434f7J"\n    - name: "slack_custom"\n      pluginName: "slack"\n      config:\n        url: "https://hooks.slack.com/services/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"\n        status:\n          - state_deploy\n          - state_scale\n        template: |\n          Custom template message: State has changed to "{{ .State }}" for\n          the release "{{ .Name }}" in the namespace "{{ .Namespace }}".\n\n          The outcome was "{{ .Outcome }}"\n')),(0,l.kt)("h3",{id:"parameters"},"Parameters"),(0,l.kt)("table",null,(0,l.kt)("thead",{parentName:"table"},(0,l.kt)("tr",{parentName:"thead"},(0,l.kt)("th",{parentName:"tr",align:null},"Name"),(0,l.kt)("th",{parentName:"tr",align:null},"Type"),(0,l.kt)("th",{parentName:"tr",align:null},"Required"),(0,l.kt)("th",{parentName:"tr",align:null},"Description"))),(0,l.kt)("tbody",{parentName:"table"},(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"url"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"Yes"),(0,l.kt)("td",{parentName:"tr",align:null},"The Slack Webhook URL")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"template"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"No"),(0,l.kt)("td",{parentName:"tr",align:null},"Optional template to replace default Webhook message")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"status"),(0,l.kt)("td",{parentName:"tr",align:null},"[]string"),(0,l.kt)("td",{parentName:"tr",align:null},"No"),(0,l.kt)("td",{parentName:"tr",align:null},"List of statuses to send Webhook message, omitting this parameter calls the webhook for all statuses")))),(0,l.kt)("h2",{id:"discord-webhooks"},"Discord Webhooks"),(0,l.kt)("p",null,"The following example shows how to configure"),(0,l.kt)("pre",null,(0,l.kt)("code",{parentName:"pre",className:"language-yaml"},'  webhooks:\n    - name: "discord_custom"\n      pluginName: "discord"\n      config:\n        id: "94700915179898981"\n        token: "-OoJOZtJJoAjLBhREuuTtTxlP4q3J219SOGIF5X4O1rro34344wdfwfIPk8CPzPWXnSxBj"\n        template: |\n          Custom template message: State has changed to "{{ .State }}" for\n          the release "{{ .Name }}" in the namespace "{{ .Namespace }}".\n\n          The outcome was "{{ .Outcome }}"\n        status:\n          - state_deploy\n          - state_scale\n    - name: "discord"\n      pluginName: "discord"\n      config:\n        id: "947009151231496821"\n        token: "-OoJOZtJJoAjLBhREuuTtTxlP4q3J21gaeIPk8CPzPWXnSxBj"\n')),(0,l.kt)("h3",{id:"parameters-1"},"Parameters"),(0,l.kt)("table",null,(0,l.kt)("thead",{parentName:"table"},(0,l.kt)("tr",{parentName:"thead"},(0,l.kt)("th",{parentName:"tr",align:null},"Name"),(0,l.kt)("th",{parentName:"tr",align:null},"Type"),(0,l.kt)("th",{parentName:"tr",align:null},"Required"),(0,l.kt)("th",{parentName:"tr",align:null},"Description"))),(0,l.kt)("tbody",{parentName:"table"},(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"id"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"Yes"),(0,l.kt)("td",{parentName:"tr",align:null},"The Discord Webhook ID")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"token"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"Yes"),(0,l.kt)("td",{parentName:"tr",align:null},"The Discord Webhook token")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"template"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"No"),(0,l.kt)("td",{parentName:"tr",align:null},"Optional template to replace default Webhook message")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"status"),(0,l.kt)("td",{parentName:"tr",align:null},"[]string"),(0,l.kt)("td",{parentName:"tr",align:null},"No"),(0,l.kt)("td",{parentName:"tr",align:null},"List of statuses to send Webhook message, omitting this parameter calls the webhook for all statuses")))),(0,l.kt)("h2",{id:"custom-messages"},"Custom Messages"),(0,l.kt)("p",null,"Rather than have the Webhook send the default messages you can configure a template to be used instead."),(0,l.kt)("p",null,"Templates are written using Go Template, you can reference the Template Variables or use any of the flow control and\ndefault functions."),(0,l.kt)("pre",null,(0,l.kt)("code",{parentName:"pre",className:"language-go"},'Consul Release Controller state has changed to "{{ .State }}" for\nthe release "{{ .Name }}" in the namespace "{{ .Namespace }}".\n\nPrimary traffic: {{ .PrimaryTraffic }}\nCandidate traffic: {{ .CandidateTraffic }}\n\n{{ if ne .Error "" }}\nAn error occurred when processing: {{ .Error }}\n{{ else }}\nThe outcome is "{{ .Outcome }}"\n{{ end }}\n')),(0,l.kt)("h3",{id:"template-variables"},"Template Variables"),(0,l.kt)("table",null,(0,l.kt)("thead",{parentName:"table"},(0,l.kt)("tr",{parentName:"thead"},(0,l.kt)("th",{parentName:"tr",align:null},"Name"),(0,l.kt)("th",{parentName:"tr",align:null},"Type"),(0,l.kt)("th",{parentName:"tr",align:null},"Description"))),(0,l.kt)("tbody",{parentName:"table"},(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"Title"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"The Title for the Webhook message")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"Name"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"The Name of the release")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"State"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"Current state of the release")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"Outcome"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"The outcome of the status, success, fail, etc. See States table above")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"PrimaryTraffic"),(0,l.kt)("td",{parentName:"tr",align:null},"int"),(0,l.kt)("td",{parentName:"tr",align:null},"Percentage of Traffic distributed to the Primary instance 0-100")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"CandidateTraffic"),(0,l.kt)("td",{parentName:"tr",align:null},"int"),(0,l.kt)("td",{parentName:"tr",align:null},"Percentage of Traffic distributed to the Candidate instance 0-100")),(0,l.kt)("tr",{parentName:"tbody"},(0,l.kt)("td",{parentName:"tr",align:null},"Error"),(0,l.kt)("td",{parentName:"tr",align:null},"string"),(0,l.kt)("td",{parentName:"tr",align:null},"An error message if the status failed")))))}d.isMDXComponent=!0}}]);