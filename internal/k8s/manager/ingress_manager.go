package manager

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/apply"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils/query"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	netutil "github.com/GoSimplicity/AI-CloudOps/pkg/utils/net"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/retry"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// IngressManager Ingress 资源管理器
type IngressManager interface {
	// CreateIngress 创建Ingress
	CreateIngress(ctx context.Context, clusterID int, namespace string, restConfig *rest.Config, yml string) error
	// GetIngress 获取指定Ingress
	GetIngress(ctx context.Context, clusterID int, namespace, name string) (*networkingv1.Ingress, error)
	// GetIngressList 获取Ingress列表
	GetIngressList(ctx context.Context, clusterID int, namespace string, queryParams *query.Query) (*model.ListResp[*model.K8sIngress], error)
	// UpdateIngress 更新Ingress
	UpdateIngress(ctx context.Context, clusterID int, namespace string, restConfig *rest.Config, yml string) error
	DeleteIngress(ctx context.Context, clusterID int, namespace, name string, options metav1.DeleteOptions) error
	BatchDeleteIngresses(ctx context.Context, clusterID int, namespace string, ingressNames []string, options metav1.DeleteOptions) error
	// GetIngressEvent(ctx context.Context, clusterID int, namespace, name string, options metav1.ListOptions) (*model.ListResp[*model.K8sEvent], error)
	PatchIngress(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*networkingv1.Ingress, error)
	UpdateIngressStatus(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error
	TestIngressTLS(ctx context.Context, clusterID int, namespace string, port int) (*model.K8sTLSTestResult, error)
	CheckIngressBackendHealth(ctx context.Context, clusterID int, namespace, name string) ([]*model.K8sBackendHealth, error)
}

type ingressManager struct {
	logger *zap.Logger
	client client.K8sClient
}

// NewIngressManager 创建新的 IngressManager 实例
func NewIngressManager(logger *zap.Logger, client client.K8sClient) IngressManager {
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

// PatchIngress 部分更新Ingress
func (i *ingressManager) PatchIngress(ctx context.Context, clusterID int, namespace, name string, data []byte, patchType string) (*networkingv1.Ingress, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", clusterID),
			zap.Error(err))
		return nil, err
	}

	// 转换 patch 类型
	pt := types.PatchType(patchType)
	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Patch(ctx, name, pt, data, metav1.PatchOptions{})
	if err != nil {
		i.logger.Error("Patch Ingress失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.String("patchType", patchType),
			zap.Error(err))
		return nil, err
	}

	i.logger.Info("成功Patch Ingress",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", name))

	return ingress, nil
}

// UpdateIngressStatus 更新Ingress状态
func (i *ingressManager) UpdateIngressStatus(ctx context.Context, clusterID int, namespace string, ingress *networkingv1.Ingress) error {
	kubeClient, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return err
	}

	_, err = kubeClient.NetworkingV1().Ingresses(namespace).UpdateStatus(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		i.logger.Error("更新Ingress状态失败",
			zap.Int("clusterID", clusterID),
			zap.String("namespace", namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return err
	}

	i.logger.Info("成功更新Ingress状态",
		zap.Int("clusterID", clusterID),
		zap.String("namespace", namespace),
		zap.String("name", ingress.Name))

	return nil
}

// BatchDeleteIngresses 批量删除Ingress
func (i *ingressManager) BatchDeleteIngresses(ctx context.Context, clusterID int, namespace string, ingressNames []string, options metav1.DeleteOptions) error {

	cli, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return err
	}

	tasks := make([]retry.WrapperTask, 0, len(ingressNames))
	for _, name := range ingressNames {

		tasks = append(tasks, retry.WrapperTask{
			Backoff: retry.DefaultBackoff,

			Task: func(ctx context.Context) error {
				if err := cli.NetworkingV1().Ingresses(namespace).Delete(ctx, name, options); err != nil {
					i.logger.Error("删除Ingress失败", zap.Error(err),
						zap.Int("cluster_id", clusterID),
						zap.String("namespace", namespace),
						zap.String("name", name))
				}
				return nil
			},
			RetryCheck: func(err error) bool {
				return k8serrors.IsTimeout(err) ||
					k8serrors.IsTooManyRequests(err) ||
					k8serrors.IsServerTimeout(err) ||
					k8serrors.IsConflict(err)
			},
		})
	}
	err = retry.RunRetryWithConcurrency(ctx, 3, tasks)
	if err != nil {
		i.logger.Error("批量删除Ingress失败",
			zap.Error(err))

		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "批量删除Ingress失败")
	}
	return nil
}

// TestIngressTLS 测试 Ingress TLS
func (i *ingressManager) TestIngressTLS(ctx context.Context, clusterID int, host string, port int) (*model.K8sTLSTestResult, error) {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	tlsConfig := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	}

	conn, err := netutil.SockConn(ctx, "tcp://"+address, netutil.ConnOptions{
		Timeout:   5 * time.Second,
		TLSConfig: tlsConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect %s", address)
	}
	defer conn.Close()

	result := &model.K8sTLSTestResult{
		Valid:    false,
		Host:     host,
		Port:     port,
		TestTime: time.Now().Format(time.RFC3339),
	}

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return result, fmt.Errorf("非tls连接，无法检测证书")
	}

	// 已握手的连接直接取状态即可
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return result, fmt.Errorf("未找到对端证书")
	}

	cert := state.PeerCertificates[0]
	result.Valid = true
	result.CertDNSNames = strings.Join(cert.DNSNames, ",")
	result.CertExpiry = cert.NotAfter.Format(time.RFC3339)
	result.CertIssuer = cert.Issuer.String()
	result.CertSerialNumber = cert.SerialNumber.String()
	result.CertSubject = cert.Subject.String()
	return result, err
}

// CheckIngressBackendHealth 检查Ingress后端健康状态  遍历 Ingress 规则的后端，基于 Endpoints 的 Ready 状态判断健康情况
func (i *ingressManager) CheckIngressBackendHealth(ctx context.Context, clusterID int, ns, name string) ([]*model.K8sBackendHealth, error) {

	kubeClient, err := i.validKubeClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	ingress, err := i.detailIngress(ctx, kubeClient, ns, name)
	if err != nil {
		return nil, err
	}

	results := make([]*model.K8sBackendHealth, 0)
	for _, rule := range ingress.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			svcName := path.Backend.Service.Name
			svcPort := path.Backend.Service.Port.Number

			healthy, msg := i.checkServiceHealthy(ctx, kubeClient, ns, svcName, svcPort)
			results = append(results, &model.K8sBackendHealth{
				CheckTime:    time.Now().Format(time.DateTime),
				ServiceName:  svcName,
				ServicePort:  int(svcPort),
				Ready:        healthy,
				ErrorMessage: msg,
			})
		}
	}
	/*
		TCP/UDP (stream 模式)不会出现在 ingress.Spec.Rules 中，
		而是需要额外解析 Controller 的 ConfigMap
		这里暂时不处理，保持原生 HTTP Ingress 行为
	*/
	return results, nil
}

// checkServiceHealthy 完全被动方式，通过 Endpoints Ready 状态判断 Service 健康
func (i *ingressManager) checkServiceHealthy(ctx context.Context, kubeClient kubernetes.Interface, namespace, svcName string, portNum int32) (bool, string) {

	ep, err := kubeClient.CoreV1().Endpoints(namespace).Get(ctx, svcName, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取 Service Endpoints 失败",
			zap.String("namespace", namespace),
			zap.String("service", svcName),
			zap.Error(err))

		return false, fmt.Sprintf("获取 Service Endpoints 失败: %v", err)
	}
	if len(ep.Subsets) == 0 {
		return false, "没有可用的 Endpoint"
	}
	readyCount := 0
	totalCount := 0
	for _, subset := range ep.Subsets {
		for _, p := range subset.Ports {
			if p.Port != portNum {
				continue
			}
			totalCount += len(subset.Addresses) + len(subset.NotReadyAddresses)
			readyCount += len(subset.Addresses)
		}
	}
	if totalCount == 0 {
		return false, "未找到匹配端口的 Endpoint"
	}
	if readyCount == totalCount {
		return true, ""
	}
	return false, fmt.Sprintf("%d/%d 个 Endpoint 就绪", readyCount, totalCount)
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

func (i *ingressManager) detailIngress(ctx context.Context, cli kubernetes.Interface, namespace, name string) (*networkingv1.Ingress, error) {
	rawIngress, err := cli.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取Ingress详情失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Ingress详情失败")
	}
	return rawIngress, nil
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
