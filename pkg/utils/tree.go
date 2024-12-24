/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/terraform-exec/tfexec"
	"go.uber.org/zap"
)

// 升级器
var UpGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RenderTerraformTemplate 渲染 Terraform 模板并写入指定目录的 main.tf 文件
func RenderTerraformTemplate(config model.TerraformConfig, workDir string, terraformTemplate string, key string, secret string,
	vpc model.VPCConfig, instance model.InstanceConfig, security model.SecurityConfig) error {
	// 创建新的结构体，将所有字段传递给模板
	data := struct {
		Region   string
		Name     string
		VPC      model.VPCConfig
		Instance model.InstanceConfig
		Security model.SecurityConfig
		Key      string
		Secret   string
	}{
		Region:   config.Region,
		Name:     config.Name,
		VPC:      vpc,
		Instance: instance,
		Security: security,
		Key:      key,
		Secret:   secret,
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

// SetupTerraform 初始化并计划 Terraform 在指定目录
func SetupTerraform(ctx context.Context, workDir string, terraformBin string) (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(workDir, terraformBin)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Terraform: %w", err)
	}

	// 初始化 Terraform
	if err := tf.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return nil, fmt.Errorf("terraform init failed: %w", err)
	}

	// 执行 Terraform Plan
	if _, err := tf.Plan(ctx, tfexec.Out("plan.out")); err != nil {
		return nil, fmt.Errorf("terraform plan failed: %w", err)
	}

	return tf, nil
}

// ApplyTerraform 执行 Terraform Apply 在指定 Terraform 实例
func ApplyTerraform(ctx context.Context, tf *tfexec.Terraform) error {
	if err := tf.Apply(ctx, tfexec.Parallelism(10)); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	return nil
}

// DestroyTerraform 执行 Terraform Destroy 在指定目录
func DestroyTerraform(ctx context.Context, workDir string, terraformBin string) error {
	tf, err := tfexec.NewTerraform(workDir, terraformBin)
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
	if err := tf.Destroy(ctx, tfexec.Parallelism(10)); err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// EnsureDir 确保目录存在，不存在则创建
func EnsureDir(dirPath string, logger *zap.Logger) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			logger.Error("创建 Terraform 目录失败", zap.Error(err))
			return fmt.Errorf("创建 Terraform 目录失败: %w", err)
		}
		logger.Info("创建 Terraform 目录", zap.String("path", dirPath))
	}
	return nil
}

// ParseConfigs 解析 Terraform 配置中的各部分
func ParseConfigs(config model.TerraformConfig, logger *zap.Logger) (model.InstanceConfig, model.VPCConfig, model.SecurityConfig, error) {
	var instanceConfig model.InstanceConfig
	if err := json.Unmarshal(config.Instance, &instanceConfig); err != nil {
		logger.Error("解析 Instance 配置失败", zap.Error(err))
		return model.InstanceConfig{}, model.VPCConfig{}, model.SecurityConfig{}, fmt.Errorf("解析 Instance 配置失败: %w", err)
	}

	var vpcConfig model.VPCConfig
	if err := json.Unmarshal(config.VPC, &vpcConfig); err != nil {
		logger.Error("解析 VPC 配置失败", zap.Error(err))
		return model.InstanceConfig{}, model.VPCConfig{}, model.SecurityConfig{}, fmt.Errorf("解析 VPC 配置失败: %w", err)
	}

	var securityConfig model.SecurityConfig
	if err := json.Unmarshal(config.Security, &securityConfig); err != nil {
		logger.Error("解析 Security 配置失败", zap.Error(err))
		return model.InstanceConfig{}, model.VPCConfig{}, model.SecurityConfig{}, fmt.Errorf("解析 Security 配置失败: %w", err)
	}

	return instanceConfig, vpcConfig, securityConfig, nil
}

// GetTerraformState 获取并解析 Terraform 状态
func GetTerraformState(ctx context.Context, tf *tfexec.Terraform, logger *zap.Logger) (*struct {
	Outputs map[string]struct {
		Value       interface{} `json:"value"`
		Description string      `json:"description"`
	} `json:"outputs"`
}, error) {
	stateJSON, err := tf.StatePull(ctx)
	if err != nil {
		logger.Error("获取 Terraform 状态失败", zap.Error(err))
		return nil, fmt.Errorf("获取 Terraform 状态失败: %w", err)
	}

	var state struct {
		Outputs map[string]struct {
			Value       interface{} `json:"value"`
			Description string      `json:"description"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		logger.Error("解析 Terraform 状态失败", zap.Error(err))
		return nil, fmt.Errorf("解析 Terraform 状态失败: %w", err)
	}

	return &state, nil
}

// ExtractIPs 从 Terraform 状态中提取公网和私网 IP
func ExtractIPs(state *struct {
	Outputs map[string]struct {
		Value       interface{} `json:"value"`
		Description string      `json:"description"`
	} `json:"outputs"`
}, logger *zap.Logger) (string, string, error) {
	publicIP, ok := state.Outputs["public_ip"]
	if !ok {
		logger.Error("Terraform 状态中缺少 'public_ip' 输出")
		return "", "", fmt.Errorf("terraform 输出缺少 'public_ip'")
	}

	privateIP, ok := state.Outputs["private_ip"]
	if !ok {
		logger.Error("Terraform 状态中缺少 'private_ip' 输出")
		return "", "", fmt.Errorf("terraform 输出缺少 'private_ip'")
	}

	publicIPStr, ok := publicIP.Value.(string)
	if !ok {
		logger.Error("'public_ip' 输出值类型错误")
		return "", "", fmt.Errorf("'public_ip' 输出不是字符串")
	}

	privateIPStr, ok := privateIP.Value.(string)
	if !ok {
		logger.Error("'private_ip' 输出值类型错误")
		return "", "", fmt.Errorf("'private_ip' 输出不是字符串")
	}

	return publicIPStr, privateIPStr, nil
}
