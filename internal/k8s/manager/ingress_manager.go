package manager

import (
	"context"
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/apply"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// IngressManager Ingress 资源管理器
type IngressManager interface {
	CreateIngress(ctx context.Context, clusterID int, namespace string, restConfig *rest.Config, yml string) error
	GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error)
	GetIngressList(ctx context.Context, clusterID int, namespace string, queryParams *query.Query) (*model.ListResp[*model.K8sIngress], error)
	UpdateIngress(ctx context.Context, clusterID int, namespace string, restConfig *rest.Config, yml string) error
	DeleteIngress(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error
}

type ingressManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewIngressManager 创建新的 IngressManager 实例
func NewIngressManager(client client.K8sClient, logger *zap.Logger) IngressManager {
	return &ingressManager{
		logger: logger,
		client: client,
	}
}

// CreateIngress 创建Ingress
func (i *ingressManager) CreateIngress(ctx context.Context, clusterID int, namespace string, kubeConfig *rest.Config, yml string) error {

	applier, err := apply.NewApplier(ctx, namespace, kubeConfig)
	if err != nil {
		i.logger.Error("获取Apply对象失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return err
	}

	err = applier.Apply(strings.NewReader(yml))
	if err != nil {
		i.logger.Error("创建Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			//zap.String("name", ingress.Name),
			zap.Error(err))
	}
	return err
}

// GetIngress 获取指定Ingress
func (i *ingressManager) GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error) {
	kubeClient, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Ingress失败")
	}

	return ingress, nil
}

// GetIngressList 获取Ingress列表
func (i *ingressManager) GetIngressList(ctx context.Context, clusterID int, namespace string, queryParams *query.Query) (*model.ListResp[*model.K8sIngress], error) {

	kubeClient, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	ingressList, err := kubeClient.NetworkingV1().Ingresses(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: queryParams.Selector().String()})

	if err != nil {
		i.logger.Error("获取Ingress列表失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))

		return nil, err
	}

	objects := make([]runtime.Object, len(ingressList.Items))
	filtered := make([]runtime.Object, 0)

	for _, item := range ingressList.Items {
		objects = append(objects, item.DeepCopy())
	}

	for _, object := range objects {
		selected := true
		for field, value := range queryParams.Filters {
			if !i.filterIngressFunc(object, query.Filter{Field: field, Value: value}) {
				selected = false
				break
			}
		}

		if selected {
			filtered = append(filtered, object)
		}
	}

	sort.Slice(filtered, func(n, m int) bool {
		if !queryParams.Ascending {
			return i.sortIngressFunc(filtered[n], filtered[m], queryParams.SortBy)
		}
		return !i.sortIngressFunc(filtered[n], filtered[m], queryParams.SortBy)
	})

	total := len(filtered)
	if queryParams.Pagination == nil {
		queryParams.Pagination = query.DefaultPagination
	}

	start, end := queryParams.Pagination.GetValidPagination(total)

	items := make([]*model.K8sIngress, 0, end-start)
	for _, o := range filtered[start:end] {
		ingress, ok := o.(*networkingv1.Ingress)
		if ok {
			items = append(items, utils.ConvertToK8sIngress(ingress, clusterID))
		}
	}
	i.logger.Debug("成功获取 Ingress 列表",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.Int("count", len(filtered)))

	return &model.ListResp[*model.K8sIngress]{
		Items: items,
		Total: int64(len(filtered)),
	}, nil
}

// UpdateIngress 更新Ingress
func (i *ingressManager) UpdateIngress(ctx context.Context, clusterID int, namespace string, kubeConfig *rest.Config, yml string) error {

	applier, err := apply.NewApplier(ctx, namespace, kubeConfig)
	if err != nil {
		i.logger.Error("获取Apply对象失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.Error(err))
		return err
	}

	err = applier.Apply(strings.NewReader(yml))
	if err != nil {
		i.logger.Error("创建Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			//zap.String("name", ingress.Name),
			zap.Error(err))
	}
	return err
}

// DeleteIngress 删除Ingress
func (i *ingressManager) DeleteIngress(ctx context.Context, clusterID int, namespace, name string, deleteOptions metav1.DeleteOptions) error {
	cli, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return err
	}

	err = cli.NetworkingV1().Ingresses(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		i.logger.Error("删除Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除Ingress失败")
	}

	i.logger.Info("成功删除Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return nil
}

func (i *ingressManager) validKubeClient(ctx context.Context, clusterID int) (kubernetes.Interface, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("cluster_id", clusterID),
			zap.Error(err))

		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}
	return kubeClient, nil
}

func (i *ingressManager) filterIngressFunc(object runtime.Object, filter query.Filter) bool {
	ingress, ok := object.(*networkingv1.Ingress)
	if !ok {
		return false
	}
	switch filter.Field {
	// TODO: 人为规定的ingress状态字段
	case query.FieldStatus:
		return strings.EqualFold(utils.IngressStatus(ingress), string(filter.Value))
	case query.FieldSearch:
		return strings.Contains(ingress.Name, string(filter.Value))
	default:
		return query.DefaultObjectMetaFilter(ingress.ObjectMeta, filter)
	}
}

func (i *ingressManager) sortIngressFunc(left runtime.Object, right runtime.Object, field query.Field) bool {
	leftIngress, ok := left.(*networkingv1.Ingress)
	if !ok {
		return false
	}
	rightIngress, ok := right.(*networkingv1.Ingress)
	if !ok {
		return false
	}

	return query.DefaultObjectMetaCompare(leftIngress.ObjectMeta, rightIngress.ObjectMeta, field)
}
