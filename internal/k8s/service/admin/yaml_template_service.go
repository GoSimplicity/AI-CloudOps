package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	yamlTask "k8s.io/apimachinery/pkg/util/yaml"
)

type YamlTemplateService interface {
	// GetYamlTemplateList 获取 YAML 模板列表
	GetYamlTemplateList(ctx context.Context) ([]*model.K8sYamlTemplate, error)
	// CreateYamlTemplate 创建 YAML 模板
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// UpdateYamlTemplate 更新 YAML 模板
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// DeleteYamlTemplate 删除 YAML 模板
	DeleteYamlTemplate(ctx context.Context, id int) error
}

type yamlTemplateService struct {
	dao    dao.K8sDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewYamlTemplateService(dao dao.K8sDAO, client client.K8sClient, l *zap.Logger) YamlTemplateService {
	return &yamlTemplateService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (y *yamlTemplateService) GetYamlTemplateList(ctx context.Context) ([]*model.K8sYamlTemplate, error) {
	return y.dao.ListAllYamlTemplates(ctx)
}

// CreateYamlTemplate 创建 YAML 模板
func (y *yamlTemplateService) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 创建模板
	return y.dao.CreateYamlTemplate(ctx, template)
}

// UpdateYamlTemplate 更新 YAML 模板
func (y *yamlTemplateService) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	// 校验 YAML 格式是否正确
	if _, err := yamlTask.ToJSON([]byte(template.Content)); err != nil {
		return fmt.Errorf("YAML 格式错误: %w", err)
	}

	// 更新模板
	return y.dao.UpdateYamlTemplate(ctx, template)
}

// DeleteYamlTemplate 删除 YAML 模板
func (y *yamlTemplateService) DeleteYamlTemplate(ctx context.Context, id int) error {
	// 检查是否有任务正在使用该模板
	tasks, err := y.dao.GetYamlTaskByTemplateID(ctx, id)
	if err != nil {
		return err
	}

	// 如果有任务使用该模板，返回错误
	if len(tasks) > 0 {
		taskNames := make([]string, len(tasks))
		for i, task := range tasks {
			taskNames[i] = task.Name
		}
		return fmt.Errorf("该模板正在被以下任务使用: %v, 删除失败", taskNames)
	}

	// 删除模板
	return y.dao.DeleteYamlTemplate(ctx, id)
}
