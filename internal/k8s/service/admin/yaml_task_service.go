package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlTask "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	TaskPending   = "Pending"
	TaskFailed    = "Failed"
	TaskSucceeded = "Succeeded"
)

type YamlTaskService interface {
	// GetYamlTaskList 获取 YAML 任务列表
	GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error)
	// CreateYamlTask 创建 YAML 任务
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// UpdateYamlTask 更新 YAML 任务
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// DeleteYamlTask 删除 YAML 任务
	DeleteYamlTask(ctx context.Context, id int) error
	// ApplyYamlTask 应用 YAML 任务
	ApplyYamlTask(ctx context.Context, id int) error
}

type yamlTaskService struct {
	yamlTaskDao     admin.YamlTaskDAO
	clusterDao      admin.ClusterDAO
	yamlTemplateDao admin.YamlTemplateDAO
	client          client.K8sClient
	l               *zap.Logger
}

func NewYamlTaskService(yamlTaskDao admin.YamlTaskDAO, clusterDao admin.ClusterDAO, yamlTemplateDao admin.YamlTemplateDAO, client client.K8sClient, l *zap.Logger) YamlTaskService {
	return &yamlTaskService{
		yamlTaskDao:     yamlTaskDao,
		clusterDao:      clusterDao,
		yamlTemplateDao: yamlTemplateDao,
		client:          client,
		l:               l,
	}
}

// GetYamlTaskList 获取 YAML 任务列表
func (y *yamlTaskService) GetYamlTaskList(ctx context.Context) ([]*model.K8sYamlTask, error) {
	return y.yamlTaskDao.ListAllYamlTasks(ctx)
}

// CreateYamlTask 创建 YAML 任务
func (y *yamlTaskService) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 检查模板是否存在
	_, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID)
	if err != nil {
		return fmt.Errorf("YAML 模板不存在: %w", err)
	}

	// 检查集群是否存在
	_, err = y.clusterDao.GetClusterByName(ctx, task.ClusterName)
	if err != nil {
		return fmt.Errorf("集群不存在: %w", err)
	}

	return y.yamlTaskDao.CreateYamlTask(ctx, task)
}

// UpdateYamlTask 更新 YAML 任务
func (y *yamlTaskService) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	// 检查任务是否存在
	_, err := y.yamlTaskDao.GetYamlTaskByID(ctx, task.ID)
	if err != nil {
		return fmt.Errorf("YAML 任务不存在: %w", err)
	}

	// 检查模板是否存在
	if task.TemplateID != 0 {
		_, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID)
		if err != nil {
			return fmt.Errorf("YAML 模板不存在: %w", err)
		}
	}

	// 检查集群是否存在
	if task.ClusterName != "" {
		_, err := y.clusterDao.GetClusterByName(ctx, task.ClusterName)
		if err != nil {
			return fmt.Errorf("集群不存在: %w", err)
		}
	}

	// 设置任务状态为 Pending，清空结果
	task.Status = TaskPending
	task.ApplyResult = ""

	return y.yamlTaskDao.UpdateYamlTask(ctx, task)
}

// DeleteYamlTask 删除 YAML 任务
func (y *yamlTaskService) DeleteYamlTask(ctx context.Context, id int) error {
	return y.yamlTaskDao.DeleteYamlTask(ctx, id)
}

// ApplyYamlTask 应用 YAML 任务
func (y *yamlTaskService) ApplyYamlTask(ctx context.Context, id int) error {
	// 获取任务信息
	task, err := y.yamlTaskDao.GetYamlTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("YAML 模板不存在: %w", err)
	}

	// 获取集群信息
	cluster, err := y.clusterDao.GetClusterByName(ctx, task.ClusterName)
	if err != nil {
		return fmt.Errorf("集群不存在: %w", err)
	}

	// 获取 Kubernetes 客户端
	dynClient, err := y.client.GetDynamicClient(cluster.ID)
	if err != nil {
		return fmt.Errorf("无法获取动态客户端: %w", err)
	}

	// 获取模板信息
	taskTemplate, err := y.yamlTemplateDao.GetYamlTemplateByID(ctx, task.TemplateID)
	if err != nil {
		return fmt.Errorf("获取 YAML 模板失败: %w", err)
	}

	// 处理变量替换
	yamlContent := taskTemplate.Content
	for _, variable := range task.Variables {
		parts := strings.SplitN(variable, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			yamlContent = strings.ReplaceAll(yamlContent, fmt.Sprintf("${%s}", key), value)
		}
	}

	// 解析 YAML 文件为 JSON
	jsonData, err := yamlTask.ToJSON([]byte(yamlContent))
	if err != nil {
		return fmt.Errorf("YAML转换JSON失败: %w", err)
	}

	// 创建 unstructured 对象
	obj := &unstructured.Unstructured{}
	if _, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonData, nil, obj); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	// 获取 GVR (GroupVersionResource)
	gvr := schema.GroupVersionResource{
		Group:    obj.GetObjectKind().GroupVersionKind().Group,
		Version:  obj.GetObjectKind().GroupVersionKind().Version,
		Resource: pkg.GetResourceName(obj.GetObjectKind().GroupVersionKind().Kind),
	}

	// 更新任务状态为成功
	task.Status = TaskSucceeded
	task.ApplyResult = "success"

	// 应用资源
	_, err = dynClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		// 资源已存在的情况处理
		if k8sErr.IsAlreadyExists(err) {
			y.l.Warn("资源已存在，请考虑更新", zap.Error(err))
		} else {
			y.l.Error("应用YAML任务失败: ", zap.Error(err))
		}
		// 更新任务状态为失败
		task.Status = TaskFailed
		task.ApplyResult = err.Error()
	}

	// 更新任务状态
	if err := y.yamlTaskDao.UpdateYamlTask(ctx, task); err != nil {
		y.l.Error("更新YAML任务失败: ", zap.Error(err))
	}

	return err
}
