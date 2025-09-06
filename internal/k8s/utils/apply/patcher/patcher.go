package patcher

import (
	"context"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GeneratePatchBytes(obj, modifiedObj runtime.Object) ([]byte, types.PatchType, error) {
	// Serialize the current configuration of the object from the server.
	current, err := runtime.Encode(unstructured.UnstructuredJSONScheme, obj)
	if err != nil {
		return nil, "", fmt.Errorf("serializing current configuration from:\n%v\nis failed, err: %v", obj, err)
	}

	modified, err := util.GetModifiedConfiguration(modifiedObj, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return nil, "", fmt.Errorf("get modified configuration is failed, err: %v", err)
	}

	gvk, err := GetGroupVersionKind(obj)
	if err != nil {
		return nil, "", fmt.Errorf("retrieving gvk is failed, err: %v", err)
	}

	// Retrieve the original configuration of the object from the annotation.
	original, err := util.GetOriginalConfiguration(obj)
	if err != nil {
		return nil, "", fmt.Errorf("retrieving original configuration from:\n%v\nis failed, err: %v", obj, err)
	}

	var patchType types.PatchType
	var patch []byte
	var lookupPatchMeta strategicpatch.LookupPatchMeta

	createPatchErrFormat := "creating patch with:\noriginal:\n%s\nmodified:\n%s\ncurrent:\n%s\nis failed, err: %v"

	versionedObj, err := scheme.Scheme.New(*gvk)
	//if err != nil {
	//	return nil, "", fmt.Errorf("retrieving gvk is failed, err: %v", err)
	//}
	switch {
	case runtime.IsNotRegisteredError(err):
		patchType = types.MergePatchType
		preconditions := []mergepatch.PreconditionFunc{mergepatch.RequireKeyUnchanged("apiVersion"),
			mergepatch.RequireKeyUnchanged("kind"), mergepatch.RequireMetadataKeyUnchanged("name")}
		patch, err = jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current, preconditions...)
		if err != nil {
			if mergepatch.IsPreconditionFailed(err) {
				return nil, "", fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
			}
			return nil, "", fmt.Errorf(createPatchErrFormat, original, modified, current, err)
		}
	case err != nil:
		return nil, "", fmt.Errorf("getting instance of versioned object for %v is failed, err: %v", gvk, err)
	default:
		// Compute a three way strategic merge patch to send to server.
		patchType = types.StrategicMergePatchType
		lookupPatchMeta, err = strategicpatch.NewPatchMetaFromStruct(versionedObj)
		if err != nil {
			return nil, "", fmt.Errorf(createPatchErrFormat, original, modified, current, err)
		}

		patch, err = strategicpatch.CreateThreeWayMergePatch(original, modified, current, lookupPatchMeta, true)
		if err != nil {
			return nil, "", fmt.Errorf(createPatchErrFormat, original, modified, current, err)
		}
	}

	if string(patch) == "{}" {
		return patch, "", nil
	}

	return patch, patchType, err
}

func GetGroupVersionKind(obj runtime.Object) (*schema.GroupVersionKind, error) {
	t, err := meta.TypeAccessor(obj)
	if err != nil {
		return nil, err
	}

	gvk := schema.FromAPIVersionAndKind(t.GetAPIVersion(), t.GetKind())

	return &gvk, nil
}

func SetGroupVersionKind(obj runtime.Object, gvk *schema.GroupVersionKind) (runtime.Object, error) {
	t, err := meta.TypeAccessor(obj)
	if err != nil {
		return nil, err
	}

	t.SetAPIVersion(gvk.GroupVersion().String())
	t.SetKind(gvk.Kind)

	return obj, nil
}

func GetResource(ns, name string, obj client.Object, cl client.Reader) (bool, error) {
	err := cl.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: name}, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
