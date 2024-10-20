package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ali_resource"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"text/template"
)

const terraformTemplate = `
provider "alicloud" {
  region = "{{.Region}}"
}

resource "alicloud_vpc" "vpc" {
  vpc_name   = "{{.VPC.VpcName}}"
  cidr_block = "{{.VPC.CidrBlock}}"
}

resource "alicloud_vswitch" "vswitch" {
  vpc_id       = alicloud_vpc.vpc.id
  cidr_block   = "{{.VPC.VSwitchCidr}}"
  zone_id      = "{{.VPC.ZoneID}}"
  vswitch_name = "{{.Name}}-vswitch"
}

resource "alicloud_security_group" "sg" {
  name        = "{{.Security.SecurityGroupName}}"
  description = "{{.Security.SecurityGroupDescription}}"
  vpc_id      = alicloud_vpc.vpc.id
}

resource "alicloud_instance" "instance" {
  security_groups           = [alicloud_security_group.sg.id]
  instance_type             = "{{.Instance.InstanceType}}"
  system_disk_category      = "{{.Instance.SystemDiskCategory}}"
  system_disk_name          = "{{.Instance.SystemDiskName}}"
  system_disk_description   = "{{.Instance.SystemDiskDescription}}"
  image_id                  = "{{.Instance.ImageID}}"
  instance_name             = "{{.Instance.InstanceName}}"
  vswitch_id                = alicloud_vswitch.vswitch.id
  internet_max_bandwidth_out = {{.Instance.InternetMaxBandwidthOut}}
}
`

type AliResourceService interface {
	// CreateResource 创建云资源
	CreateResource(ctx context.Context, config model.TerraformConfig) error
	// GetResource 获取云资源
	GetResource(ctx context.Context, id int) (model.TerraformConfig, error)
	// UpdateResource 更新云资源
	UpdateResource(ctx context.Context, id int, updatedConfig model.TerraformConfig) error
	// DeleteResource 删除云资源
	DeleteResource(ctx context.Context, id int) error
}

type aliResourceService struct {
	logger       *zap.Logger
	dao          ali_resource.AliResourceDAO
	terraformBin string
}

func NewAliResourceService(logger *zap.Logger, dao ali_resource.AliResourceDAO) AliResourceService {
	return &aliResourceService{
		logger:       logger,
		dao:          dao,
		terraformBin: viper.GetString("terraform.bin_path"),
	}
}

// renderTerraformTemplate 渲染 Terraform 模板并写入指定目录的 main.tf 文件
func (a *aliResourceService) renderTerraformTemplate(config model.TerraformConfig, workDir string) error {
	// 反序列化 VPC 字段
	var vpc model.VPCConfig
	if err := json.Unmarshal(config.VPC, &vpc); err != nil {
		return fmt.Errorf("failed to deserialize VPCConfig: %w", err)
	}

	// 反序列化 Instance 字段
	var instance model.InstanceConfig
	if err := json.Unmarshal(config.Instance, &instance); err != nil {
		return fmt.Errorf("failed to deserialize InstanceConfig: %w", err)
	}

	// 反序列化 Security 字段
	var security model.SecurityConfig
	if err := json.Unmarshal(config.Security, &security); err != nil {
		return fmt.Errorf("failed to deserialize SecurityConfig: %w", err)
	}

	// 创建新的结构体，将所有字段传递给模板
	data := struct {
		Region   string
		Name     string
		VPC      model.VPCConfig
		Instance model.InstanceConfig
		Security model.SecurityConfig
	}{
		Region:   config.Region,
		Name:     config.Name,
		VPC:      vpc,
		Instance: instance,
		Security: security,
	}

	// 解析并执行模板
	tmpl, err := template.New("terraform").Parse(terraformTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse terraform template: %w", err)
	}

	// 渲染模板到内存缓冲区
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute terraform template: %w", err)
	}

	// 将渲染后的模板写入 main.tf 文件
	mainTFPath := filepath.Join(workDir, "main.tf")
	if err := os.WriteFile(mainTFPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write main.tf: %w", err)
	}

	return nil
}

// setupTerraform 初始化并计划 Terraform 在指定目录
func (a *aliResourceService) setupTerraform(ctx context.Context, workDir string) (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(workDir, a.terraformBin)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Terraform: %w", err)
	}

	// 初始化 Terraform
	if err := tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	// 执行 Terraform Plan
	if _, err := tf.Plan(ctx); err != nil {
		return nil, fmt.Errorf("terraform plan failed: %w", err)
	}

	return tf, nil
}

// applyTerraform 执行 Terraform Apply 在指定 Terraform 实例
func (a *aliResourceService) applyTerraform(ctx context.Context, tf *tfexec.Terraform) error {
	if err := tf.Apply(ctx); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}
	return nil
}

// destroyTerraform 执行 Terraform Destroy 在指定目录
func (a *aliResourceService) destroyTerraform(ctx context.Context, workDir string) error {
	tf, err := tfexec.NewTerraform(workDir, a.terraformBin)
	if err != nil {
		return fmt.Errorf("failed to initialize Terraform: %w", err)
	}

	// 初始化 Terraform (必要时)
	if err := tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// 执行 Terraform Plan
	if _, err := tf.Plan(ctx); err != nil {
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	// 执行 Terraform Destroy
	if err := tf.Destroy(ctx); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// CreateResource 创建云资源
func (a *aliResourceService) CreateResource(ctx context.Context, config model.TerraformConfig) error {
	a.logger.Info("开始创建阿里云资源", zap.String("name", config.Name))

	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd() // 获取当前工作目录
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("failed to get project root directory: %w", err)
	}

	// 指定工作目录为项目根目录下的 terraform 目录
	terraformDir := filepath.Join(projectRootDir, "terraform", config.Name)

	// 检查并创建 terraform 目录（如果不存在）
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		err = os.MkdirAll(terraformDir, 0755) // 创建目录
		if err != nil {
			a.logger.Error("创建 terraform 目录失败", zap.Error(err))
			return fmt.Errorf("failed to create terraform directory: %w", err)
		}
	}

	a.logger.Info("工作目录设置为:", zap.String("path", terraformDir))

	// 渲染 Terraform 配置文件到指定的 terraformDir
	if err := a.renderTerraformTemplate(config, terraformDir); err != nil {
		a.logger.Error("渲染 Terraform 模板失败", zap.Error(err))
		return err
	}

	// 初始化 Terraform 并计划
	tf, err := a.setupTerraform(ctx, terraformDir) // 将 terraformDir 传递给 setupTerraform
	if err != nil {
		a.logger.Error("Terraform 初始化或 Plan 执行失败", zap.Error(err))
		return err
	}

	// 执行 Terraform Apply
	if err := a.applyTerraform(ctx, tf); err != nil {
		a.logger.Error("Terraform Apply 执行失败", zap.Error(err))
		return err
	}

	// 保存资源信息到数据库
	resourceID, err := a.dao.Create(ctx, &config)
	if err != nil {
		a.logger.Error("保存资源信息到数据库失败", zap.Error(err))
		return err
	}

	a.logger.Info("阿里云资源创建成功", zap.Int("resource_id", resourceID))
	return nil
}

// GetResource 获取云资源
func (a *aliResourceService) GetResource(ctx context.Context, id int) (model.TerraformConfig, error) {
	a.logger.Info("获取阿里云资源", zap.Int("resource_id", id))

	// 从数据库获取资源配置信息
	config, err := a.dao.Get(ctx, id)
	if err != nil {
		a.logger.Error("获取资源信息失败", zap.Error(err))
		return model.TerraformConfig{}, err
	}

	a.logger.Info("获取阿里云资源成功", zap.Int("resource_id", id))
	return *config, nil
}

// UpdateResource 更新云资源
func (a *aliResourceService) UpdateResource(ctx context.Context, id int, updatedConfig model.TerraformConfig) error {
	a.logger.Info("更新阿里云资源", zap.Int("resource_id", id))

	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd() // 获取当前工作目录
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("failed to get project root directory: %w", err)
	}

	// 指定工作目录为项目根目录下的 terraform 目录
	terraformDir := filepath.Join(projectRootDir, "terraform", updatedConfig.Name)

	// 检查并创建 terraform 目录（如果不存在）
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		err = os.MkdirAll(terraformDir, 0755) // 创建目录
		if err != nil {
			a.logger.Error("创建 terraform 目录失败", zap.Error(err))
			return fmt.Errorf("failed to create terraform directory: %w", err)
		}
	}

	a.logger.Info("工作目录设置为:", zap.String("path", terraformDir))

	// 渲染 Terraform 配置文件
	if err := a.renderTerraformTemplate(updatedConfig, terraformDir); err != nil {
		a.logger.Error("渲染 Terraform 模板失败", zap.Error(err))
		return err
	}

	// 初始化 Terraform 并计划
	tf, err := a.setupTerraform(ctx, terraformDir)
	if err != nil {
		a.logger.Error("Terraform 初始化或 Plan 执行失败", zap.Error(err))
		return err
	}

	// 执行 Terraform Apply
	if err := a.applyTerraform(ctx, tf); err != nil {
		a.logger.Error("Terraform Apply 执行失败", zap.Error(err))
		return err
	}

	// 更新资源信息到数据库
	if err := a.dao.Update(ctx, id, &updatedConfig); err != nil {
		a.logger.Error("更新资源信息到数据库失败", zap.Error(err))
		return err
	}

	a.logger.Info("阿里云资源更新成功", zap.Int("resource_id", id))
	return nil
}

// DeleteResource 删除云资源
func (a *aliResourceService) DeleteResource(ctx context.Context, id int) error {
	a.logger.Info("删除阿里云资源", zap.Int("resource_id", id))

	// 获取资源配置信息
	config, err := a.dao.Get(ctx, id)
	if err != nil {
		a.logger.Error("获取资源信息失败", zap.Error(err))
		return err
	}

	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd() // 获取当前工作目录
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("failed to get project root directory: %w", err)
	}

	// 指定工作目录为项目根目录下的 terraform 目录
	terraformDir := filepath.Join(projectRootDir, "terraform", config.Name)

	// 渲染 Terraform 配置文件
	if err := a.renderTerraformTemplate(*config, terraformDir); err != nil {
		a.logger.Error("渲染 Terraform 模板失败", zap.Error(err))
		return err
	}

	// 执行 Terraform Destroy
	if err := a.destroyTerraform(ctx, terraformDir); err != nil {
		a.logger.Error("Terraform Destroy 执行失败", zap.Error(err))
		return err
	}

	// 删除资源信息从数据库
	if err := a.dao.Delete(ctx, id); err != nil {
		a.logger.Error("删除资源信息到数据库失败", zap.Error(err))
		return err
	}

	a.logger.Info("阿里云资源删除成功", zap.Int("resource_id", id))
	return nil
}
