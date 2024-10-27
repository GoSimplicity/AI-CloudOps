package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ali_resource"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ecs"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/tree"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"time"
)

const terraformTemplate = `
provider "alicloud" {
  access_key = "{{.Key}}"
  secret_key = "{{.Secret}}"
  region     = "{{.Region}}"
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
  security_groups             = [alicloud_security_group.sg.id]
  instance_type               = "{{.Instance.InstanceType}}"
  system_disk_category        = "{{.Instance.SystemDiskCategory}}"
  system_disk_name            = "{{.Instance.SystemDiskName}}"
  system_disk_description     = "{{.Instance.SystemDiskDescription}}"
  image_id                    = "{{.Instance.ImageID}}"
  instance_name               = "{{.Instance.InstanceName}}"
  vswitch_id                  = alicloud_vswitch.vswitch.id
  internet_max_bandwidth_out  = {{.Instance.InternetMaxBandwidthOut}}
}

output "public_ip" {
  description = "实例的公网 IP 地址"
  value       = alicloud_instance.instance.public_ip
}

output "private_ip" {
  description = "实例的私有 IP 地址"
  value       = alicloud_instance.instance.private_ip
}

output "vpc_id" {
  description = "VPC ID"
  value       = alicloud_vpc.vpc.id
}

output "vswitch_id" {
  description = "VSwitch ID"
  value       = alicloud_vswitch.vswitch.id
}

output "security_group_id" {
  description = "Security Group ID"
  value       = alicloud_security_group.sg.id
}
`

type AliResourceService interface {
	// CreateResource 创建云资源
	CreateResource(ctx context.Context, config model.TerraformConfig) (string, error)
	// GetTaskStatus 获取任务状态
	GetTaskStatus(ctx context.Context, taskID string) (model.Task, error)
	// UpdateResource 更新云资源
	UpdateResource(ctx context.Context, id int, updatedConfig model.TerraformConfig) error
	// DeleteResource 删除云资源
	DeleteResource(ctx context.Context, id int) error
	// StartWorker 启动后台任务
	StartWorker()
}

type aliResourceService struct {
	logger       *zap.Logger
	dao          ali_resource.AliResourceDAO
	terraformBin string
	key          string
	secret       string
	redisClient  redis.Cmdable
	ecsDao       ecs.TreeEcsDAO
	semaphore    chan struct{}
}

func NewAliResourceService(logger *zap.Logger, dao ali_resource.AliResourceDAO, redisClient redis.Cmdable, ecsDao ecs.TreeEcsDAO) AliResourceService {
	return &aliResourceService{
		logger:       logger,
		dao:          dao,
		ecsDao:       ecsDao,
		redisClient:  redisClient,
		terraformBin: viper.GetString("terraform.bin_path"),
		key:          viper.GetString("terraform.key"),
		secret:       viper.GetString("terraform.secret"),
		semaphore:    make(chan struct{}, 10),
	}
}

// CreateResource 创建云资源（异步）
func (a *aliResourceService) CreateResource(ctx context.Context, config model.TerraformConfig) (string, error) {
	// 生成唯一任务 ID
	taskID := uuid.New().String()

	// 创建任务结构
	task := model.Task{
		TaskID:     taskID,
		Config:     config,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Action:     "create", // 设置任务类型为创建
		RetryCount: 0,
	}

	// 创建任务记录
	if err := a.dao.CreateTask(ctx, &task); err != nil {
		a.logger.Error("创建任务记录失败", zap.Error(err))
		return "", fmt.Errorf("创建任务记录失败: %w", err)
	}

	// 将任务 ID 推送到 Redis 队列
	queueName := "task_queue"
	if err := a.redisClient.LPush(ctx, queueName, taskID).Err(); err != nil {
		a.logger.Error("将任务推送到队列失败", zap.Error(err))
		return "", fmt.Errorf("将任务推送到队列失败: %w", err)
	}

	return taskID, nil
}

// GetTaskStatus 获取任务状态
func (a *aliResourceService) GetTaskStatus(ctx context.Context, taskID string) (model.Task, error) {
	task, err := a.dao.GetTaskByID(ctx, taskID)
	if err != nil {
		a.logger.Error("获取任务状态失败", zap.String("task_id", taskID), zap.Error(err))
		return model.Task{}, fmt.Errorf("获取任务状态失败: %w", err)
	}

	return *task, nil
}

// StartWorker 启动后台工作者，处理队列中的任务
func (a *aliResourceService) StartWorker() {
	queueName := "task_queue"

	for {
		// 从队列中阻塞获取一个任务 ID
		result, err := a.redisClient.BRPop(context.Background(), 0, queueName).Result()
		if err != nil {
			a.logger.Error("从队列中获取任务失败", zap.Error(err))
			time.Sleep(time.Second * 5)
			continue
		}

		if len(result) < 2 {
			a.logger.Error("无效的任务 ID")
			continue
		}

		actualTaskID := result[1]

		// 处理任务
		go a.processTask(actualTaskID)
	}
}

// processTask 处理单个任务
func (a *aliResourceService) processTask(taskID string) {
	// 获取信号量
	a.semaphore <- struct{}{}
	defer func() { <-a.semaphore }()

	ctx := context.Background()

	// 获取任务数据
	task, err := a.dao.GetTaskByID(ctx, taskID)
	if err != nil {
		a.logger.Error("获取任务数据失败", zap.String("task_id", taskID), zap.Error(err))
		return
	}

	// 更新任务状态为 processing
	if err := a.dao.UpdateTaskStatus(ctx, taskID, "processing", "", nil); err != nil {
		a.logger.Error("更新任务状态失败", zap.String("task_id", taskID), zap.Error(err))
		return
	}

	// 根据 Action 调用相应的方法
	switch task.Action {
	case "create":
		err = a.executeCreateResource(ctx, task.Config)
	case "update":
		err = a.executeUpdateResource(ctx, task.Config)
	case "delete":
		err = a.executeDeleteResource(ctx, task.Config)
	default:
		a.logger.Error("未知的任务类型", zap.String("task_id", taskID), zap.String("action", task.Action))
		// 更新任务状态为 failed
		a.dao.UpdateTaskStatus(ctx, taskID, "failed", "未知的任务类型", nil)
		return
	}

	if err != nil {
		a.logger.Error("后台处理任务失败", zap.String("task_id", taskID), zap.Error(err))
		// 检查是否超过最大重试次数
		if task.RetryCount < 3 {
			newRetryCount := task.RetryCount + 1
			a.dao.UpdateTaskStatus(ctx, taskID, "pending", "任务失败，准备重试", &newRetryCount)
			a.redisClient.LPush(ctx, "task_queue", taskID)
			a.logger.Info("任务重试中", zap.String("task_id", taskID), zap.Int("retry_count", task.RetryCount))
		} else {
			// 更新任务状态为 failed，并记录错误信息
			if updateErr := a.dao.UpdateTaskStatus(ctx, taskID, "failed", err.Error(), nil); updateErr != nil {
				a.logger.Error("更新任务状态失败", zap.String("task_id", taskID), zap.Error(updateErr))
			}
		}
		return
	}

	// 更新任务状态为 success
	if err := a.dao.UpdateTaskStatus(ctx, taskID, "success", "", nil); err != nil {
		a.logger.Error("更新任务状态失败", zap.String("task_id", taskID), zap.Error(err))
		return
	}

	a.logger.Info("任务处理成功", zap.String("task_id", taskID))
}

// executeCreateResource 执行实际的资源创建逻辑（原 CreateResource 逻辑）
func (a *aliResourceService) executeCreateResource(ctx context.Context, config model.TerraformConfig) error {
	// 获取当前时间，用于多处使用
	currentTime := time.Now().UTC().Format(time.RFC3339)

	// 解析配置
	instanceConfig, vpcConfig, securityConfig, err := tree.ParseConfigs(config, a.logger)
	if err != nil {
		return err
	}

	// 生成资源的 Hash 值
	hash := generateHash(config)

	// 创建 ResourceEcs 实例
	resource := &model.ResourceEcs{
		ResourceTree: model.ResourceTree{
			InstanceName:     config.Name,
			Hash:             hash,
			Vendor:           "2",
			CreateByOrder:    true,
			Image:            instanceConfig.ImageID,
			VpcId:            vpcConfig.VpcName,
			ZoneId:           instanceConfig.AvailabilityZone,
			Env:              config.Env,
			PayType:          config.PayType,
			Status:           "创建中",
			Description:      config.Description,
			Tags:             config.Tags,
			SecurityGroupIds: securityConfig.SecurityGroupVpcID,
			CreationTime:     currentTime,
		},
		InstanceType:    instanceConfig.InstanceType,
		ImageId:         instanceConfig.ImageID,
		StartTime:       currentTime,
		LastInvokedTime: currentTime,
	}

	// 保存资源记录到数据库
	if err := a.ecsDao.Create(ctx, resource); err != nil {
		a.logger.Error("创建 ECS 资源失败", zap.Error(err))
		return fmt.Errorf("创建 ECS 资源失败: %w", err)
	}

	ecsResource, err := a.ecsDao.GetByHash(ctx, hash)
	if err != nil {
		a.logger.Error("获取 ECS 资源失败", zap.Error(err))
		return fmt.Errorf("获取 ECS 资源失败: %w", err)
	}

	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd()
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("获取项目根目录失败: %w", err)
	}

	// 构建 Terraform 工作目录路径
	terraformDir := filepath.Join(projectRootDir, "terraform", fmt.Sprintf("resource_%d", ecsResource.ID))

	// 确保 Terraform 目录存在
	if err := tree.EnsureDir(terraformDir, a.logger); err != nil {
		return err
	}

	a.logger.Info("设置工作目录", zap.String("path", terraformDir))

	// 渲染 Terraform 配置文件
	if err := tree.RenderTerraformTemplate(config, terraformDir, terraformTemplate, a.key, a.secret, vpcConfig, instanceConfig, securityConfig); err != nil {
		a.logger.Error("渲染 Terraform 模板失败", zap.Error(err))
		return fmt.Errorf("渲染 Terraform 模板失败: %w", err)
	}

	// 初始化并计划 Terraform
	tf, err := tree.SetupTerraform(ctx, terraformDir, a.terraformBin)
	if err != nil {
		a.logger.Error("Terraform 初始化或 Plan 执行失败", zap.Error(err))
		return fmt.Errorf("terraform 初始化或 Plan 失败: %w", err)
	}

	// 执行 Terraform Apply
	if err := tree.ApplyTerraform(ctx, tf); err != nil {
		a.logger.Error("Terraform Apply 执行失败", zap.Error(err))
		return fmt.Errorf("terraform Apply 失败: %w", err)
	}

	// 获取并解析 Terraform 状态
	state, err := tree.GetTerraformState(ctx, tf, a.logger)
	if err != nil {
		return err
	}

	// 提取 Terraform 输出的 IP 地址
	publicIP, privateIP, err := tree.ExtractIPs(state, a.logger)
	if err != nil {
		return err
	}

	// 更新资源记录的 IP 地址和状态
	updatedResource := &model.ResourceEcs{
		ResourceTree: model.ResourceTree{
			Hash:             hash,
			IpAddr:           publicIP, // 单个公网 IP
			Status:           "运行中",
			PrivateIpAddress: privateIP,
			PublicIpAddress:  publicIP,
		},
	}

	if err := a.ecsDao.UpdateByHash(ctx, updatedResource); err != nil {
		a.logger.Error("更新资源记录失败", zap.Error(err))
		return fmt.Errorf("更新资源记录失败: %w", err)
	}

	return nil
}

func (a *aliResourceService) executeUpdateResource(ctx context.Context, config model.TerraformConfig) error {
	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd()
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("获取项目根目录失败: %w", err)
	}

	// 构建 Terraform 工作目录路径
	terraformDir := filepath.Join(projectRootDir, "terraform", fmt.Sprintf("resource_%d", config.ID))

	// 确保 Terraform 目录存在
	if err := tree.EnsureDir(terraformDir, a.logger); err != nil {
		return err
	}

	a.logger.Info("设置工作目录", zap.String("path", terraformDir))

	// 解析配置
	instanceConfig, vpcConfig, securityConfig, err := tree.ParseConfigs(config, a.logger)
	if err != nil {
		return err
	}

	// 渲染 Terraform 配置文件
	if err := tree.RenderTerraformTemplate(config, terraformDir, terraformTemplate, a.key, a.secret, vpcConfig, instanceConfig, securityConfig); err != nil {
		a.logger.Error("渲染 Terraform 模板失败", zap.Error(err))
		return fmt.Errorf("渲染 Terraform 模板失败: %w", err)
	}

	// 获取现有资源
	existingResource, err := a.ecsDao.GetByID(ctx, config.ID)
	if err != nil {
		a.logger.Error("获取现有资源失败", zap.Error(err))
		return fmt.Errorf("获取现有资源失败: %w", err)
	}

	// 更新现有资源的字段
	existingResource.ResourceTree.InstanceName = config.Name
	existingResource.ResourceTree.Env = config.Env
	existingResource.ResourceTree.PayType = config.PayType
	existingResource.ResourceTree.Description = config.Description
	existingResource.ResourceTree.Tags = config.Tags

	// 更新 Hash
	existingResource.ResourceTree.Hash = generateHash(config)

	// 保存更新后的资源
	if err := a.ecsDao.Update(ctx, existingResource); err != nil {
		a.logger.Error("更新 ECS 资源失败", zap.Error(err))
		return fmt.Errorf("更新 ECS 资源失败: %w", err)
	}

	// 初始化并计划 Terraform
	tf, err := tree.SetupTerraform(ctx, terraformDir, a.terraformBin)
	if err != nil {
		a.logger.Error("Terraform 初始化或 Plan 执行失败", zap.Error(err))
		return fmt.Errorf("terraform 初始化或 Plan 失败: %w", err)
	}

	// 执行 Terraform Apply
	if err := tree.ApplyTerraform(ctx, tf); err != nil {
		a.logger.Error("Terraform Apply 执行失败", zap.Error(err))
		return fmt.Errorf("terraform Apply 失败: %w", err)
	}

	// 获取并解析 Terraform 状态
	state, err := tree.GetTerraformState(ctx, tf, a.logger)
	if err != nil {
		return err
	}

	// 提取 Terraform 输出的 IP 地址
	publicIP, privateIP, err := tree.ExtractIPs(state, a.logger)
	if err != nil {
		return err
	}

	// 更新资源记录的 IP 地址和状态
	updatedResource := &model.ResourceEcs{
		ResourceTree: model.ResourceTree{
			InstanceName:     config.Name,
			Hash:             generateHash(config),
			IpAddr:           publicIP, // 单个公网 IP
			Status:           "运行中",
			PrivateIpAddress: privateIP,
			PublicIpAddress:  publicIP,
		},
	}

	if err := a.ecsDao.Update(ctx, updatedResource); err != nil {
		a.logger.Error("更新资源记录失败", zap.Error(err))
		return fmt.Errorf("更新资源记录失败: %w", err)
	}

	return nil
}

// UpdateResource 更新云资源
func (a *aliResourceService) UpdateResource(ctx context.Context, id int, updatedConfig model.TerraformConfig) error {
	// 首先，获取现有资源以确保其存在
	existingResource, err := a.ecsDao.GetByID(ctx, id)
	if err != nil {
		a.logger.Error("获取现有资源失败", zap.Error(err))
		return fmt.Errorf("获取现有资源失败: %w", err)
	}

	// 设置资源 ID 到配置中，用于更新
	updatedConfig.ID = existingResource.ID

	// 生成唯一任务 ID
	taskID := uuid.New().String()

	// 创建任务结构
	task := model.Task{
		TaskID:     taskID,
		Config:     updatedConfig,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Action:     "update",
		RetryCount: 0,
	}

	// 创建任务记录
	if err := a.dao.CreateTask(ctx, &task); err != nil {
		a.logger.Error("创建更新任务记录失败", zap.Error(err))
		return fmt.Errorf("创建更新任务记录失败: %w", err)
	}

	// 将任务 ID 推送到 Redis 队列
	queueName := "task_queue"
	if err := a.redisClient.LPush(ctx, queueName, taskID).Err(); err != nil {
		a.logger.Error("将更新任务推送到队列失败", zap.Error(err))
		return fmt.Errorf("将更新任务推送到队列失败: %w", err)
	}

	return nil
}

// DeleteResource 删除云资源
func (a *aliResourceService) DeleteResource(ctx context.Context, id int) error {
	// 首先，获取现有资源以确保其存在
	existingResource, err := a.ecsDao.GetByID(ctx, id)
	if err != nil {
		a.logger.Error("获取现有资源失败", zap.Error(err))
		return fmt.Errorf("获取现有资源失败: %w", err)
	}

	// 生成唯一任务 ID
	taskID := uuid.New().String()

	// 创建任务结构
	task := model.Task{
		TaskID:     taskID,
		Config:     model.TerraformConfig{ID: existingResource.ID},
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Action:     "delete", // 设置任务类型为删除
		RetryCount: 0,        // 初始化重试次数
	}

	// 创建任务记录
	if err := a.dao.CreateTask(ctx, &task); err != nil {
		a.logger.Error("创建删除任务记录失败", zap.Error(err))
		return fmt.Errorf("创建删除任务记录失败: %w", err)
	}

	// 将任务 ID 推送到 Redis 队列
	queueName := "task_queue"
	if err := a.redisClient.LPush(ctx, queueName, taskID).Err(); err != nil {
		a.logger.Error("将删除任务推送到队列失败", zap.Error(err))
		return fmt.Errorf("将删除任务推送到队列失败: %w", err)
	}

	return nil
}

// executeDeleteResource 执行实际的资源删除逻辑
func (a *aliResourceService) executeDeleteResource(ctx context.Context, config model.TerraformConfig) error {
	// 获取现有资源
	existingResource, err := a.ecsDao.GetByID(ctx, config.ID)
	if err != nil {
		a.logger.Error("获取现有资源失败", zap.Error(err))
		return fmt.Errorf("获取现有资源失败: %w", err)
	}

	if len(existingResource.BindNodes) > 0 {
		existingResource.Status = "错误"

		err = a.ecsDao.UpdateEcsResourceStatusByHash(ctx, existingResource)
		if err != nil {
			a.logger.Error("更新资源状态失败", zap.Error(err))
			return err
		}

		return ErrResourceBound
	}

	existingResource.Status = "删除中"

	err = a.ecsDao.UpdateEcsResourceStatusByHash(ctx, existingResource)
	if err != nil {
		a.logger.Error("更新资源状态失败", zap.Error(err))
		return err
	}

	// 获取项目根目录（当前工作目录）
	projectRootDir, err := os.Getwd()
	if err != nil {
		a.logger.Error("获取项目根目录失败", zap.Error(err))
		return fmt.Errorf("获取项目根目录失败: %w", err)
	}

	// 构建 Terraform 工作目录路径，基于资源 ID
	terraformDir := filepath.Join(projectRootDir, "terraform", fmt.Sprintf("resource_%d", existingResource.ID))

	// 初始化 Terraform
	_, err = tree.SetupTerraform(ctx, terraformDir, a.terraformBin)
	if err != nil {
		a.logger.Error("Terraform 初始化失败", zap.Error(err))
		return fmt.Errorf("terraform 初始化失败: %w", err)
	}

	// 执行 Terraform Destroy
	if err := tree.DestroyTerraform(ctx, terraformDir, a.terraformBin); err != nil {
		a.logger.Error("Terraform Destroy 执行失败", zap.Error(err))
		return fmt.Errorf("terraform Destroy 失败: %w", err)
	}

	// 从数据库中删除资源记录
	if err := a.ecsDao.Delete(ctx, existingResource.ID); err != nil {
		a.logger.Error("删除资源信息到数据库失败", zap.Error(err))
		return fmt.Errorf("删除资源信息到数据库失败: %w", err)
	}

	a.logger.Info("阿里云资源删除成功", zap.Int("resource_id", existingResource.ID))
	return nil
}

// generateHash 生成资源的哈希值
func generateHash(config model.TerraformConfig) string {
	h := sha256.New()
	h.Write([]byte(config.Name))
	h.Write([]byte(config.Region))
	h.Write([]byte(config.Env))
	h.Write([]byte(config.PayType))
	return fmt.Sprintf("%x", h.Sum(nil))
}
