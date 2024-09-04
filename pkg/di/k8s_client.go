package di

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "open-cluster-management.io/api/cluster/v1"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// InitK8sClient 初始化 Kubernetes controller-runtime client
func InitK8sClient() client.Client {
	// 初始化日志记录器
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	// 获取 Kubernetes 配置
	cfg := config.GetConfigOrDie()
	cfg.QPS = 10
	cfg.Burst = 100
	// 创建 manager 并附带 Scheme
	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Scheme: setupScheme(),
	})
	if err != nil {
		fmt.Printf("无法启动 K8s manager: %v\n", err)
	}
	// 启动 manager 并阻塞直到缓存启动
	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			fmt.Printf("无法启动 K8s manager: %v\n", err)
		}
	}()
	// 等待缓存同步
	mgr.GetCache().WaitForCacheSync(context.Background())
	// 获取 Kubernetes client
	k8sClient := mgr.GetClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法初始化K8s client: %v\n", err)
	}

	return k8sClient
}

// SetupScheme 初始化并返回注册了 Kubernetes 资源的 Scheme
func setupScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}
