/*
Copyright 2018 The Knative Authors

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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/knative/pkg/apis"
)

func TestClusterChannelProvisionerValidate(t *testing.T) {
	tests := []struct {
		name string
		p    *ClusterChannelProvisioner
		want *apis.FieldError
	}{{
		name: "valid",
		p: &ClusterChannelProvisioner{
			Spec: ClusterChannelProvisionerSpec{},
		},
	}, {
		name: "empty",
		p:    &ClusterChannelProvisioner{},
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.p.Validate(context.TODO())
			if diff := cmp.Diff(test.want.Error(), got.Error()); diff != "" {
				t.Errorf("validate (-want, +got) = %v", diff)
			}
		})
	}
}
