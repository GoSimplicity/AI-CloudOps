package apply

import (
	"context"
	"k8s.io/client-go/rest"

	//"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/apply/patcher"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/serializer"
	"go.uber.org/zap"
	//"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	//"sigs.k8s.io/controller-runtime/pkg/client"

	//"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"strings"
)

type Resource struct {
	mainfest     string
	unstructured *unstructured.Unstructured
}

type ResourceApplyParams struct {
	ApplyStr         string
	ClusterID        int
	Cfg              *rest.Config
	runtimeClientMgr *KubeRuntimeClientManager
	//client   *kubernetes.Clientset
}

func manifestToUnstructured(manifest string) ([]*unstructured.Unstructured, map[schema.GroupVersionKind][]*Resource, error) {

	manifests := SplitManifests(manifest)

	errors := make([]error, 0)

	resourceMap := make(map[schema.GroupVersionKind][]*Resource)
	resources := make([]*unstructured.Unstructured, 0)

	for _, m := range manifests {
		if isEmptyManifest(m) {
			continue
		}
		u, err := serializer.NewDecoder().YamlToUnstructured([]byte(m))
		if err != nil {
			errors = append(errors, err)
			continue
		}

		gvk := u.GetObjectKind().GroupVersionKind()
		if !gvk.Empty() && resourceMap[gvk] == nil {
			resourceMap[gvk] = []*Resource{}
		}

		resources = append(resources, u)
		resourceMap[gvk] = append(resourceMap[gvk], &Resource{
			mainfest:     m,
			unstructured: u,
		})
	}
	return resources, resourceMap, utilerrors.NewAggregate(errors)
}

func isEmptyManifest(m string) bool {
	if m == "" {
		return true
	}
	for _, line := range strings.Split(m, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return false
		}
	}
	return true
}

func CreateOrPatchResource(ctx context.Context, params *ResourceApplyParams, log *zap.Logger) ([]*unstructured.Unstructured, error) {
	var errors []error

	curResource, _, err := manifestToUnstructured(params.ApplyStr)
	if err != nil {
		log.Error("Failed to convert currently deplyed resource yaml to Unstructured", zap.Error(err), zap.String("manifest", params.ApplyStr))
		return nil, err
	}

	runtimeCli, err := params.runtimeClientMgr.GetControllerRuntimeClient(params.ClusterID, params.Cfg)
	if err != nil {
		log.Error("Failed to get controller runtime client", zap.Error(err), zap.Int("clusterID", params.ClusterID))
		return nil, err
	}

	for _, u := range curResource {
		err = CreateOrPatchUnstructured(ctx, u, runtimeCli)
		if err != nil {
			log.Error("Failed to create or patch resource", zap.Error(err), zap.String("resource", u.GetName()))
			errors = append(errors, err)
		}
	}
	return nil, utilerrors.NewAggregate(errors)
}
