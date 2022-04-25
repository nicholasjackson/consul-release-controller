/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Webhooks []Webhook `json:"webhooks,omitempty"`

	// Releaser defines the configuration for the releaser plugin
	Releaser Releaser `json:"releaser,omitempty"`

	// Runtime defines the configuration for the runtime plugin
	Runtime Runtime `json:"runtime,omitempty"`

	// Strategy defines the configuration for the strategy plugin
	Strategy Strategy `json:"strategy,omitempty"`

	// Monitor defines the configuration for the strategy plugin
	Monitor Monitor `json:"monitor,omitempty"`

	// PostDeploymentTest defines the configuration for the post deployment tests plugin
	PostDeploymentTest Test `json:"postDeploymentTest,omitempty"`
}

type Webhook struct {
	Name       string        `json:"name"`
	PluginName string        `json:"pluginName"`
	Config     WebhookConfig `json:"config"`
}

type WebhookConfig struct {
	ID       string   `json:"id,omitempty"`
	Token    string   `json:"token,omitempty"`
	URL      string   `json:"url,omitempty"`
	Template string   `json:"template,omitempty"`
	Status   []string `json:"status,omitempty"`
}

type Releaser struct {
	PluginName string         `json:"pluginName"`
	Config     ReleaserConfig `json:"config"`
}

type ReleaserConfig struct {
	ConsulService string `json:"consulService"`
	Namespace     string `json:"namespace,omitempty"`
	Partition     string `json:"partition,omitempty"`
}

type Runtime struct {
	PluginName string        `json:"pluginName"`
	Config     RuntimeConfig `json:"config"`
}

type RuntimeConfig struct {
	// Name of an existing Deployment in the same namespace
	Deployment string `json:"deployment"`
}

type Strategy struct {
	PluginName string         `json:"pluginName"`
	Config     StrategyConfig `json:"config"`
}

type StrategyConfig struct {
	InitialDelay   string `json:"initialDelay,omitempty"`
	Interval       string `json:"interval,omitempty"`
	InitialTraffic int    `json:"initialTraffic,omitempty"`
	TrafficStep    int    `json:"trafficStep,omitempty"`
	MaxTraffic     int    `json:"maxTraffic,omitempty"`
	ErrorThreshold int    `json:"errorThreshold,omitempty"`
}

type Monitor struct {
	PluginName string        `json:"pluginName"`
	Config     MonitorConfig `json:"config"`
}

type MonitorConfig struct {
	Address string  `json:"address"`
	Queries []Query `json:"queries,omitempty"`
}

type Query struct {
	Name   string `json:"name,omitempty"`
	Preset string `json:"preset,omitempty"`
	Min    int    `json:"min,omitempty"`
	Max    int    `json:"max,omitempty"`
	Query  string `json:"query,omitempty"`
}

type Test struct {
	PluginName string     `json:"pluginName"`
	Config     TestConfig `json:"config"`
}

type TestConfig struct {
	Path               string `json:"path"`
	Method             string `json:"method"`
	Payload            string `json:"payload,omitempty"`
	RequiredTestPasses int    `json:"requiredTestPasses"`
	Interval           string `json:"interval"`
	Timeout            string `json:"timeout"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
