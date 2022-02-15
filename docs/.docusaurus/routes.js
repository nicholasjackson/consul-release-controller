
import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/consul-release-controller/__docusaurus/debug',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug','261'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/config',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/config','fec'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/content',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/content','064'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/globalData',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/globalData','b7e'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/metadata',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/metadata','ec0'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/registry',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/registry','68e'),
    exact: true
  },
  {
    path: '/consul-release-controller/__docusaurus/debug/routes',
    component: ComponentCreator('/consul-release-controller/__docusaurus/debug/routes','958'),
    exact: true
  },
  {
    path: '/consul-release-controller/markdown-page',
    component: ComponentCreator('/consul-release-controller/markdown-page','9a5'),
    exact: true
  },
  {
    path: '/consul-release-controller/',
    component: ComponentCreator('/consul-release-controller/','618'),
    routes: [
      {
        path: '/consul-release-controller/',
        component: ComponentCreator('/consul-release-controller/','44b'),
        exact: true,
        'sidebar': "tutorialSidebar"
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*')
  }
];
