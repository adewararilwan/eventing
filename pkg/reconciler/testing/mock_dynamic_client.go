/*
Copyright 2019 The Knative Authors

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

package testing

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// All of the functions in dynamic.Interface get mocked equivalents.
type MockDynamicResource func(innerInterface dynamic.Interface, resource schema.GroupVersionResource) (MockHandled, dynamic.NamespaceableResourceInterface)

// All of the functions in dynamic.Resource get mocked equivalents. For the function
// dynamic.Resource.Foo(), the mocked equivalent will be:
// MockDynamicFoo func(ctx *MockDynamicContext[, Foo's arguments]) (MockHandled[, Foo's returns])
type MockDynamicCreate func(ctx *MockDynamicContext, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (MockHandled, *unstructured.Unstructured, error)
type MockDynamicUpdate func(ctx *MockDynamicContext, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (MockHandled, *unstructured.Unstructured, error)
type MockDynamicUpdateStatus func(ctx *MockDynamicContext, obj *unstructured.Unstructured, options metav1.UpdateOptions) (MockHandled, *unstructured.Unstructured, error)
type MockDynamicDelete func(ctx *MockDynamicContext, name string, options *metav1.DeleteOptions, subresources ...string) (MockHandled, error)
type MockDynamicDeleteCollection func(ctx *MockDynamicContext, options *metav1.DeleteOptions, listOptions metav1.ListOptions) (MockHandled, error)
type MockDynamicGet func(ctx *MockDynamicContext, name string, options metav1.GetOptions, subresources ...string) (MockHandled, *unstructured.Unstructured, error)
type MockDynamicList func(ctx *MockDynamicContext, opts metav1.ListOptions) (MockHandled, *unstructured.UnstructuredList, error)
type MockDynamicWatch func(ctx *MockDynamicContext, opts metav1.ListOptions) (MockHandled, watch.Interface, error)
type MockDynamicPatch func(ctx *MockDynamicContext, name string, pt types.PatchType, data []byte, options metav1.UpdateOptions, subresources ...string) (MockHandled, *unstructured.Unstructured, error)

type MockDynamicInterface struct {
	innerInterface dynamic.Interface
	mocks          DynamicMocks
}

var _ dynamic.Interface = (*MockDynamicInterface)(nil)

func NewMockDynamicInterface(innerInterface dynamic.Interface, mocks DynamicMocks) *MockDynamicInterface {
	return &MockDynamicInterface{
		innerInterface: innerInterface,
		mocks:          mocks,
	}
}

func (m MockDynamicInterface) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	for i, mockResource := range m.mocks.MockResources {
		handled, err := mockResource(m.innerInterface, resource)
		if handled == Handled {
			if len(m.mocks.MockResources) > 1 {
				m.mocks.MockResources = append(m.mocks.MockResources[:i], m.mocks.MockResources[i+1:]...)
			}
			return err
		}
	}

	// We want to wrap the returned value in a mockDynamicResourceInterface, so that it the rest of
	// the dynamic mocks still apply.
	return &mockDynamicResourceInterface{
		ctx: &MockDynamicContext{
			InnerInterface: m.innerInterface.Resource(resource),
			Resource:       resource,
		},
		mocks: m.mocks,
	}
}

// mockDynamicResourceInterface is a dynamic.ResourceInterface that allows mock responses to be
// returned, instead of calling the inner dynamic.ResourceInterface.
type mockDynamicResourceInterface struct {
	ctx   *MockDynamicContext
	mocks DynamicMocks
}

type MockDynamicContext struct {
	InnerInterface dynamic.ResourceInterface
	Resource       schema.GroupVersionResource
	Namespace      string
}

var _ dynamic.NamespaceableResourceInterface = (*mockDynamicResourceInterface)(nil)

// The mocks to run on each function type. Each function will run through the mocks in its list
// until one responds with 'Handled'. If there is more than one mock in the list, then the one that
// responds 'Handled' will be removed and not run on subsequent calls to the function. If no mocks
// respond 'Handled', then the real underlying client is called.
type DynamicMocks struct {
	// MockResources corresponds to dynamic.Interface.
	MockResources []MockDynamicResource

	// All other fields correspond to their dynamic.ResourceInterface equivalents.
	MockCreates           []MockDynamicCreate
	MockUpdates           []MockDynamicUpdate
	MockUpdateStatuses    []MockDynamicUpdateStatus
	MockDeletes           []MockDynamicDelete
	MockDeleteCollections []MockDynamicDeleteCollection
	MockGets              []MockDynamicGet
	MockLists             []MockDynamicList
	MockWatches           []MockDynamicWatch
	MockPatches           []MockDynamicPatch
}

func (m *mockDynamicResourceInterface) Namespace(ns string) dynamic.ResourceInterface {
	// We are being a little lazy. We reuse the same mockDynamicResourceInterface for both the
	// dynamic.NamespaceableResourceInterface and dynamic.ResourceInterface. Once a namespace is
	// set, it can't be set again. So we panic here. I don't expect this to occur in any 'normal'
	// code, because the compiler should not allow the second call to Namespace.
	if i, ok := m.ctx.InnerInterface.(dynamic.NamespaceableResourceInterface); ok {
		return &mockDynamicResourceInterface{
			ctx: &MockDynamicContext{
				InnerInterface: i.Namespace(ns),
				Resource:       m.ctx.Resource,
				Namespace:      ns,
			},
			mocks: m.mocks,
		}
	}
	panic("mockDynamicResourceInterface.Namespace() called when the inner interface is not a NamespaceableResourceInterface")
}

// All of the functions are handled almost identically:
// 1. Run through the mocks in order:
//   a. If the mock handled the request, then:
//      i. If there is at least one other mock in the list, remove this mock.
//      ii. Return the response from the mock.
// 2. No mock handled the request, so call the inner client.

func (m *mockDynamicResourceInterface) Create(obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	for i, mockCreate := range m.mocks.MockCreates {
		handled, u, err := mockCreate(m.ctx, obj, options, subresources...)
		if handled == Handled {
			if len(m.mocks.MockCreates) > 1 {
				m.mocks.MockCreates = append(m.mocks.MockCreates[:i], m.mocks.MockCreates[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.Create(obj, options, subresources...)
}

func (m *mockDynamicResourceInterface) Update(obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	for i, mockUpdate := range m.mocks.MockUpdates {
		handled, u, err := mockUpdate(m.ctx, obj, options, subresources...)
		if handled == Handled {
			if len(m.mocks.MockUpdates) > 1 {
				m.mocks.MockUpdates = append(m.mocks.MockUpdates[:i], m.mocks.MockUpdates[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.Update(obj, options, subresources...)
}

func (m *mockDynamicResourceInterface) UpdateStatus(obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	for i, mockUpdateStatus := range m.mocks.MockUpdateStatuses {
		handled, u, err := mockUpdateStatus(m.ctx, obj, options)
		if handled == Handled {
			if len(m.mocks.MockUpdateStatuses) > 1 {
				m.mocks.MockUpdateStatuses = append(m.mocks.MockUpdateStatuses[:i], m.mocks.MockUpdateStatuses[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.UpdateStatus(obj, options)
}

func (m *mockDynamicResourceInterface) Delete(name string, options *metav1.DeleteOptions, subresources ...string) error {
	for i, mockDelete := range m.mocks.MockDeletes {
		handled, err := mockDelete(m.ctx, name, options, subresources...)
		if handled == Handled {
			if len(m.mocks.MockDeletes) > 1 {
				m.mocks.MockDeletes = append(m.mocks.MockDeletes[:i], m.mocks.MockDeletes[i+1:]...)
			}
			return err
		}
	}
	return m.ctx.InnerInterface.Delete(name, options, subresources...)
}

func (m *mockDynamicResourceInterface) DeleteCollection(options *metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	for i, mockDeleteCollection := range m.mocks.MockDeleteCollections {
		handled, err := mockDeleteCollection(m.ctx, options, listOptions)
		if handled == Handled {
			if len(m.mocks.MockDeleteCollections) > 1 {
				m.mocks.MockDeleteCollections = append(m.mocks.MockDeleteCollections[:i], m.mocks.MockDeleteCollections[i+1:]...)
			}
			return err
		}
	}
	return m.ctx.InnerInterface.DeleteCollection(options, listOptions)
}

func (m *mockDynamicResourceInterface) Get(name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	for i, mockGet := range m.mocks.MockGets {
		handled, u, err := mockGet(m.ctx, name, options, subresources...)
		if handled == Handled {
			if len(m.mocks.MockGets) > 1 {
				m.mocks.MockGets = append(m.mocks.MockGets[:i], m.mocks.MockGets[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.Get(name, options, subresources...)
}

func (m *mockDynamicResourceInterface) List(opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	for i, mockList := range m.mocks.MockLists {
		handled, u, err := mockList(m.ctx, opts)
		if handled == Handled {
			if len(m.mocks.MockLists) > 1 {
				m.mocks.MockLists = append(m.mocks.MockLists[:i], m.mocks.MockLists[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.List(opts)
}

func (m *mockDynamicResourceInterface) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	for i, mockWatch := range m.mocks.MockWatches {
		handled, w, err := mockWatch(m.ctx, opts)
		if handled == Handled {
			if len(m.mocks.MockWatches) > 1 {
				m.mocks.MockWatches = append(m.mocks.MockWatches[:i], m.mocks.MockWatches[i+1:]...)
			}
			return w, err
		}
	}
	return m.ctx.InnerInterface.Watch(opts)
}

func (m *mockDynamicResourceInterface) Patch(name string, pt types.PatchType, data []byte, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	for i, mockPatch := range m.mocks.MockPatches {
		handled, u, err := mockPatch(m.ctx, name, pt, data, options, subresources...)
		if handled == Handled {
			if len(m.mocks.MockPatches) > 1 {
				m.mocks.MockPatches = append(m.mocks.MockPatches[:i], m.mocks.MockPatches[i+1:]...)
			}
			return u, err
		}
	}
	return m.ctx.InnerInterface.Patch(name, pt, data, options, subresources...)
}
