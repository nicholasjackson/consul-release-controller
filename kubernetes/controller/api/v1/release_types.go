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

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Releaser defines the configuration for the releaser plugin
	Releaser Releaser `json:"releaser,omitempty"`

	// Runtime defines the configuration for the runtime plugin
	Runtime Runtime `json:"runtime,omitempty"`

	// Strategy defines the configuration for the strategy plugin
	Strategy Strategy `json:"strategy,omitempty"`

	// Monitor defines the configuration for the strategy plugin
	Monitor Monitor `json:"monitor,omitempty"`
}

type Releaser struct {
	PluginName string         `json:"pluginName,omitempty"`
	Config     ReleaserConfig `json:"config,omitempty"`
}

type ReleaserConfig struct {
	ConsulService string `json:"consulService"`
}

type Runtime struct {
	PluginName string        `json:"pluginName,omitempty"`
	Config     RuntimeConfig `json:"config,omitempty"`
}

type RuntimeConfig struct {
	// Name of an existing Deployment in the same namespace
	Deployment string `json:"deployment,omitempty"`
}

type Strategy struct {
	PluginName string         `json:"pluginName,omitempty"`
	Config     StrategyConfig `json:"config,omitempty"`
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
	PluginName string        `json:"pluginName,omitempty"`
	Config     MonitorConfig `json:"config,omitempty"`
}

type MonitorConfig struct {
	Address string  `json:"address,omitempty"`
	Queries []Query `json:"queries,omitempty"`
}

type Query struct {
	Name   string `json:"name,omitempty"`
	Preset string `json:"preset,omitempty"`
	Min    int    `json:"min,omitempty"`
	Max    int    `json:"max,omitempty"`
}

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

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

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
