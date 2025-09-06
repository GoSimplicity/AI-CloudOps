package apply

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerRuntimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	controllerRuntimeCluster "sigs.k8s.io/controller-runtime/pkg/cluster"
	"sync"
)

var runtimeClientManagerIns *KubeRuntimeClientManager

// KubeRuntimeClientManager 保存多集群运行时客户端
type KubeRuntimeClientManager struct {
	controllerRuntimeClusterMap sync.Map
	generalMutex                sync.Mutex
}

var once sync.Once

func NewKubeRuntimeClientManager() *KubeRuntimeClientManager {
	if runtimeClientManagerIns == nil {
		once.Do(func() {
			runtimeClientManagerIns = &KubeRuntimeClientManager{}
		})
	}
	return runtimeClientManagerIns
}

func (cm *KubeRuntimeClientManager) GetControllerRuntimeClient(clusterID int, restConfig *rest.Config) (controllerRuntimeClient.Client, error) {
	//clusterID = handleClusterID(clusterID)

	cls, err := cm.getControllerRuntimeCluster(clusterID, restConfig)
	if err != nil {
		return nil, err
	}

	return cls.GetClient(), nil
}

func (cm *KubeRuntimeClientManager) getControllerRuntimeCluster(clusterID int, restConfig *rest.Config) (controllerRuntimeCluster.Cluster, error) {
	cls, ok := cm.controllerRuntimeClusterMap.Load(clusterID)
	if ok {
		return cls.(controllerRuntimeCluster.Cluster), nil
	}

	//var cfg *rest.Config

	controllerClient, err := createControllerRuntimeCluster(restConfig)
	if err == nil {
		go func() {
			if err := controllerClient.Start(ctrl.SetupSignalHandler()); err != nil {
				fmt.Printf("failed to start controller runtime cluster, error: %s", err.Error())
			}
		}()

		if !controllerClient.GetCache().WaitForCacheSync(context.Background()) {
			return nil, fmt.Errorf("failed to wait for controller runtime cluster to sync")
		}
		cm.controllerRuntimeClusterMap.Store(clusterID, controllerClient)
	}
	return controllerClient, err
}

func createControllerRuntimeCluster(restConfig *rest.Config) (controllerRuntimeCluster.Cluster, error) {
	scheme := runtime.NewScheme()

	// add all known types
	// if you want to support custom types, call _ = yourCustomAPIGroup.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)

	c, err := controllerRuntimeCluster.New(restConfig, func(clusterOptions *controllerRuntimeCluster.Options) {
		clusterOptions.Scheme = scheme
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to init client")
	}

	return c, nil
}
