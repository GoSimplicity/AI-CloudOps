<template>
  <div class="resource-management">
    <a-page-header
      title="资源管理平台"
      sub-title="多云资源统一管理"
      class="page-header"
    >
      <template #extra>
        <a-button type="primary" @click="handleSyncResources">
          <sync-outlined /> 同步资源
        </a-button>
      </template>
    </a-page-header>

    <a-card class="filter-card">
      <a-form layout="inline" :model="filterForm">
        <a-form-item label="云厂商">
          <a-select
            v-model:value="filterForm.provider"
            style="width: 120px"
            placeholder="选择厂商"
            allow-clear
          >
            <a-select-option v-for="provider in cloudProviders" :key="provider.value" :value="provider.value">
              {{ provider.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="地区">
          <a-select
            v-model:value="filterForm.region"
            style="width: 150px"
            placeholder="选择地区"
            allow-clear
          >
            <a-select-option v-for="region in regions" :key="region.value" :value="region.value">
              {{ region.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="资源名称">
          <a-input
            v-model:value="filterForm.name"
            placeholder="输入资源名称"
            allow-clear
          />
        </a-form-item>
        <a-form-item label="状态">
          <a-select
            v-model:value="filterForm.status"
            style="width: 120px"
            placeholder="选择状态"
            allow-clear
          >
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Stopped">已停止</a-select-option>
            <a-select-option value="Starting">启动中</a-select-option>
            <a-select-option value="Stopping">停止中</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <search-outlined /> 搜索
          </a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">
            重置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-tabs v-model:activeKey="activeTab" class="resource-tabs" @change="handleTabChange">
      <a-tab-pane key="ecs" tab="云服务器 ECS">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('ecs')">
              <plus-outlined /> 创建实例
            </a-button>
          </template>
          <a-table
            :columns="ecsColumns"
            :data-source="ecsData"
            :loading="loading"
            :pagination="pagination"
            @change="handleTableChange"
            row-key="instance_id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'ipAddr'">
                <div>
                  <div>内网: {{ record.private_ip_address?.join(', ') || '-' }}</div>
                  <div v-if="record.public_ip_address && record.public_ip_address.length > 0">
                    公网: {{ record.public_ip_address?.join(', ') }}
                  </div>
                </div>
              </template>
              <template v-if="column.key === 'region'">
                {{ record.region_id }}/{{ record.zone_id }}
              </template>
              <template v-if="column.key === 'action'">
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item key="detail" @click="handleViewDetail('ecs', record)">
                        <info-circle-outlined /> 详情
                      </a-menu-item>
                      <a-menu-item key="start" @click="handleEcsAction('start', record)" v-if="record.status === 'Stopped'">
                        <play-circle-outlined /> 启动
                      </a-menu-item>
                      <a-menu-item key="stop" @click="handleEcsAction('stop', record)" v-if="record.status === 'Running'">
                        <pause-circle-outlined /> 停止
                      </a-menu-item>
                      <a-menu-item key="restart" @click="handleEcsAction('restart', record)" v-if="record.status === 'Running'">
                        <reload-outlined /> 重启
                      </a-menu-item>
                      <a-menu-item key="delete" @click="handleDeleteResource('ecs', record)">
                        <delete-outlined /> 删除
                      </a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="link">
                    操作 <down-outlined />
                  </a-button>
                </a-dropdown>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="vpc" tab="专有网络 VPC">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('vpc')">
              <plus-outlined /> 创建VPC
            </a-button>
          </template>
          <a-table
            :columns="vpcColumns"
            :data-source="vpcData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('vpc', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('vpc', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="sg" tab="安全组">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('sg')">
              <plus-outlined /> 创建安全组
            </a-button>
          </template>
          <a-table
            :columns="sgColumns"
            :data-source="sgData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('sg', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('sg', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="elb" tab="负载均衡 ELB">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('elb')">
              <plus-outlined /> 创建负载均衡
            </a-button>
          </template>
          <a-table
            :columns="elbColumns"
            :data-source="elbData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" @click="handleViewDetail('elb', record)">详情</a-button>
                <a-button type="link" @click="handleDeleteResource('elb', record)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
      
      <a-tab-pane key="rds" tab="云数据库 RDS">
        <a-card class="resource-card">
          <template #extra>
            <a-button type="primary" @click="showCreateModal('rds')">
              <plus-outlined /> 创建数据库实例
            </a-button>
          </template>
          <a-table
            :columns="rdsColumns"
            :data-source="rdsData"
            :loading="loading"
            :pagination="pagination"
            row-key="id"
            @change="handleTableChange"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item key="detail" @click="handleViewDetail('rds', record)">
                        <info-circle-outlined /> 详情
                      </a-menu-item>
                      <a-menu-item key="start" @click="handleRdsAction('start', record)" v-if="record.status === 'stopped'">
                        <play-circle-outlined /> 启动
                      </a-menu-item>
                      <a-menu-item key="stop" @click="handleRdsAction('stop', record)" v-if="record.status === 'running'">
                        <pause-circle-outlined /> 停止
                      </a-menu-item>
                      <a-menu-item key="restart" @click="handleRdsAction('restart', record)" v-if="record.status === 'running'">
                        <reload-outlined /> 重启
                      </a-menu-item>
                      <a-menu-item key="delete" @click="handleDeleteResource('rds', record)">
                        <delete-outlined /> 删除
                      </a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="link">
                    操作 <down-outlined />
                  </a-button>
                </a-dropdown>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- ECS创建对话框 -->
    <a-modal
      v-model:visible="modals.ecs"
      title="创建云服务器实例"
      width="800px"
      :footer="null"
      :destroyOnClose="true"
    >
      <a-steps :current="currentStep" size="small" class="create-steps">
        <a-step title="基础配置" />
        <a-step title="网络配置" />
        <a-step title="系统配置" />
        <a-step title="确认信息" />
      </a-steps>

      <a-form :model="createForm" layout="vertical" ref="createFormRef" class="create-form">
        <!-- 步骤 1: 基础配置 -->
        <div v-if="currentStep === 0">
          <a-form-item label="云服务商" name="provider" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.provider" placeholder="选择云服务商" @change="handleProviderChange">
              <a-select-option value="aliyun">阿里云</a-select-option>
              <a-select-option value="aws">AWS</a-select-option>
              <a-select-option value="tencent">腾讯云</a-select-option>
              <a-select-option value="huawei">华为云</a-select-option>
              <a-select-option value="azure">Azure</a-select-option>
              <a-select-option value="gcp">Google Cloud</a-select-option>
              <a-select-option value="local">本地环境</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="付费类型" name="payType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.payType" placeholder="选择付费类型" @change="handlePayTypeChange"
              :disabled="!createForm.provider">
              <a-select-option value="PostPaid">按量付费</a-select-option>
              <a-select-option value="PrePaid">包年包月</a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="地域" name="region" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.region" placeholder="选择地域" @change="handleRegionChange"
              :disabled="!createForm.payType">
              <a-select-option v-for="data in regionOptions" :key="data.region" :value="data.region">
                {{ data.region }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="可用区" name="zoneId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.zoneId" placeholder="选择可用区" @change="handleZoneChange"
              :disabled="!createForm.region">
              <a-select-option v-for="zone in zoneOptions" :key="zone.zone" :value="zone.zone">
                {{ zone.zone }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="实例规格" name="instanceType" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.instanceType" placeholder="选择实例规格" @change="handleInstanceTypeChange"
              :disabled="!createForm.zoneId" show-search :filter-option="filterInstanceType" :options="instanceTypeOptions.map(type => ({
                value: type.instanceType,
                label: `${type.instanceType} (${type.cpu}核${type.memory}GB)`
              }))">
            </a-select>
          </a-form-item>

          <a-form-item label="镜像" name="imageId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.imageId" placeholder="选择镜像" @change="handleImageIdChange"
              :disabled="!createForm.instanceType" show-search :filter-option="filterImage" :options="imageOptions.map(image => ({
                value: image.imageId,
                label: `${image.osName} (${image.osType} - ${image.architecture})`
              }))" :virtual="false" :dropdown-style="{ maxHeight: '400px', overflow: 'auto' }">
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 2: 网络配置 -->
        <div v-if="currentStep === 1">
          <a-form-item label="实例数量" name="amount" :rules="[{ required: true }]">
            <a-input-number v-model:value="createForm.amount" :min="1" :max="100" style="width: 100%" />
          </a-form-item>

          <a-form-item label="VPC" name="vpcId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vpcId" placeholder="选择VPC" @change="handleVpcChange"
              :loading="vpcLoading">
              <a-select-option v-for="vpc in vpcOptions" :key="vpc.vpcId" :value="vpc.vpcId">
                {{ vpc.vpcName }} ({{ vpc.cidrBlock }})
              </a-select-option>
              <a-empty v-if="vpcOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无VPC资源" />
            </a-select>
          </a-form-item>

          <a-form-item label="交换机" name="vSwitchId" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.vSwitchId" placeholder="选择交换机" :loading="vSwitchLoading"
              :disabled="!createForm.vpcId">
              <a-select-option v-for="vSwitch in vSwitchOptions" :key="vSwitch.vSwitchId" :value="vSwitch.vSwitchId">
                {{ vSwitch.vSwitchName }} ({{ vSwitch.cidrBlock }})
              </a-select-option>
              <a-empty v-if="vSwitchOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无可用交换机" />
            </a-select>
          </a-form-item>

          <a-form-item label="安全组" name="securityGroupIds" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.securityGroupIds" placeholder="选择安全组" mode="multiple"
              :loading="securityGroupLoading" :disabled="!createForm.vpcId">
              <a-select-option v-for="sg in securityGroupOptions" :key="sg.securityGroupId" :value="sg.securityGroupId">
                {{ sg.securityGroupName }} ({{ sg.description || '无描述' }})
              </a-select-option>
              <a-empty v-if="securityGroupOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
                description="暂无可用安全组" />
            </a-select>
          </a-form-item>
        </div>

        <!-- 步骤 3: 系统配置 -->
        <div v-if="currentStep === 2">
          <a-form-item label="实例名称" name="instanceName" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.instanceName" placeholder="实例名称，如web-server-01" />
          </a-form-item>

          <a-form-item label="主机名" name="hostname" :rules="[{ required: true }]">
            <a-input v-model:value="createForm.hostname" placeholder="主机名，如cloudops" />
          </a-form-item>

          <a-form-item label="登录密码" name="password" :rules="[{ required: true }]">
            <a-input-password v-model:value="createForm.password" placeholder="请输入登录密码" />
          </a-form-item>

          <a-form-item label="实例描述" name="description">
            <a-textarea v-model:value="createForm.description" placeholder="实例描述" :rows="2" />
          </a-form-item>

          <a-form-item label="系统盘类型" name="systemDiskCategory" :rules="[{ required: true }]">
            <a-select v-model:value="createForm.systemDiskCategory" placeholder="选择系统盘类型"
              @change="handleSystemDiskCategoryChange">
              <a-select-option v-for="disk in systemDiskOptions" :key="disk.systemDiskCategory"
                :value="disk.systemDiskCategory">
                {{ disk.systemDiskCategory }}
              </a-select-option>
              <a-empty v-if="systemDiskOptions.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
                description="暂无可用系统盘类型" />
            </a-select>
          </a-form-item>

          <a-form-item label="系统盘大小 (GB)" name="systemDiskSize" :rules="[{ required: true }]">
            <a-slider v-model:value="createForm.systemDiskSize" :min="20" :max="500" :step="10"
              :marks="{ 20: '20G', 100: '100G', 200: '200G', 500: '500G' }" />
          </a-form-item>

          <a-form-item label="数据盘类型" name="dataDiskCategory">
            <a-select v-model:value="createForm.dataDiskCategory" placeholder="选择数据盘类型"
              @change="handleDataDiskCategoryChange" :disabled="!createForm.systemDiskCategory">
              <a-select-option v-for="disk in dataDiskOptions" :key="disk.dataDiskCategory"
                :value="disk.dataDiskCategory">
                {{ disk.dataDiskCategory }}
              </a-select-option>
            </a-select>
          </a-form-item>

          <a-form-item label="数据盘大小 (GB)" name="dataDiskSize">
            <a-slider v-model:value="createForm.dataDiskSize" :min="20" :max="2000" :step="10"
              :marks="{ 20: '20G', 100: '100G', 500: '500G', 2000: '2TB' }" :disabled="!createForm.dataDiskCategory" />
          </a-form-item>

          <a-form-item label="标签" name="tags">
            <div class="tag-input-container">
              <div v-for="(tag, index) in tagsArray" :key="index" class="tag-item">
                <a-tag closable @close="removeTag(index)">{{ tag }}</a-tag>
              </div>
              <a-input v-model:value="tagInputValue" placeholder="输入标签，格式为key=value，按回车添加" @pressEnter="addTag"
                style="width: 200px" />
            </div>
          </a-form-item>
        </div>

        <!-- 步骤 4: 确认信息 -->
        <div v-if="currentStep === 3" class="confirmation-step">
          <a-descriptions bordered :column="1" size="small">
            <a-descriptions-item label="云服务商">{{ getProviderName(createForm.provider) }}</a-descriptions-item>
            <a-descriptions-item label="付费类型">{{ getPayTypeName(createForm.payType) }}</a-descriptions-item>
            <a-descriptions-item label="地域">
              {{ getRegionById(createForm.region)?.region || createForm.region }}
            </a-descriptions-item>
            <a-descriptions-item label="可用区">
              {{ getZoneById(createForm.zoneId)?.zone || createForm.zoneId }}
            </a-descriptions-item>
            <a-descriptions-item label="实例规格">
              {{ getInstanceTypeById(createForm.instanceType)?.instanceType || createForm.instanceType }}
            </a-descriptions-item>
            <a-descriptions-item label="镜像">{{ createForm.imageId }}</a-descriptions-item>
            <a-descriptions-item label="实例数量">{{ createForm.amount }}</a-descriptions-item>
            <a-descriptions-item label="VPC">
              {{ getVpcById(createForm.vpcId)?.vpcName || createForm.vpcId }}
            </a-descriptions-item>
            <a-descriptions-item label="交换机">
              {{ getVSwitchById(createForm.vSwitchId)?.vSwitchName || createForm.vSwitchId }}
            </a-descriptions-item>
            <a-descriptions-item label="安全组">
              <template v-if="createForm.securityGroupIds && createForm.securityGroupIds.length > 0">
                <a-tag v-for="(sgId, idx) in createForm.securityGroupIds" :key="idx" color="blue">
                  {{ getSecurityGroupById(sgId)?.securityGroupName || sgId }}
                </a-tag>
              </template>
              <template v-else>
                <span>未选择安全组</span>
              </template>
            </a-descriptions-item>
            <a-descriptions-item label="实例名称">{{ createForm.instanceName }}</a-descriptions-item>
            <a-descriptions-item label="系统盘">
              {{ getSystemDiskById(createForm.systemDiskCategory)?.systemDiskCategory ||
                createForm.systemDiskCategory }} {{ createForm.systemDiskSize }}GB
            </a-descriptions-item>
            <a-descriptions-item label="数据盘" v-if="createForm.dataDiskCategory">
              {{ getDataDiskById(createForm.dataDiskCategory)?.dataDiskCategory ||
                createForm.dataDiskCategory }} {{ createForm.dataDiskSize }}GB
            </a-descriptions-item>
            <a-descriptions-item label="标签" v-if="tagsArray.length > 0">
              <a-tag v-for="(tag, index) in tagsArray" :key="index" color="blue">{{ tag }}</a-tag>
            </a-descriptions-item>
          </a-descriptions>

          <a-alert type="info" showIcon style="margin-top: 20px;">
            <template #message>
              <span>创建 ECS 服务器后，服务器将立即启动，实例费用将根据付费类型收取。</span>
            </template>
          </a-alert>
        </div>

        <div class="steps-action">
          <a-button v-if="currentStep > 0" style="margin-right: 8px" @click="prevStep">
            上一步
          </a-button>
          <a-button v-if="currentStep < 3" type="primary" @click="nextStep">
            下一步
          </a-button>
          <a-button v-if="currentStep === 3" type="primary" @click="handleCreateSubmit" :loading="createLoading">
            创建实例
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <!-- 资源详情对话框 -->
    <a-drawer
      v-model:visible="detailVisible"
      :title="`${resourceDetailTitle}详情`"
      width="600"
      :destroyOnClose="true"
      class="detail-drawer"
    >
      <a-skeleton :loading="detailLoading" active>
        <template v-if="resourceType === 'ecs'">
          <a-descriptions bordered :column="1">
            <a-descriptions-item label="实例 ID">{{ resourceDetail.instance_id }}</a-descriptions-item>
            <a-descriptions-item label="实例名称">{{ resourceDetail.instance_name }}</a-descriptions-item>
            <a-descriptions-item label="实例状态">
              <a-tag :color="getStatusColor(resourceDetail.status)">
                {{ getStatusText(resourceDetail.status) }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="区域">{{ resourceDetail.region_id }}</a-descriptions-item>
            <a-descriptions-item label="可用区">{{ resourceDetail.zone_id }}</a-descriptions-item>
            <a-descriptions-item label="实例规格">{{ resourceDetail.instanceType }}</a-descriptions-item>
            <a-descriptions-item label="CPU">{{ resourceDetail.cpu }} 核</a-descriptions-item>
            <a-descriptions-item label="内存">{{ resourceDetail.memory }} GB</a-descriptions-item>
            <a-descriptions-item label="操作系统">{{ resourceDetail.osName }}</a-descriptions-item>
            <a-descriptions-item label="IP 地址">
              <div>
                <div>内网: {{ resourceDetail.private_ip_address?.join(', ') || '-' }}</div>
                <div v-if="resourceDetail.public_ip_address && resourceDetail.public_ip_address.length > 0">
                  公网: {{ resourceDetail.public_ip_address?.join(', ') }}
                </div>
              </div>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ resourceDetail.creation_time }}</a-descriptions-item>
            <a-descriptions-item label="付费方式">
              {{ getPayTypeName(resourceDetail.instance_charge_type) }}
            </a-descriptions-item>
          </a-descriptions>

          <a-divider orientation="left">磁盘信息</a-divider>
          <a-table :dataSource="disks" :columns="diskColumns" :pagination="false" size="small"
            :row-key="(record: any) => record.diskId"></a-table>

          <a-divider orientation="left">标签</a-divider>
          <div class="tag-list">
            <a-tag v-for="(tag, index) in resourceDetail.tags" :key="index" color="blue">{{ tag }}</a-tag>
            <a-empty v-if="!resourceDetail.tags || resourceDetail.tags.length === 0" :image="Empty.PRESENTED_IMAGE_SIMPLE"
              description="暂无标签" />
          </div>

          <div class="drawer-actions">
            <a-button-group>
              <a-button type="primary" :disabled="resourceDetail.status === 'Running'" @click="handleEcsAction('start', resourceDetail)">
                <play-circle-outlined /> 启动
              </a-button>
              <a-button :disabled="resourceDetail.status !== 'Running'" @click="handleEcsAction('stop', resourceDetail)">
                <pause-circle-outlined /> 停止
              </a-button>
              <a-button :disabled="resourceDetail.status !== 'Running'" @click="handleEcsAction('restart', resourceDetail)">
                <reload-outlined /> 重启
              </a-button>
            </a-button-group>
            <a-button danger @click="handleDeleteResource('ecs', resourceDetail)">
              <delete-outlined /> 删除
            </a-button>
          </div>
        </template>

        <template v-else>
          <a-descriptions bordered :column="2">
            <a-descriptions-item label="名称" span="2">{{ resourceDetail?.instanceName }}</a-descriptions-item>
            <a-descriptions-item label="ID">{{ resourceDetail?.instanceId || resourceDetail?.id }}</a-descriptions-item>
            <a-descriptions-item label="状态" v-if="resourceDetail?.status">
              <a-tag :color="getStatusColor(resourceDetail?.status)">
                {{ getStatusText(resourceDetail?.status) }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="云厂商">{{ getProviderName(resourceDetail?.provider) }}</a-descriptions-item>
            <a-descriptions-item label="地区">{{ resourceDetail?.regionId }}</a-descriptions-item>
            <!-- 其他资源类型的特定字段 -->
            <template v-if="resourceType === 'vpc'">
              <a-descriptions-item label="CIDR块" span="2">{{ resourceDetail?.cidrBlock }}</a-descriptions-item>
            </template>
            <a-descriptions-item label="创建时间" span="2">{{ resourceDetail?.creationTime }}</a-descriptions-item>
            <a-descriptions-item label="描述" span="2">{{ resourceDetail?.description }}</a-descriptions-item>
            <a-descriptions-item label="标签" span="2">
              <template v-if="resourceDetail?.tags && resourceDetail.tags.length > 0">
                <a-tag v-for="(tag, index) in resourceDetail?.tags" :key="index">{{ tag }}</a-tag>
              </template>
              <template v-else>
                <span>无标签</span>
              </template>
            </a-descriptions-item>
          </a-descriptions>
        </template>
      </a-skeleton>
      
      <div style="margin-top: 24px; text-align: right;">
        <a-button @click="detailVisible = false">关闭</a-button>
      </div>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue';
import { message, Modal, Empty } from 'ant-design-vue';
import {
  SyncOutlined,
  PlusOutlined,
  SearchOutlined,
  InfoCircleOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined,
  DeleteOutlined,
  DownOutlined,
} from '@ant-design/icons-vue';

import {
  getEcsResourceList,
  getEcsResourceDetail,
  createEcsResource,
  startEcsResource,
  stopEcsResource,
  restartEcsResource,
  deleteEcsResource,
  getInstanceOptions,
  getVpcResourceList,
  listSecurityGroups,
} from '#/api/core/tree';

// 接口定义
interface ResourceEcs {
  instance_id: string;
  instance_name: string;
  cloud_provider: string;
  region_id: string;
  zone_id: string;
  status: string;
  cpu: number;
  memory: number;
  instanceType: string;
  osName: string;
  private_ip_address?: string[];
  public_ip_address?: string[];
  creation_time: string;
  instance_charge_type: string;
  diskIds?: string[];
  tags?: string[];
}

interface Disk {
  diskId: string;
  diskName: string;
  type: string;
  category: string;
  size: number;
}

interface VpcOption {
  vpcId: string;
  vpcName: string;
  cidrBlock: string;
  description?: string;
}

interface VSwitchOption {
  vSwitchId: string;
  vSwitchName: string;
  cidrBlock: string;
  zoneId: string;
  vpcId: string;
}

interface SecurityGroupOption {
  securityGroupId: string;
  securityGroupName: string;
  description?: string;
  vpcId: string;
}

interface ListInstanceOptionsResp {
  region: string;
  zone: string;
  instanceType: string;
  cpu: number;
  memory: number;
  imageId: string;
  osName: string;
  osType: string;
  architecture: string;
  systemDiskCategory: string;
  dataDiskCategory: string;
  payType: string;
  valid: boolean;
}

// 云厂商列表
const cloudProviders = [
  { label: '阿里云', value: 'aliyun' },
  { label: '腾讯云', value: 'tencent' },
  { label: '华为云', value: 'huawei' },
  { label: 'AWS', value: 'aws' }
];

// 区域列表
const regions = [
  { label: '华北1（青岛）', value: 'cn-qingdao' },
  { label: '华北2（北京）', value: 'cn-beijing' },
  { label: '华东1（杭州）', value: 'cn-hangzhou' },
  { label: '华东2（上海）', value: 'cn-shanghai' },
  { label: '华南1（深圳）', value: 'cn-shenzhen' }
];

// 活动标签页
const activeTab = ref('ecs');

// 加载状态
const loading = ref(false);
const detailLoading = ref(false);
const createLoading = ref(false);
const vpcLoading = ref(false);
const vSwitchLoading = ref(false);
const securityGroupLoading = ref(false);

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 过滤条件
const filterForm = reactive({
  provider: 'aliyun',
  region: 'cn-hangzhou',
  name: '',
  status: undefined,
  pageNumber: 1,
  pageSize: 10
});

// 模态框状态
const modals = reactive({
  ecs: false,
  vpc: false,
  sg: false,
  elb: false,
  rds: false
});

// 详情抽屉
const detailVisible = ref(false);
const resourceType = ref('');
const resourceDetail = ref<any>({});
const disks = ref<Disk[]>([]);
const resourceDetailTitle = computed(() => {
  const typeMap: Record<string, string> = {
    'ecs': '云服务器',
    'vpc': '专有网络',
    'sg': '安全组',
    'elb': '负载均衡',
    'rds': '云数据库'
  };
  return typeMap[resourceType.value] || '资源';
});

// ECS相关状态
const ecsData = ref<ResourceEcs[]>([]);
const currentStep = ref(0);
const createFormRef = ref(null);
const tagsArray = ref<string[]>([]);
const tagInputValue = ref('');
const regionOptions = ref<ListInstanceOptionsResp[]>([]);
const zoneOptions = ref<ListInstanceOptionsResp[]>([]);
const instanceTypeOptions = ref<ListInstanceOptionsResp[]>([]);
const imageOptions = ref<ListInstanceOptionsResp[]>([]);
const systemDiskOptions = ref<ListInstanceOptionsResp[]>([]);
const dataDiskOptions = ref<ListInstanceOptionsResp[]>([]);
const vpcOptions = ref<VpcOption[]>([]);
const vSwitchOptions = ref<VSwitchOption[]>([]);
const securityGroupOptions = ref<SecurityGroupOption[]>([]);

// 创建表单数据
const createForm = reactive({
  provider: 'aliyun',
  region: '',
  imageId: '',
  instanceType: '',
  amount: 1,
  zoneId: '',
  vpcId: '',
  vSwitchId: '',
  securityGroupIds: [] as string[],
  hostname: '',
  password: '',
  instanceName: '',
  payType: '',
  instanceChargeType: '',
  spotStrategy: 'NoSpot',
  description: '',
  systemDiskCategory: '',
  systemDiskSize: 40,
  dataDiskCategory: '',
  dataDiskSize: 100,
  dryRun: false,
  tags: {} as Record<string, string>,
  periodUnit: 'Month',
  period: 1,
  autoRenew: false,
  spotDuration: 1,
});

// ECS表格列定义
const ecsColumns = [
  { title: '实例名称/ID', dataIndex: 'instance_name', key: 'instanceName', width: 180, ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType', width: 130, ellipsis: true },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr', width: 160, ellipsis: true },
  { title: '地区/可用区', dataIndex: 'region', key: 'region', width: 160, ellipsis: true },
  { title: '创建时间', dataIndex: 'creation_time', key: 'creation_time', width: 170 },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// VPC表格列定义
const vpcColumns = [
  { title: 'VPC名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: 'VPC ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: 'CIDR块', dataIndex: 'cidrBlock', key: 'cidrBlock' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: '云厂商', dataIndex: 'provider', key: 'provider' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 安全组表格列定义
const sgColumns = [
  { title: '安全组名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '安全组ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '云厂商', dataIndex: 'provider', key: 'provider' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: 'VPC ID', dataIndex: 'vpcId', key: 'vpcId' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// ELB表格列定义
const elbColumns = [
  { title: '负载均衡名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '负载均衡ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '地址类型', dataIndex: 'addressType', key: 'addressType' },
  { title: 'IP地址', dataIndex: 'address', key: 'address' },
  { title: '地区', dataIndex: 'regionId', key: 'regionId' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// RDS表格列定义
const rdsColumns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName' },
  { title: '实例ID', dataIndex: 'instanceId', key: 'instanceId' },
  { title: '状态', dataIndex: 'status', key: 'status' },
  { title: '数据库类型', dataIndex: 'dbType', key: 'dbType' },
  { title: '规格', dataIndex: 'instanceType', key: 'instanceType' },
  { title: '地区/可用区', dataIndex: 'regionAndZone', key: 'regionAndZone' },
  { title: '创建时间', dataIndex: 'creationTime', key: 'creationTime' },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// 磁盘表格列定义
const diskColumns = [
  { title: '磁盘名称', dataIndex: 'diskName', key: 'diskName' },
  { title: '磁盘ID', dataIndex: 'diskId', key: 'diskId' },
  { title: '类型', dataIndex: 'type', key: 'type' },
  { title: '类别', dataIndex: 'category', key: 'category' },
  { title: '大小(GB)', dataIndex: 'size', key: 'size' },
];

// 模拟数据 - VPC
const vpcData = ref([
  {
    id: 1,
    instanceName: 'vpc-default',
    instanceId: 'vpc-2zeisljxz9bmxhj2qyyy',
    cidrBlock: '172.16.0.0/16',
    regionId: 'cn-hangzhou',
    provider: 'aliyun',
    status: 'Available',
    creationTime: '2025-05-01 09:02:04',
    description: '默认VPC',
    tags: ['default'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - 安全组
const sgData = ref([
  {
    id: 1,
    instanceName: 'sg-default',
    instanceId: 'sg-2ze8mmbpj96wr4i8xxxx',
    provider: 'aliyun',
    regionId: 'cn-hangzhou',
    vpcId: 'vpc-default',
    creationTime: '2025-09-01 09:15:41',
    description: '默认安全组',
    tags: ['default'],
    lastSyncTime: '2025-04-30 10:00:00'
  }
]);

// 模拟数据 - ELB
const elbData = ref([
  {
    id: 1,
    instanceName: 'web-lb-prod',
    instanceId: 'lb-2zejplm93vgl58s1xxxx',
    status: 'running',
    addressType: '公网',
    address: '47.98.234.567',
    regionId: 'cn-hangzhou',
    provider: 'aliyun',
    creationTime: '2025-05-05 16:40:32',
    description: '生产环境Web负载均衡器',
    tags: ['env:prod', 'service:web'],
    lastSyncTime: '2025-04-30 10:02:44'
  }
]);

// 模拟数据 - RDS
const rdsData = ref([
  {
    id: 1,
    instanceName: 'mysql-prod-master',
    instanceId: 'rm-2ze3o57f291q7xxxx',
    status: 'running',
    dbType: 'MySQL 5.7',
    instanceType: 'rds.mysql.s3.large',
    regionAndZone: '华东1(杭州)/可用区B',
    regionId: 'cn-hangzhou',
    zoneId: 'cn-hangzhou-b',
    provider: 'aliyun',
    vpcId: 'vpc-default',
    creationTime: '2025-05-10 10:03:56',
    description: '生产环境MySQL主库',
    tags: ['env:prod', 'db:mysql', 'role:master'],
    lastSyncTime: '2025-04-30 10:04:31'
  }
]);

// 步骤变化标志
const stepChanged = ref(false);

// 组件挂载时执行
onMounted(() => {
  fetchEcsList();
});

// 监听步骤变化
watch(currentStep, async (newVal, oldVal) => {
  stepChanged.value = true;

  // 当从第三步返回第二步时，确保系统盘信息不丢失
  if (newVal === 2 && oldVal === 3) {
    if (createForm.imageId && createForm.instanceType && !createForm.systemDiskCategory) {
      await refreshSystemDiskOptions();
    }
  }

  // 当从第一步返回第零步时，确保实例类型和镜像兼容
  if (newVal === 0 && oldVal === 1) {
    if (createForm.imageId && createForm.instanceType) {
      await verifyInstanceTypeAndImageCompatibility();
    }
  }

  stepChanged.value = false;
});

// 获取状态颜色
const getStatusColor = (status: string) => {
  const statusColorMap: Record<string, string> = {
    'Running': 'green',
    'Stopped': 'red',
    'Starting': 'blue',
    'Stopping': 'orange',
    'Creating': 'purple',
    'Available': 'green',
    'running': 'green',
    'stopped': 'red',
    'starting': 'blue',
    'stopping': 'orange'
  };
  return statusColorMap[status] || 'default';
};

// 获取状态文本
const getStatusText = (status: string) => {
  const statusTextMap: Record<string, string> = {
    'Running': '运行中',
    'Stopped': '已停止',
    'Starting': '启动中',
    'Stopping': '停止中',
    'Creating': '创建中',
    'Available': '可用',
    'running': '运行中',
    'stopped': '已停止',
    'starting': '启动中',
    'stopping': '停止中'
  };
  return statusTextMap[status] || status;
};

// 获取云厂商名称
const getProviderName = (provider: string) => {
  const providerMap: Record<string, string> = {
    'aliyun': '阿里云',
    'tencent': '腾讯云',
    'huawei': '华为云',
    'aws': 'AWS'
  };
  return providerMap[provider] || provider;
};

// 获取付费类型名称
const getPayTypeName = (payType: string): string => {
  const map: Record<string, string> = {
    'PostPaid': '按量付费',
    'PrePaid': '包年包月',
  };
  return map[payType] || payType;
};

// 获取区域名称
const getRegionById = (regionId: string) => {
  return regionOptions.value.find(region => region.region === regionId);
};

// 获取可用区名称
const getZoneById = (zoneId: string) => {
  return zoneOptions.value.find(zone => zone.zone === zoneId);
};

// 获取实例类型详情
const getInstanceTypeById = (instanceTypeId: string) => {
  return instanceTypeOptions.value.find(type => type.instanceType === instanceTypeId);
};

// 获取系统盘类型详情
const getSystemDiskById = (diskId: string) => {
  return systemDiskOptions.value.find(disk => disk.systemDiskCategory === diskId);
};

// 获取数据盘类型详情
const getDataDiskById = (diskId: string) => {
  return dataDiskOptions.value.find(disk => disk.dataDiskCategory === diskId);
};

// 获取VPC详情
const getVpcById = (vpcId: string) => {
  return vpcOptions.value.find(vpc => vpc.vpcId === vpcId);
};

// 获取交换机详情
const getVSwitchById = (vSwitchId: string) => {
  return vSwitchOptions.value.find(vSwitch => vSwitch.vSwitchId === vSwitchId);
};

// 获取安全组详情
const getSecurityGroupById = (securityGroupId: string) => {
  return securityGroupOptions.value.find(sg => sg.securityGroupId === securityGroupId);
};

// 刷新系统盘选项
const refreshSystemDiskOptions = async () => {
  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId
    };

    const response = await getInstanceOptions(req);
    systemDiskOptions.value = response || [];
  } catch (error) {
    console.error('刷新系统盘选项失败:', error);
    message.error('获取系统盘类型列表失败');
  }
};

// 验证实例类型和镜像兼容性
const verifyInstanceTypeAndImageCompatibility = async () => {
  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId
    };

    const response = await getInstanceOptions(req);

    // 如果没有返回数据，说明当前实例类型和镜像不兼容
    if (!response || response.length === 0) {
      message.warning('当前选择的实例类型与镜像架构不兼容，请重新选择');
      createForm.imageId = '';

      // 重新加载镜像列表
      await handleInstanceTypeChange(createForm.instanceType);
    }
  } catch (error) {
    console.error('验证实例类型和镜像兼容性失败:', error);
  }
};

// 实例类型过滤器
const filterInstanceType = (input: string, option: any) => {
  const normalizedInput = input.toLowerCase().replace(/\s+/g, '');
  const normalizedLabel = option.label.toLowerCase().replace(/\s+/g, '');
  return normalizedLabel.indexOf(normalizedInput) >= 0;
};

// 镜像过滤器
const filterImage = (input: string, option: any) => {
  const normalizedInput = input.toLowerCase().replace(/\s+/g, '');
  const normalizedLabel = option.label.toLowerCase().replace(/\s+/g, '');
  return normalizedLabel.indexOf(normalizedInput) >= 0;
};

// 获取ECS列表
const fetchEcsList = async () => {
  loading.value = true;
  try {
    const params = {
      provider: filterForm.provider,
      region: filterForm.region,
      page: pagination.current,
      size: pagination.pageSize,
      instanceName: filterForm.name || undefined,
      status: filterForm.status || undefined
    };
    const response = await getEcsResourceList(params);
    ecsData.value = response.items || [];
    pagination.total = response.total || 0;
  } catch (error) {
    message.error('获取ECS实例列表失败');
    console.error('获取ECS实例列表失败:', error);
  } finally {
    loading.value = false;
  }
};

// 获取VPC选项
const fetchVpcOptions = async () => {
  if (!createForm.provider || !createForm.region) return;

  vpcLoading.value = true;
  vpcOptions.value = [];
  createForm.vpcId = '';
  createForm.vSwitchId = '';

  try {
    const req = {
      pageNumber: 1,
      pageSize: 50,
      provider: createForm.provider,
      region: createForm.region,
    };

    const response = await getVpcResourceList(req);

    vpcOptions.value = response.data.map((vpc: any) => ({
      vpcId: vpc.instance_id || vpc.vpc_id || vpc.vpcId,
      vpcName: vpc.vpcName || vpc.instance_name || '',
      cidrBlock: vpc.cidrBlock || '',
      description: vpc.description || ''
    }));

    vSwitchLoading.value = true;
    const vSwitches: VSwitchOption[] = [];

    for (const vpc of response.data) {
      if (vpc.vSwitchIds && Array.isArray(vpc.vSwitchIds) && vpc.vSwitchIds.length > 0) {
        for (const vSwitchId of vpc.vSwitchIds) {
          vSwitches.push({
            vSwitchId: vSwitchId,
            vSwitchName: `交换机-${vSwitchId.substring(vSwitchId.length - 8)}`,
            cidrBlock: '',
            zoneId: '',
            vpcId: vpc.instance_id || vpc.vpc_id || vpc.vpcId
          });
        }
      }
    }

    vSwitchOptions.value = vSwitches;
  } catch (error) {
    message.error('获取VPC列表失败');
    console.error('获取VPC列表失败:', error);
  } finally {
    vpcLoading.value = false;
    vSwitchLoading.value = false;
  }
};

// 获取安全组选项
const fetchSecurityGroupOptions = async () => {
  if (!createForm.provider || !createForm.region) return;

  securityGroupLoading.value = true;
  securityGroupOptions.value = [];
  createForm.securityGroupIds = [];

  try {
    const req = {
      provider: createForm.provider,
      region: createForm.region,
      pageNumber: 1,
      pageSize: 100
    };

    const response = await listSecurityGroups(req);

    securityGroupOptions.value = response.data.map((sg: any) => ({
      securityGroupId: sg.instance_id || sg.security_group_id,
      securityGroupName: sg.securityGroupName || sg.instance_name,
      description: sg.description || '',
      vpcId: sg.vpcId || sg.vpc_id || ''
    }));
  } catch (error) {
    message.error('获取安全组列表失败');
    console.error('获取安全组列表失败:', error);
  } finally {
    securityGroupLoading.value = false;
  }
};

// 处理表格变化
const handleTableChange = (pag: any) => {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  filterForm.pageNumber = pag.current;
  filterForm.pageSize = pag.pageSize;
  
  if (activeTab.value === 'ecs') {
    fetchEcsList();
  }
};

// 处理标签页切换
const handleTabChange = (key: string) => {
  // 切换到ECS标签页时刷新数据
  if (key === 'ecs') {
    fetchEcsList();
  }
};

// 处理搜索
const handleSearch = () => {
  pagination.current = 1;
  if (activeTab.value === 'ecs') {
    fetchEcsList();
  }
};

// 重置过滤条件
const resetFilter = () => {
  filterForm.provider = 'aliyun';
  filterForm.region = 'cn-hangzhou';
  filterForm.name = '';
  filterForm.status = undefined;
  filterForm.pageNumber = 1;
  filterForm.pageSize = 10;
  pagination.current = 1;
  
  if (activeTab.value === 'ecs') {
    fetchEcsList();
  }
};

// 同步资源
const handleSyncResources = () => {
  if (!filterForm.provider || !filterForm.region) {
    message.warning('请选择需要同步的云厂商和地区');
    return;
  }
  
  message.loading('正在同步资源，请稍候...', 2);
  
  setTimeout(() => {
    if (activeTab.value === 'ecs') {
      fetchEcsList();
    }
    message.success('资源同步成功');
  }, 2000);
};

// 显示创建模态框
const showCreateModal = (type: string) => {
  if (type === 'ecs') {
    currentStep.value = 0;
    Object.assign(createForm, {
      provider: 'aliyun',
      region: '',
      imageId: '',
      instanceType: '',
      amount: 1,
      zoneId: '',
      vpcId: '',
      vSwitchId: '',
      securityGroupIds: [],
      hostname: '',
      password: '',
      instanceName: '',
      payType: '',
      instanceChargeType: '',
      spotStrategy: 'NoSpot',
      description: '',
      systemDiskCategory: '',
      systemDiskSize: 40,
      dataDiskCategory: '',
      dataDiskSize: 100,
      dryRun: false,
      tags: {},
      periodUnit: 'Month',
      period: 1,
      autoRenew: false,
      spotDuration: 1,
    });
    tagsArray.value = [];
    tagInputValue.value = '';
  }
  
  (modals as Record<string, boolean>)[type] = true;
};

// 显示资源详情
const handleViewDetail = async (type: string, record: any) => {
  resourceType.value = type;
  detailVisible.value = true;
  detailLoading.value = true;
  
  try {
    if (type === 'ecs') {
      const req = {
        provider: record.cloud_provider,
        region: record.region_id,
        instanceId: record.instance_id
      };

      const response = await getEcsResourceDetail(req);
      resourceDetail.value = response.data;

      if (resourceDetail.value.diskIds && resourceDetail.value.diskIds.length > 0) {
        disks.value = resourceDetail.value.diskIds.map((diskId: string, index: number) => {
          return {
            diskId: diskId,
            diskName: index === 0 ? '系统盘' : `数据盘${index}`,
            type: index === 0 ? 'system' : 'data',
            category: 'cloud_essd',
            size: index === 0 ? 40 : 100
          };
        });
      } else {
        disks.value = [];
      }
    } else {
      // 对于其他资源类型，直接显示模拟数据
      resourceDetail.value = record;
    }
  } catch (error) {
    message.error(`获取${resourceDetailTitle.value}详情失败`);
    console.error(`获取${resourceDetailTitle.value}详情失败:`, error);
  } finally {
    detailLoading.value = false;
  }
};

// 处理ECS操作(启动/停止/重启)
const handleEcsAction = async (action: string, record: any) => {
  const actionMap: Record<string, string> = {
    'start': '启动',
    'stop': '停止',
    'restart': '重启'
  };
  
  const hide = message.loading(`正在${actionMap[action]}云服务器，请稍候...`, 0);
  
  try {
    const req = {
      provider: record.cloud_provider,
      region: record.region_id,
      instanceId: record.instance_id
    };

    if (action === 'start') {
      await startEcsResource(req);
      record.status = 'Starting';
    } else if (action === 'stop') {
      await stopEcsResource(req);
      record.status = 'Stopping';
    } else if (action === 'restart') {
      await restartEcsResource(req);
      record.status = 'Stopping';
    }
    
    message.success(`云服务器${actionMap[action]}操作已提交`);
    setTimeout(() => fetchEcsList(), 2000);
  } catch (error) {
    message.error(`${actionMap[action]}云服务器失败`);
    console.error(`${actionMap[action]}云服务器失败:`, error);
  } finally {
    hide();
  }
};

// 处理RDS操作(启动/停止/重启)
const handleRdsAction = (action: string, record: any) => {
  const actionMap: Record<string, string> = {
    'start': '启动',
    'stop': '停止',
    'restart': '重启'
  };
  
  message.loading(`正在${actionMap[action]}数据库实例，请稍候...`, 1);
  
  // 模拟操作请求
  setTimeout(() => {
    // 更新本地数据状态
    if (action === 'start') {
      record.status = 'running';
    } else if (action === 'stop') {
      record.status = 'stopped';
    }
    
    message.success(`数据库实例${actionMap[action]}操作已完成`);
  }, 1500);
};

// 删除资源
const handleDeleteResource = (type: string, record: any) => {
  const typeMap: Record<string, string> = {
    'ecs': '云服务器',
    'vpc': 'VPC',
    'sg': '安全组',
    'elb': '负载均衡',
    'rds': '数据库'
  };
  
  Modal.confirm({
    title: `确定要删除${typeMap[type]}吗？`,
    content: `您正在删除${typeMap[type]}: ${record.instance_name || record.instanceName}，该操作不可恢复。`,
    okText: '确认删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      if (type === 'ecs') {
        const hide = message.loading(`正在删除${typeMap[type]}，请稍候...`, 0);
        
        try {
          const req = {
            provider: record.cloud_provider,
            region: record.region_id,
            instanceId: record.instance_id
          };

          await deleteEcsResource(req);
          
          message.success(`${typeMap[type]}删除成功`);
          
          // 关闭详情抽屉（如果当前打开）
          if (detailVisible.value && resourceDetail.value && 
              resourceDetail.value.instance_id === record.instance_id) {
            detailVisible.value = false;
          }
          
          // 刷新列表
          fetchEcsList();
        } catch (error) {
          message.error(`删除${typeMap[type]}失败`);
          console.error(`删除${typeMap[type]}失败:`, error);
        } finally {
          hide();
        }
      } else {
        // 模拟其他资源类型的删除操作
        message.loading(`正在删除${typeMap[type]}，请稍候...`, 1);
        
        setTimeout(() => {
          if (type === 'vpc') {
            vpcData.value = vpcData.value.filter(item => item.id !== record.id);
          } else if (type === 'sg') {
            sgData.value = sgData.value.filter(item => item.id !== record.id);
          } else if (type === 'elb') {
            elbData.value = elbData.value.filter(item => item.id !== record.id);
          } else if (type === 'rds') {
            rdsData.value = rdsData.value.filter(item => item.id !== record.id);
          }
          
          message.success(`${typeMap[type]}删除成功`);
          
          // 关闭详情抽屉（如果当前打开）
          if (detailVisible.value && resourceType.value === type && 
              resourceDetail.value && resourceDetail.value.id === record.id) {
            detailVisible.value = false;
          }
        }, 1500);
      }
    }
  });
};

// 创建表单步骤控制
const nextStep = async () => {
  if (currentStep.value < 3) {
    if (currentStep.value === 0) {
      // 进入网络配置前，先验证实例类型和镜像是否兼容
      if (createForm.imageId && createForm.instanceType) {
        await verifyInstanceTypeAndImageCompatibility();
      }

      await fetchVpcOptions();
      await fetchSecurityGroupOptions();
    } else if (currentStep.value === 1 && !stepChanged.value) {
      // 进入系统配置前，确保系统盘类型已加载
      if (createForm.imageId && createForm.instanceType && (!systemDiskOptions.value.length || !createForm.systemDiskCategory)) {
        await refreshSystemDiskOptions();
      }
    }
    currentStep.value += 1;
  }
};

const prevStep = async () => {
  if (currentStep.value > 0) {
    currentStep.value -= 1;

    // 如果从第三步返回第二步，确保系统盘信息不丢失
    if (currentStep.value === 2 && !stepChanged.value) {
      if (createForm.imageId && createForm.instanceType && !createForm.systemDiskCategory) {
        await refreshSystemDiskOptions();
      }
    }

    // 如果从第一步返回第零步，需要确保实例类型和镜像兼容
    if (currentStep.value === 0 && !stepChanged.value) {
      if (createForm.imageId && createForm.instanceType) {
        await verifyInstanceTypeAndImageCompatibility();
      }
    }
  }
};

// 标签操作
const addTag = () => {
  if (tagInputValue.value && tagInputValue.value.includes('=')) {
    tagsArray.value.push(tagInputValue.value);

    const parts = tagInputValue.value.split('=');
    if (parts.length === 2 && createForm.tags) {
      const key = parts[0]?.trim();
      const value = parts[1]?.trim();

      if (key && value) {
        createForm.tags[key] = value;
      } else {
        message.warning('标签格式不正确，请确保包含 key=value 格式');
      }
    }

    tagInputValue.value = '';
  } else {
    message.warning('标签格式应为 key=value');
  }
};

const removeTag = (index: number) => {
  if (index >= 0 && index < tagsArray.value.length) {
    const tag = tagsArray.value[index];
    if (tag) {
      const parts = tag.split('=');
      if (parts.length === 2) {
        const key = parts[0]?.trim();
        if (key && createForm.tags && key in createForm.tags) {
          delete createForm.tags[key];
        }
      }

      tagsArray.value.splice(index, 1);
    }
  }
};

// 创建实例提交
const handleCreateSubmit = async () => {
  createLoading.value = true;

  // 再次验证实例类型与镜像的兼容性
  if (createForm.imageId && createForm.instanceType) {
    await verifyInstanceTypeAndImageCompatibility();

    // 确保系统盘类型已设置
    if (!createForm.systemDiskCategory) {
      await refreshSystemDiskOptions();
    }

    // 如果验证后镜像被清空，说明不兼容
    if (!createForm.imageId) {
      message.error('实例类型与镜像架构不兼容，请返回修改');
      createLoading.value = false;
      return;
    }
  }

  createForm.instanceChargeType = createForm.payType;

  try {
    // 将 Record<string, string> 类型的 tags 转换为 string[] 类型
    const tagsArray: string[] = [];
    if (createForm.tags) {
      for (const key in createForm.tags) {
        if (Object.prototype.hasOwnProperty.call(createForm.tags, key)) {
          tagsArray.push(`${key}=${createForm.tags[key]}`);
        }
      }
    }

    const createParams = {
      provider: createForm.provider,
      periodUnit: createForm.periodUnit,
      period: createForm.period,
      region: createForm.region,
      zoneId: createForm.zoneId,
      autoRenew: createForm.autoRenew,
      instanceChargeType: createForm.instanceChargeType,
      spotStrategy: createForm.spotStrategy,
      spotDuration: createForm.spotDuration,
      systemDiskSize: createForm.systemDiskSize,
      systemDiskCategory: createForm.systemDiskCategory,
      dataDiskSize: createForm.dataDiskSize,
      dataDiskCategory: createForm.dataDiskCategory,
      dryRun: createForm.dryRun,
      tags: tagsArray, // 使用转换后的字符串数组
      imageId: createForm.imageId,
      instanceType: createForm.instanceType,
      amount: createForm.amount || 1,
      vpcId: createForm.vpcId,
      vSwitchId: createForm.vSwitchId,
      securityGroupIds: createForm.securityGroupIds,
      hostname: createForm.hostname,
      password: createForm.password,
      instanceName: createForm.instanceName,
      payType: createForm.payType,
      description: createForm.description
    };

    await createEcsResource(createParams);
    message.success('ECS实例创建成功');
    modals.ecs = false;
    setTimeout(() => fetchEcsList(), 50000);
  } catch (error) {
    message.error('创建ECS实例失败');
    console.error('创建ECS实例失败:', error);
  } finally {
    createLoading.value = false;
  }
};

// 表单联动处理
const handleProviderChange = async (value: string) => {
  createForm.payType = '';
  createForm.region = '';
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  regionOptions.value = [];
  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  try {
    const req = { provider: value };
    const response = await getInstanceOptions(req);
    regionOptions.value = response || [];
  } catch (error) {
    message.error('获取地域列表失败');
  }
};

const handlePayTypeChange = async (value: string) => {
  createForm.region = '';
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: value
    };
    const response = await getInstanceOptions(req);
    regionOptions.value = response || [];

    if (regionOptions.value.length === 0) {
      // 使用备选数据
      regionOptions.value = [
        { region: 'cn-beijing', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
        { region: 'cn-hangzhou', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
        { region: 'cn-shanghai', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' },
        { region: 'cn-shenzhen', valid: true, dataDiskCategory: '', systemDiskCategory: '', instanceType: '', zone: '', payType: '', cpu: 0, memory: 0, imageId: '', osName: '', osType: '', architecture: '' }
      ];
    }
  } catch (error) {
    console.error('获取地域列表失败:', error);
    message.error('获取地域列表失败');
  }
};

const handleRegionChange = async (value: string) => {
  createForm.zoneId = '';
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';
  createForm.vpcId = '';
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  zoneOptions.value = [];
  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];
  vpcOptions.value = [];
  vSwitchOptions.value = [];
  securityGroupOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: value
    };
    const response = await getInstanceOptions(req);
    zoneOptions.value = response || [];
  } catch (error) {
    console.error('获取可用区列表失败:', error);
    message.error('获取可用区列表失败');
  }
};

const handleZoneChange = async (value: string) => {
  createForm.instanceType = '';
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  instanceTypeOptions.value = [];
  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: value
    };
    const response = await getInstanceOptions(req);
    instanceTypeOptions.value = response || [];
  } catch (error) {
    console.error('获取实例规格列表失败:', error);
    message.error('获取实例规格列表失败');
  }
};

const handleInstanceTypeChange = async (value: string) => {
  createForm.imageId = '';
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  imageOptions.value = [];
  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      pageNumber: 1,
      pageSize: 10,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: value
    };
    const response = await getInstanceOptions(req);
    imageOptions.value = response || [];

    if (imageOptions.value.length === 0) {
      message.warning('当前配置下没有可用的镜像选项');
    }
  } catch (error) {
    console.error('获取镜像列表失败:', error);
    message.error('获取镜像列表失败');
  }
};

const handleImageIdChange = async (value: string) => {
  createForm.systemDiskCategory = '';
  createForm.dataDiskCategory = '';

  systemDiskOptions.value = [];
  dataDiskOptions.value = [];

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: value
    };
    const response = await getInstanceOptions(req);

    // 如果响应为空，可能是实例类型和镜像不兼容
    if (!response || response.length === 0) {
      message.warning('选择的镜像与实例规格不兼容，请重新选择');
      createForm.imageId = '';
      return;
    }

    systemDiskOptions.value = response || [];
  } catch (error) {
    console.error('获取系统盘类型列表失败:', error);
    message.error('获取系统盘类型列表失败');
  }
};

const handleSystemDiskCategoryChange = async (value: string) => {
  createForm.dataDiskCategory = '';
  dataDiskOptions.value = [];

  createForm.systemDiskCategory = value;

  try {
    const req = {
      provider: createForm.provider,
      payType: createForm.payType,
      region: createForm.region,
      zone: createForm.zoneId,
      instanceType: createForm.instanceType,
      imageId: createForm.imageId,
      systemDiskCategory: value
    };
    const response = await getInstanceOptions(req);
    dataDiskOptions.value = response || [];
  } catch (error) {
    console.error('获取数据盘类型列表失败:', error);
    message.error('获取数据盘类型列表失败');
  }
};

const handleVpcChange = (vpcId: string) => {
  createForm.vSwitchId = '';
  createForm.securityGroupIds = [];

  const filteredVSwitches = vSwitchOptions.value.filter(vSwitch => vSwitch.vpcId === vpcId);
  const zoneVSwitch = filteredVSwitches.find(vSwitch => vSwitch.zoneId === createForm.zoneId);

  if (zoneVSwitch) {
    createForm.vSwitchId = zoneVSwitch.vSwitchId;
  } else if (filteredVSwitches.length > 0) {
    createForm.vSwitchId = filteredVSwitches[0]?.vSwitchId || '';
  }

  const filteredSecurityGroups = securityGroupOptions.value.filter(sg => sg.vpcId === vpcId);
  if (filteredSecurityGroups.length > 0) {
    createForm.securityGroupIds = [filteredSecurityGroups[0]?.securityGroupId || ''];
  }
};

const handleDataDiskCategoryChange = () => {
  // 数据盘类型选择变更时的处理逻辑
};
</script>

<style scoped lang="scss">
.resource-management {
  padding: 0 16px;
  
  .page-header {
    margin-bottom: 16px;
    padding: 16px 0;
  }
  .filter-card {
    margin-bottom: 16px;
  }
  .resource-tabs {
    .resource-card {
      margin-top: 16px;
      
      :deep(.ant-card-body) {
        padding: 0;
      }
    }
  }
  
  .action-buttons {
    display: flex;
    gap: 8px;
  }
  
  :deep(.ant-table-pagination.ant-pagination) {
    margin: 16px;
  }

  .create-steps {
    margin-bottom: 24px;
  }

  .create-form {
    max-height: 500px;
    overflow-y: auto;
    padding: 0 12px;
  }

  .steps-action {
    margin-top: 24px;
    display: flex;
    justify-content: flex-end;
  }

  .tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: this;
  }

  .drawer-actions {
    display: flex;
    justify-content: space-between;
    margin-top: 24px;
  }

  .tag-input-container {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }

  .tag-item {
    margin-bottom: 4px;
  }

  :deep(.ant-form-item) {
    margin-bottom: 20px;
  }

  :deep(.ant-tag) {
    margin-right: 0;
  }
}
</style>