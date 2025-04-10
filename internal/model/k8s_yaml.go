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

package model

// K8sYamlTask Kubernetes YAML 任务的配置
type K8sYamlTask struct {
	Model
	Name        string     `json:"name" gorm:"type:varchar(255);comment:YAML 任务名称"`                 // YAML 任务名称
	UserID      int        `json:"user_id" gorm:"comment:创建者用户ID"`                                  // 创建者用户ID
	TemplateID  int        `json:"template_id" gorm:"comment:关联的模板ID"`                              // 关联的模板ID
	ClusterId   int        `json:"cluster_id,omitempty" gorm:"comment:集群名称"`                        // 集群名称
	Variables   StringList `json:"variables,omitempty" gorm:"type:text;comment:yaml 变量，格式 k=v,k=v"` // YAML 变量
	Status      string     `json:"status,omitempty" gorm:"comment:当前状态"`                            // 当前状态
	ApplyResult string     `json:"apply_result,omitempty" gorm:"comment:apply 后的返回数据"`              // apply 结果
}

// K8sYamlTemplate Kubernetes YAML 模板的配置
type K8sYamlTemplate struct {
	Model
	Name      string `json:"name" binding:"required,min=1,max=50" gorm:"size:100;comment:模板名称"` // 模板名称
	UserID    int    `json:"user_id" gorm:"comment:创建者用户ID"`                                    // 创建者用户ID
	Content   string `json:"content,omitempty" gorm:"type:text;comment:yaml 模板内容"`              // YAML 模板内容
	ClusterId int    `json:"cluster_id,omitempty" gorm:"comment:对应集群id"`
}