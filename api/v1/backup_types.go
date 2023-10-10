/*
Copyright 2023.

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

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// Name of the backup
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of backup ("job", "cronjob")
	// +kubebuilder:validation:Required
	Type BackupType `json:"type"` // cronjob, job

	// Schedule of the backup ex: "0 0 * * *", run every day at midnight
	//
	// required if Type is set to "cronjob", see https://crontab.guru/ for more information
	Schedule string `json:"schedule,omitempty"`
}

// BackupStatus defines the observed state of Backup
type BackupStatus struct {
	// Phase of the backup
	Phase BackupPhase `json:"phase,omitempty"`

	// Conditions of the backup
	Conditions []BackupCondition `json:"conditions,omitempty"`
}

type BackupCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status string `json:"status,omitempty"` // True, False, Unknown

	// Type (Ready, Failed)
	Type string `json:"type,omitempty"` // Ready, Failed

	// Last time the condition transitioned from one status to another.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// Message about the condition for a better understanding
	Message string `json:"message,omitempty"`

	// Reason for the backup
	Reason string `json:"reason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Schedule",type=string,JSONPath=`.spec.schedule`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupSpec   `json:"spec,omitempty"`
	Status BackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
