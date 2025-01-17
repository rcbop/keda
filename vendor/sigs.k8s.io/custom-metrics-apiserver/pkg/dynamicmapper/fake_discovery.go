/*
Copyright 2016 The Kubernetes Authors.

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

package dynamicmapper

import (
	"fmt"

	swagger "github.com/emicklei/go-restful-swagger12"
	openapi_v2 "github.com/google/gnostic/openapiv2"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/openapi"
	kubeversion "k8s.io/client-go/pkg/version"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/testing"
)

// NB: this is a copy of k8s.io/client-go/discovery/fake.  The original returns `nil, nil`
// for some methods, which is generally confuses lots of code.

type FakeDiscovery struct {
	*testing.Fake
}

func (c *FakeDiscovery) ServerResourcesForGroupVersion(groupVersion string) (*metav1.APIResourceList, error) {
	action := testing.ActionImpl{
		Verb:     "get",
		Resource: schema.GroupVersionResource{Resource: "resource"},
	}
	c.Invokes(action, nil)
	for _, resourceList := range c.Resources {
		if resourceList.GroupVersion == groupVersion {
			return resourceList, nil
		}
	}
	return nil, fmt.Errorf("GroupVersion %q not found", groupVersion)
}

func (c *FakeDiscovery) ServerResources() ([]*metav1.APIResourceList, error) {
	action := testing.ActionImpl{
		Verb:     "get",
		Resource: schema.GroupVersionResource{Resource: "resource"},
	}
	c.Invokes(action, nil)
	return c.Resources, nil
}

func (c *FakeDiscovery) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	sgs, err := c.ServerGroups()
	if err != nil {
		return nil, nil, err
	}
	resultGroups := []*metav1.APIGroup{}
	for i := range sgs.Groups {
		resultGroups = append(resultGroups, &sgs.Groups[i])
	}

	action := testing.ActionImpl{
		Verb:     "get",
		Resource: schema.GroupVersionResource{Resource: "resource"},
	}
	c.Invokes(action, nil)
	return resultGroups, c.Resources, nil
}

func (c *FakeDiscovery) ServerPreferredResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *FakeDiscovery) ServerPreferredNamespacedResources() ([]*metav1.APIResourceList, error) {
	return nil, nil
}

func (c *FakeDiscovery) ServerGroups() (*metav1.APIGroupList, error) {
	groups := map[string]*metav1.APIGroup{}
	groupVersions := map[metav1.GroupVersionForDiscovery]struct{}{}
	for _, resourceList := range c.Resources {
		groupVer, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, err
		}
		groupVerForDisc := metav1.GroupVersionForDiscovery{
			GroupVersion: resourceList.GroupVersion,
			Version:      groupVer.Version,
		}

		group, groupPresent := groups[groupVer.Group]
		if !groupPresent {
			group = &metav1.APIGroup{
				Name: groupVer.Group,
				// use the fist seen version as the preferred version
				PreferredVersion: groupVerForDisc,
			}
			groups[groupVer.Group] = group
		}

		// we'll dedup in the end by deleting the group-versions
		// from the global map one at a time
		group.Versions = append(group.Versions, groupVerForDisc)
		groupVersions[groupVerForDisc] = struct{}{}
	}

	groupList := make([]metav1.APIGroup, 0, len(groups))
	for _, group := range groups {
		newGroup := metav1.APIGroup{
			Name:             group.Name,
			PreferredVersion: group.PreferredVersion,
		}

		for _, groupVer := range group.Versions {
			if _, ok := groupVersions[groupVer]; ok {
				delete(groupVersions, groupVer)
				newGroup.Versions = append(newGroup.Versions, groupVer)
			}
		}

		groupList = append(groupList, newGroup)
	}

	return &metav1.APIGroupList{
		Groups: groupList,
	}, nil
}

func (c *FakeDiscovery) ServerVersion() (*version.Info, error) {
	action := testing.ActionImpl{}
	action.Verb = "get"
	action.Resource = schema.GroupVersionResource{Resource: "version"}

	c.Invokes(action, nil)
	versionInfo := kubeversion.Get()
	return &versionInfo, nil
}

func (c *FakeDiscovery) SwaggerSchema(version schema.GroupVersion) (*swagger.ApiDeclaration, error) {
	action := testing.ActionImpl{}
	action.Verb = "get"
	if version == v1.SchemeGroupVersion {
		action.Resource = schema.GroupVersionResource{Resource: "/swaggerapi/api/" + version.Version}
	} else {
		action.Resource = schema.GroupVersionResource{Resource: "/swaggerapi/apis/" + version.Group + "/" + version.Version}
	}

	c.Invokes(action, nil)
	return &swagger.ApiDeclaration{}, nil
}

func (c *FakeDiscovery) OpenAPISchema() (*openapi_v2.Document, error) {
	return &openapi_v2.Document{}, nil
}

func (c *FakeDiscovery) OpenAPIV3() openapi.Client {
	panic("unimplemented")
}

func (c *FakeDiscovery) RESTClient() restclient.Interface {
	return nil
}
