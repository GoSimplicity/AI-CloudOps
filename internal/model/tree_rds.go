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

// ResourceRds 数据库资源
type ResourceRds struct {
	ResourceBase
	Engine            string `json:"engine" gorm:"type:varchar(50);comment:数据库引擎类型,如mysql,postgresql"`
	EngineVersion     string `json:"engineVersion" gorm:"type:varchar(50);comment:数据库版本,如8.0,5.7"`
	DBInstanceClass   string `json:"dbInstanceClass" gorm:"type:varchar(100);comment:实例规格"`
	DBInstanceType    string `json:"dbInstanceType" gorm:"type:varchar(50);comment:实例类型,如Primary,Readonly"`
	DBInstanceNetType string `json:"dbInstanceNetType" gorm:"type:varchar(50);comment:实例网络类型"`
	MasterInstanceId  string `json:"masterInstanceId" gorm:"type:varchar(100);comment:主实例ID"`
	ReplicateId       string `json:"replicateId" gorm:"type:varchar(100);comment:复制实例ID"`
	DBStatus          string `json:"dbStatus" gorm:"type:varchar(50);comment:数据库状态"`
	Port              int    `json:"port" gorm:"comment:数据库端口;default:3306"`
	ConnectionString  string `json:"connectionString" gorm:"type:varchar(255);comment:连接字符串"`
	// 多对多关系
	RdsTreeNodes []*TreeNode `json:"rdsTreeNodes" gorm:"many2many:resource_rds_tree_nodes;comment:关联服务树节点"`
}

// RdsCreationParams RDS创建参数
type RdsCreationParams struct {
	Provider          CloudProvider     `json:"provider" binding:"required"`
	Region            string            `json:"region" binding:"required"`
	ZoneId            string            `json:"zoneId" binding:"required"`
	Engine            string            `json:"engine" binding:"required"`
	EngineVersion     string            `json:"engineVersion" binding:"required"`
	DBInstanceClass   string            `json:"dbInstanceClass" binding:"required"`
	VpcId             string            `json:"vpcId" binding:"required"`
	DBInstanceNetType string            `json:"dbInstanceNetType" binding:"required"`
	PayType           string            `json:"payType" binding:"required"`
	TreeNodeId        uint              `json:"treeNodeId" binding:"required"`
	Description       string            `json:"description"`
	Tags              map[string]string `json:"tags"`
}

// ListRdsResourcesReq RDS资源列表查询参数
type ListRdsResourcesReq struct {
	PageNumber int           `form:"pageNumber" json:"pageNumber"`
	PageSize   int           `form:"pageSize" json:"pageSize"`
	Provider   CloudProvider `form:"provider" json:"provider"`
	Region     string        `form:"region" json:"region"`
}

// ResourceRDSResp RDS资源响应
type ResourceRDSResp struct {
	ResourceRds
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type GetRdsDetailReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	RdsId    string        `json:"rdsId" binding:"required"`
}

type StartRdsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	RdsId    string        `json:"rdsId" binding:"required"`
}

type StopRdsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	RdsId    string        `json:"rdsId" binding:"required"`
}

type RestartRdsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	RdsId    string        `json:"rdsId" binding:"required"`
}

type DeleteRdsReq struct {
	Provider CloudProvider `json:"provider" binding:"required"`
	Region   string        `json:"region" binding:"required"`
	RdsId    string        `json:"rdsId" binding:"required"`
}
