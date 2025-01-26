-- MySQL dump 10.13  Distrib 8.2.0, for macos13 (arm64)
--
-- Host: localhost    Database: cloudOps
-- ------------------------------------------------------
-- Server version	8.2.0

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- Create database cloudOps
CREATE DATABASE IF NOT EXISTS cloudOps;

--
-- Table structure for table `apis`
--

DROP TABLE IF EXISTS `apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `apis` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(50) NOT NULL COMMENT 'API名称',
  `path` varchar(255) NOT NULL COMMENT 'API路径',
  `method` tinyint(1) NOT NULL COMMENT 'HTTP请求方法 1GET 2POST 3PUT 4DELETE',
  `description` varchar(500) DEFAULT NULL COMMENT 'API描述',
  `version` varchar(20) DEFAULT 'v1' COMMENT 'API版本',
  `category` tinyint(1) NOT NULL COMMENT 'API分类 1系统 2业务',
  `is_public` tinyint(1) DEFAULT '0' COMMENT '是否公开 0否 1是',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_del` (`name`),
  KEY `idx_apis_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=120 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `audit_logs`
--

DROP TABLE IF EXISTS `audit_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `audit_logs` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint unsigned NOT NULL COMMENT '操作用户ID',
  `ip_address` varchar(45) NOT NULL COMMENT '操作IP地址',
  `user_agent` varchar(255) NOT NULL COMMENT '用户代理',
  `http_method` varchar(10) NOT NULL COMMENT 'HTTP请求方法',
  `endpoint` varchar(255) NOT NULL COMMENT '请求端点',
  `operation_type` enum('CREATE','UPDATE','DELETE','OTHER') NOT NULL COMMENT '操作类型',
  `target_type` varchar(64) NOT NULL COMMENT '目标资源类型',
  `target_id` varchar(255) DEFAULT NULL COMMENT '目标资源ID',
  `status_code` bigint NOT NULL COMMENT 'HTTP状态码',
  `request_body` json DEFAULT NULL COMMENT '请求体',
  `response_body` json DEFAULT NULL COMMENT '响应体',
  `duration` bigint NOT NULL COMMENT '请求耗时',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_audit_logs_user_id` (`user_id`),
  KEY `idx_audit_logs_operation_type` (`operation_type`),
  KEY `idx_audit_logs_target_id` (`target_id`),
  KEY `idx_audit_logs_created_at` (`created_at`),
  KEY `idx_audit_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `bind_ecs`
--

DROP TABLE IF EXISTS `bind_ecs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `bind_ecs` (
  `resource_ecs_id` bigint NOT NULL COMMENT '主键ID',
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`resource_ecs_id`,`tree_node_id`),
  KEY `fk_bind_ecs_tree_node` (`tree_node_id`),
  CONSTRAINT `fk_bind_ecs_resource_ecs` FOREIGN KEY (`resource_ecs_id`) REFERENCES `resource_ecs` (`id`),
  CONSTRAINT `fk_bind_ecs_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `bind_elb`
--

DROP TABLE IF EXISTS `bind_elb`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `bind_elb` (
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  `resource_elb_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`tree_node_id`,`resource_elb_id`),
  KEY `fk_bind_elb_resource_elb` (`resource_elb_id`),
  CONSTRAINT `fk_bind_elb_resource_elb` FOREIGN KEY (`resource_elb_id`) REFERENCES `resource_elbs` (`id`),
  CONSTRAINT `fk_bind_elb_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `bind_elbs`
--

DROP TABLE IF EXISTS `bind_elbs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `bind_elbs` (
  `resource_elb_id` bigint NOT NULL COMMENT '主键ID',
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`resource_elb_id`,`tree_node_id`),
  KEY `fk_bind_elbs_tree_node` (`tree_node_id`),
  CONSTRAINT `fk_bind_elbs_resource_elb` FOREIGN KEY (`resource_elb_id`) REFERENCES `resource_elbs` (`id`),
  CONSTRAINT `fk_bind_elbs_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `bind_rds`
--

DROP TABLE IF EXISTS `bind_rds`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `bind_rds` (
  `resource_rds_id` bigint NOT NULL COMMENT '主键ID',
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`resource_rds_id`,`tree_node_id`),
  KEY `fk_bind_rds_tree_node` (`tree_node_id`),
  CONSTRAINT `fk_bind_rds_resource_rds` FOREIGN KEY (`resource_rds_id`) REFERENCES `resource_rds` (`id`),
  CONSTRAINT `fk_bind_rds_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `casbin_rule`
--

DROP TABLE IF EXISTS `casbin_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) DEFAULT NULL,
  `v0` varchar(100) DEFAULT NULL,
  `v1` varchar(100) DEFAULT NULL,
  `v2` varchar(100) DEFAULT NULL,
  `v3` varchar(100) DEFAULT NULL,
  `v4` varchar(100) DEFAULT NULL,
  `v5` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_apps`
--

DROP TABLE IF EXISTS `k8s_apps`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_apps` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '应用名称',
  `k8s_project_id` bigint DEFAULT NULL COMMENT '关联的 Kubernetes 项目ID',
  `tree_node_id` bigint DEFAULT NULL COMMENT '关联的树节点ID',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `cluster` varchar(100) DEFAULT NULL COMMENT '所属集群名称',
  `service_type` longtext COMMENT '服务类型',
  `namespace` longtext COMMENT 'Kubernetes 命名空间',
  `envs` longtext COMMENT '环境变量组，格式 key=value',
  `labels` longtext COMMENT '标签组，格式 key=value',
  `commands` longtext COMMENT '启动命令组',
  `args` longtext COMMENT '启动参数，空格分隔',
  `cpu_request` longtext COMMENT 'CPU 请求量',
  `cpu_limit` longtext COMMENT 'CPU 限制量',
  `memory_request` longtext COMMENT '内存请求量',
  `memory_limit` longtext COMMENT '内存限制量',
  `volume_json` text COMMENT '卷和挂载配置JSON',
  `port_json` text COMMENT '容器和服务端口配置',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_apps_deleted_at` (`deleted_at`),
  KEY `fk_k8s_projects_k8s_apps` (`k8s_project_id`),
  CONSTRAINT `fk_k8s_projects_k8s_apps` FOREIGN KEY (`k8s_project_id`) REFERENCES `k8s_projects` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_clusters`
--

DROP TABLE IF EXISTS `k8s_clusters`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_clusters` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '集群名称',
  `name_zh` varchar(100) DEFAULT NULL COMMENT '集群中文名称',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `cpu_request` longtext COMMENT 'CPU 请求量',
  `cpu_limit` longtext COMMENT 'CPU 限制量',
  `memory_request` longtext COMMENT '内存请求量',
  `memory_limit` longtext COMMENT '内存限制量',
  `restricted_name_space` longtext COMMENT '资源限制命名空间',
  `status` longtext COMMENT '集群状态',
  `env` longtext COMMENT '集群环境，例如 prod, stage, dev, rc, press',
  `version` longtext COMMENT '集群版本',
  `api_server_addr` longtext COMMENT 'API Server 地址',
  `kube_config_content` text COMMENT 'kubeConfig 内容',
  `action_timeout_seconds` bigint DEFAULT NULL COMMENT '操作超时时间（秒）',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_clusters_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_cronjobs`
--

DROP TABLE IF EXISTS `k8s_cronjobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_cronjobs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '定时任务名称',
  `cluster` varchar(100) DEFAULT NULL COMMENT '所属集群',
  `tree_node_id` bigint DEFAULT NULL COMMENT '关联的树节点ID',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `k8s_project_id` bigint DEFAULT NULL COMMENT '关联的 Kubernetes 项目ID',
  `namespace` longtext COMMENT '命名空间',
  `schedule` longtext COMMENT '调度表达式',
  `image` longtext COMMENT '镜像',
  `commands` longtext COMMENT '启动命令组',
  `args` longtext COMMENT '启动参数，空格分隔',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_cronjobs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_instances`
--

DROP TABLE IF EXISTS `k8s_instances`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_instances` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '实例名称',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `cluster` varchar(100) DEFAULT NULL COMMENT '所属集群',
  `envs` longtext COMMENT '环境变量组，格式 key=value',
  `labels` longtext COMMENT '标签组，格式 key=value',
  `commands` longtext COMMENT '启动命令组',
  `args` longtext COMMENT '启动参数，空格分隔',
  `cpu_request` longtext COMMENT 'CPU 请求量',
  `cpu_limit` longtext COMMENT 'CPU 限制量',
  `memory_request` longtext COMMENT '内存请求量',
  `memory_limit` longtext COMMENT '内存限制量',
  `volume_json` text COMMENT '卷和挂载配置JSON',
  `port_json` text COMMENT '容器和服务端口配置',
  `image` longtext COMMENT '镜像',
  `replicas` bigint DEFAULT NULL COMMENT '副本数量',
  `k8s_app_id` bigint DEFAULT NULL COMMENT '关联的 Kubernetes 应用ID',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_instances_deleted_at` (`deleted_at`),
  KEY `fk_k8s_apps_k8s_instances` (`k8s_app_id`),
  CONSTRAINT `fk_k8s_apps_k8s_instances` FOREIGN KEY (`k8s_app_id`) REFERENCES `k8s_apps` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_pods`
--

DROP TABLE IF EXISTS `k8s_pods`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_pods` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(200) DEFAULT NULL COMMENT 'Pod 名称',
  `namespace` varchar(200) DEFAULT NULL COMMENT 'Pod 所属的命名空间',
  `status` longtext COMMENT 'Pod 状态，例如 Running, Pending',
  `node_name` varchar(191) DEFAULT NULL COMMENT 'Pod 所在节点名称',
  `labels` text COMMENT 'Pod 标签键值对',
  `annotations` text COMMENT 'Pod 注解键值对',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_pods_deleted_at` (`deleted_at`),
  KEY `idx_k8s_pods_node_name` (`node_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_projects`
--

DROP TABLE IF EXISTS `k8s_projects`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_projects` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '项目名称',
  `name_zh` varchar(100) DEFAULT NULL COMMENT '项目中文名称',
  `cluster` varchar(100) DEFAULT NULL COMMENT '所属集群名称',
  `tree_node_id` bigint DEFAULT NULL COMMENT '关联的树节点ID',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_projects_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_yaml_tasks`
--

DROP TABLE IF EXISTS `k8s_yaml_tasks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_yaml_tasks` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(255) DEFAULT NULL COMMENT 'YAML 任务名称',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `template_id` bigint DEFAULT NULL COMMENT '关联的模板ID',
  `cluster_id` bigint DEFAULT NULL COMMENT '集群名称',
  `variables` text COMMENT 'yaml 变量，格式 k=v,k=v',
  `status` longtext COMMENT '当前状态',
  `apply_result` longtext COMMENT 'apply 后的返回数据',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_yaml_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `k8s_yaml_templates`
--

DROP TABLE IF EXISTS `k8s_yaml_templates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `k8s_yaml_templates` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '模板名称',
  `user_id` bigint DEFAULT NULL COMMENT '创建者用户ID',
  `content` text COMMENT 'yaml 模板内容',
  `cluster_id` bigint DEFAULT NULL COMMENT '对应集群id',
  PRIMARY KEY (`id`),
  KEY `idx_k8s_yaml_templates_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `menus`
--

DROP TABLE IF EXISTS `menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `menus` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(50) NOT NULL COMMENT '菜单显示名称',
  `parent_id` bigint DEFAULT '0' COMMENT '上级菜单ID,0表示顶级菜单',
  `path` varchar(255) NOT NULL COMMENT '前端路由访问路径',
  `component` varchar(255) NOT NULL COMMENT '前端组件文件路径',
  `route_name` varchar(50) NOT NULL COMMENT '前端路由名称',
  `hidden` tinyint(1) DEFAULT '0' COMMENT '菜单是否隐藏 0显示 1隐藏',
  `redirect` varchar(255) DEFAULT '' COMMENT '重定向路径',
  `meta` json DEFAULT NULL COMMENT '菜单元数据',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_route_del` (`route_name`),
  KEY `idx_menus_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=32 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_alert_events`
--

DROP TABLE IF EXISTS `monitor_alert_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_events` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `alert_name` varchar(200) NOT NULL COMMENT '告警名称',
  `fingerprint` varchar(100) NOT NULL COMMENT '告警唯一ID',
  `status` varchar(50) NOT NULL DEFAULT 'firing' COMMENT '告警状态(firing/silenced/claimed/resolved)',
  `rule_id` bigint NOT NULL COMMENT '关联的告警规则ID',
  `send_group_id` bigint NOT NULL COMMENT '关联的发送组ID',
  `event_times` bigint NOT NULL DEFAULT '1' COMMENT '触发次数',
  `silence_id` varchar(100) DEFAULT NULL COMMENT 'AlertManager返回的静默ID',
  `ren_ling_user_id` bigint DEFAULT NULL COMMENT '认领告警的用户ID',
  `labels` text NOT NULL COMMENT '标签组,格式为key=value',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_fingerprint_deleted_at` (`fingerprint`,`deleted_at`),
  KEY `idx_monitor_alert_events_rule_id` (`rule_id`),
  KEY `idx_monitor_alert_events_send_group_id` (`send_group_id`),
  KEY `idx_monitor_alert_events_ren_ling_user_id` (`ren_ling_user_id`),
  KEY `idx_monitor_alert_events_deleted_at` (`deleted_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_alert_manager_pools`
--

DROP TABLE IF EXISTS `monitor_alert_manager_pools`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_manager_pools` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) NOT NULL COMMENT 'AlertManager实例名称',
  `alert_manager_instances` text NOT NULL COMMENT 'AlertManager实例列表',
  `user_id` bigint NOT NULL COMMENT '所属用户ID',
  `resolve_timeout` varchar(50) NOT NULL DEFAULT '5m' COMMENT '告警恢复超时时间',
  `group_wait` varchar(50) NOT NULL DEFAULT '30s' COMMENT '首次告警等待时间',
  `group_interval` varchar(50) NOT NULL DEFAULT '5m' COMMENT '告警分组间隔时间',
  `repeat_interval` varchar(50) NOT NULL DEFAULT '4h' COMMENT '重复告警间隔',
  `group_by` text NOT NULL COMMENT '告警分组标签列表',
  `receiver` varchar(100) NOT NULL COMMENT '默认接收者',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_alert_manager_pools_deleted_at` (`deleted_at`),
  KEY `idx_monitor_alert_manager_pools_user_id` (`user_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_alert_rules`
--

DROP TABLE IF EXISTS `monitor_alert_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_rules` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '告警规则名称',
  `user_id` bigint NOT NULL COMMENT '创建该告警规则的用户ID',
  `pool_id` bigint NOT NULL COMMENT '关联的Prometheus实例池ID',
  `send_group_id` bigint NOT NULL COMMENT '关联的发送组ID',
  `tree_node_id` bigint NOT NULL COMMENT '绑定的树节点ID',
  `enable` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用告警规则',
  `expr` text NOT NULL COMMENT '告警规则表达式',
  `severity` varchar(50) NOT NULL DEFAULT 'warning' COMMENT '告警级别(critical/warning/info)',
  `grafana_link` text COMMENT 'Grafana大盘链接',
  `for_time` varchar(50) NOT NULL DEFAULT '5m' COMMENT '持续时间',
  `labels` text COMMENT '标签组(key=value)',
  `annotations` text COMMENT '注解(key=value)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_alert_rules_deleted_at` (`deleted_at`),
  KEY `idx_monitor_alert_rules_user_id` (`user_id`),
  KEY `idx_monitor_alert_rules_pool_id` (`pool_id`),
  KEY `idx_monitor_alert_rules_send_group_id` (`send_group_id`),
  KEY `idx_monitor_alert_rules_tree_node_id` (`tree_node_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_on_duty_changes`
--

DROP TABLE IF EXISTS `monitor_on_duty_changes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_changes` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID',
  `user_id` bigint DEFAULT NULL COMMENT '创建者ID',
  `date` varchar(10) NOT NULL COMMENT '换班日期',
  `origin_user_id` bigint DEFAULT NULL COMMENT '原值班人ID',
  `on_duty_user_id` bigint DEFAULT NULL COMMENT '新值班人ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_date_deleted_at` (`on_duty_group_id`,`date`,`deleted_at`),
  KEY `idx_monitor_on_duty_changes_user_id` (`user_id`),
  KEY `idx_monitor_on_duty_changes_origin_user_id` (`origin_user_id`),
  KEY `idx_monitor_on_duty_changes_on_duty_user_id` (`on_duty_user_id`),
  KEY `idx_monitor_on_duty_changes_deleted_at` (`deleted_at`),
  KEY `idx_monitor_on_duty_changes_on_duty_group_id` (`on_duty_group_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_on_duty_groups`
--

DROP TABLE IF EXISTS `monitor_on_duty_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '值班组名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索',
  `user_id` bigint DEFAULT NULL COMMENT '创建该值班组的用户ID',
  `shift_days` bigint DEFAULT NULL COMMENT '轮班周期，以天为单位',
  `yesterday_normal_duty_user_id` bigint DEFAULT NULL COMMENT '昨天的正常排班值班人ID，由cron任务设置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_on_duty_groups_deleted_at` (`deleted_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_on_duty_histories`
--

DROP TABLE IF EXISTS `monitor_on_duty_histories`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_histories` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID',
  `date_string` varchar(10) NOT NULL COMMENT '值班日期',
  `on_duty_user_id` bigint DEFAULT NULL COMMENT '当天值班人员ID',
  `origin_user_id` bigint DEFAULT NULL COMMENT '原计划值班人员ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_group_date_deleted_at` (`on_duty_group_id`,`date_string`,`deleted_at`),
  KEY `idx_monitor_on_duty_histories_origin_user_id` (`origin_user_id`),
  KEY `idx_monitor_on_duty_histories_deleted_at` (`deleted_at`),
  KEY `idx_monitor_on_duty_histories_on_duty_group_id` (`on_duty_group_id`),
  KEY `idx_monitor_on_duty_histories_on_duty_user_id` (`on_duty_user_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_on_duty_users`
--

DROP TABLE IF EXISTS `monitor_on_duty_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_users` (
  `monitor_on_duty_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_on_duty_group_id`,`user_id`),
  KEY `fk_monitor_on_duty_users_user` (`user_id`),
  CONSTRAINT `fk_monitor_on_duty_users_monitor_on_duty_group` FOREIGN KEY (`monitor_on_duty_group_id`) REFERENCES `monitor_on_duty_groups` (`id`),
  CONSTRAINT `fk_monitor_on_duty_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_record_rules`
--

DROP TABLE IF EXISTS `monitor_record_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_record_rules` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '记录规则名称',
  `user_id` bigint NOT NULL COMMENT '创建该记录规则的用户ID',
  `pool_id` bigint NOT NULL COMMENT '关联的Prometheus实例池ID',
  `tree_node_id` bigint NOT NULL COMMENT '绑定的树节点ID',
  `enable` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用记录规则',
  `for_time` varchar(50) NOT NULL DEFAULT '5m' COMMENT '持续时间',
  `expr` text NOT NULL COMMENT '记录规则表达式',
  `labels` text COMMENT '标签组(key=value)',
  `annotations` text COMMENT '注解(key=value)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_record_rules_deleted_at` (`deleted_at`),
  KEY `idx_monitor_record_rules_user_id` (`user_id`),
  KEY `idx_monitor_record_rules_pool_id` (`pool_id`),
  KEY `idx_monitor_record_rules_tree_node_id` (`tree_node_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_scrape_jobs`
--

DROP TABLE IF EXISTS `monitor_scrape_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_scrape_jobs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '采集任务名称',
  `user_id` bigint NOT NULL COMMENT '任务关联的用户ID',
  `enable` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用采集任务',
  `service_discovery_type` varchar(50) NOT NULL DEFAULT 'http' COMMENT '服务发现类型(k8s/http)',
  `metrics_path` varchar(255) NOT NULL DEFAULT '/metrics' COMMENT '监控采集的路径',
  `scheme` varchar(10) NOT NULL DEFAULT 'http' COMMENT '监控采集的协议方案(http/https)',
  `scrape_interval` bigint NOT NULL DEFAULT '30' COMMENT '采集的时间间隔(秒)',
  `scrape_timeout` bigint NOT NULL DEFAULT '10' COMMENT '采集的超时时间(秒)',
  `pool_id` bigint NOT NULL COMMENT '关联的采集池ID',
  `relabel_configs_yaml_string` text COMMENT 'relabel配置的YAML字符串',
  `refresh_interval` bigint NOT NULL DEFAULT '300' COMMENT '刷新目标的时间间隔(秒)',
  `port` bigint NOT NULL DEFAULT '9090' COMMENT '采集端口号',
  `tree_node_ids` text COMMENT '服务树节点ID列表',
  `kube_config_file_path` varchar(255) DEFAULT NULL COMMENT 'K8s配置文件路径',
  `tls_ca_file_path` varchar(255) DEFAULT NULL COMMENT 'TLS CA证书文件路径',
  `tls_ca_content` text COMMENT 'TLS CA证书内容',
  `bearer_token` text COMMENT '鉴权Token内容',
  `bearer_token_file` varchar(255) DEFAULT NULL COMMENT '鉴权Token文件路径',
  `kubernetes_sd_role` varchar(50) DEFAULT 'pod' COMMENT 'K8s服务发现角色',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_scrape_jobs_deleted_at` (`deleted_at`),
  KEY `idx_monitor_scrape_jobs_user_id` (`user_id`),
  KEY `idx_monitor_scrape_jobs_pool_id` (`pool_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_scrape_pools`
--

DROP TABLE IF EXISTS `monitor_scrape_pools`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_scrape_pools` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) NOT NULL COMMENT 'pool池名称',
  `prometheus_instances` text COMMENT 'Prometheus实例ID列表',
  `alert_manager_instances` text COMMENT 'AlertManager实例ID列表',
  `user_id` bigint NOT NULL COMMENT '所属用户ID',
  `scrape_interval` smallint NOT NULL DEFAULT '30' COMMENT '采集间隔(秒)',
  `scrape_timeout` smallint NOT NULL DEFAULT '10' COMMENT '采集超时(秒)',
  `remote_timeout_seconds` smallint NOT NULL DEFAULT '5' COMMENT '远程写入超时(秒)',
  `support_alert` tinyint(1) NOT NULL DEFAULT '0' COMMENT '告警支持(0:不支持,1:支持)',
  `support_record` tinyint(1) NOT NULL DEFAULT '0' COMMENT '预聚合支持(0:不支持,1:支持)',
  `external_labels` text COMMENT '外部标签（格式：[key1=val1,key2=val2]）',
  `remote_write_url` varchar(512) DEFAULT NULL COMMENT '远程写入地址',
  `remote_read_url` varchar(512) DEFAULT NULL COMMENT '远程读取地址',
  `alert_manager_url` varchar(512) DEFAULT NULL COMMENT 'AlertManager地址',
  `rule_file_path` varchar(512) DEFAULT NULL COMMENT '告警规则文件路径',
  `record_file_path` varchar(512) DEFAULT NULL COMMENT '记录规则文件路径',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  KEY `idx_monitor_scrape_pools_user_id` (`user_id`),
  KEY `idx_monitor_scrape_pools_deleted_at` (`deleted_at`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_send_group_first_upgrade_users`
--

DROP TABLE IF EXISTS `monitor_send_group_first_upgrade_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_send_group_first_upgrade_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_monitor_send_group_first_upgrade_users_user` (`user_id`),
  CONSTRAINT `fk_monitor_send_group_first_upgrade_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_monitor_send_group_first_upgrade_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_send_group_second_upgrade_users`
--

DROP TABLE IF EXISTS `monitor_send_group_second_upgrade_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_send_group_second_upgrade_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_monitor_send_group_second_upgrade_users_user` (`user_id`),
  CONSTRAINT `fk_monitor_send_group_second_upgrade_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_monitor_send_group_second_upgrade_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_send_group_static_receive_users`
--

DROP TABLE IF EXISTS `monitor_send_group_static_receive_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_send_group_static_receive_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_monitor_send_group_static_receive_users_user` (`user_id`),
  CONSTRAINT `fk_monitor_send_group_static_receive_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_monitor_send_group_static_receive_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `monitor_send_groups`
--

DROP TABLE IF EXISTS `monitor_send_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_send_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '发送组英文名称',
  `name_zh` varchar(100) DEFAULT NULL COMMENT '发送组中文名称',
  `enable` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否启用发送组',
  `user_id` bigint NOT NULL COMMENT '创建该发送组的用户ID',
  `pool_id` bigint NOT NULL COMMENT '关联的AlertManager实例ID',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID',
  `fei_shu_qun_robot_token` varchar(255) DEFAULT NULL COMMENT '飞书机器人Token',
  `repeat_interval` varchar(50) DEFAULT '4h' COMMENT '重复发送时间间隔',
  `send_resolved` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否发送恢复通知',
  `notify_methods` text COMMENT '通知方法列表',
  `need_upgrade` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否需要告警升级',
  `upgrade_minutes` bigint DEFAULT '30' COMMENT '告警升级等待时间(分钟)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_deleted_at` (`name`,`deleted_at`),
  UNIQUE KEY `idx_name_zh_deleted_at` (`name_zh`,`deleted_at`),
  KEY `idx_monitor_send_groups_deleted_at` (`deleted_at`),
  KEY `idx_monitor_send_groups_user_id` (`user_id`),
  KEY `idx_monitor_send_groups_pool_id` (`pool_id`),
  KEY `idx_monitor_send_groups_on_duty_group_id` (`on_duty_group_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `resource_ecs`
--

DROP TABLE IF EXISTS `resource_ecs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `resource_ecs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `instance_name` varchar(100) DEFAULT NULL COMMENT '资源实例名称，支持模糊搜索',
  `hash` varchar(200) DEFAULT NULL COMMENT '用于资源更新的哈希值',
  `vendor` longtext COMMENT '云厂商名称，1=个人，2=阿里云，3=华为云，4=腾讯云，5=AWS',
  `create_by_order` tinyint(1) DEFAULT NULL COMMENT '是否由工单创建，工单创建的资源不会被自动更新删除',
  `image` varchar(100) DEFAULT NULL COMMENT '镜像名称',
  `vpc_id` varchar(100) DEFAULT NULL COMMENT '专有网络 VPC ID',
  `zone_id` varchar(100) DEFAULT NULL COMMENT '实例所属可用区 ID，如 cn-hangzhou-g',
  `env` varchar(50) DEFAULT NULL COMMENT '环境标识，如 dev、stage、prod',
  `pay_type` varchar(50) DEFAULT NULL COMMENT '付费类型，按量付费或包年包月',
  `status` varchar(50) DEFAULT NULL COMMENT '资源状态，如 运行中、已停止、创建中',
  `description` text COMMENT '资源描述，如 CentOS 7.4 操作系统',
  `tags` varchar(500) DEFAULT NULL COMMENT '资源标签集合，用于分类和筛选',
  `security_group_ids` varchar(500) DEFAULT NULL COMMENT '安全组 ID 列表',
  `private_ip_address` varchar(500) DEFAULT NULL COMMENT '私有 IP 地址列表',
  `public_ip_address` varchar(500) DEFAULT NULL COMMENT '公网 IP 地址列表',
  `ip_addr` varchar(45) DEFAULT NULL COMMENT '单个公网 IP 地址',
  `port` bigint DEFAULT '22' COMMENT '端口号',
  `username` varchar(191) DEFAULT 'root' COMMENT '用户名',
  `encrypted_password` varchar(500) DEFAULT NULL COMMENT '加密后的密码',
  `key` longtext COMMENT '秘钥',
  `mode` varchar(191) DEFAULT 'password' COMMENT '认证方式',
  `creation_time` varchar(30) DEFAULT NULL COMMENT '创建时间，ISO 8601 格式',
  `os_type` varchar(50) DEFAULT NULL COMMENT '操作系统类型，例如 win、linux',
  `vm_type` bigint DEFAULT '1' COMMENT '设备类型，1=虚拟设备，2=物理设备',
  `instance_type` varchar(100) DEFAULT NULL COMMENT '实例类型，例：ecs.g8a.2xlarge',
  `cpu` bigint DEFAULT NULL COMMENT '虚拟 CPU 核数',
  `memory` bigint DEFAULT NULL COMMENT '内存大小，单位 GiB',
  `disk` bigint DEFAULT NULL COMMENT '磁盘大小，单位 GiB',
  `os_name` varchar(100) DEFAULT NULL COMMENT '操作系统名称，例：CentOS 7.4 64 位',
  `image_id` varchar(100) DEFAULT NULL COMMENT '镜像模板 ID',
  `hostname` varchar(100) DEFAULT NULL COMMENT '主机名',
  `password` longtext COMMENT '密码',
  `network_interfaces` varchar(500) DEFAULT NULL COMMENT '弹性网卡 ID 集合',
  `disk_ids` varchar(500) DEFAULT NULL COMMENT '云盘 ID 集合',
  `start_time` varchar(30) DEFAULT NULL COMMENT '最近启动时间, ISO 8601 标准, UTC+0 时间',
  `auto_release_time` varchar(30) DEFAULT NULL COMMENT '自动释放时间, ISO 8601 标准, UTC+0 时间',
  `last_invoked_time` varchar(30) DEFAULT NULL COMMENT '最近调用时间, ISO 8601 标准, UTC+0 时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_resource_ecs_ip_addr` (`ip_addr`),
  UNIQUE KEY `idx_resource_ecs_instance_name` (`instance_name`),
  UNIQUE KEY `idx_resource_ecs_hash` (`hash`),
  KEY `idx_resource_ecs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `resource_elbs`
--

DROP TABLE IF EXISTS `resource_elbs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `resource_elbs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `instance_name` varchar(100) DEFAULT NULL COMMENT '资源实例名称，支持模糊搜索',
  `hash` varchar(200) DEFAULT NULL COMMENT '用于资源更新的哈希值',
  `vendor` longtext COMMENT '云厂商名称，1=个人，2=阿里云，3=华为云，4=腾讯云，5=AWS',
  `create_by_order` tinyint(1) DEFAULT NULL COMMENT '是否由工单创建，工单创建的资源不会被自动更新删除',
  `image` varchar(100) DEFAULT NULL COMMENT '镜像名称',
  `vpc_id` varchar(100) DEFAULT NULL COMMENT '专有网络 VPC ID',
  `zone_id` varchar(100) DEFAULT NULL COMMENT '实例所属可用区 ID，如 cn-hangzhou-g',
  `env` varchar(50) DEFAULT NULL COMMENT '环境标识，如 dev、stage、prod',
  `pay_type` varchar(50) DEFAULT NULL COMMENT '付费类型，按量付费或包年包月',
  `status` varchar(50) DEFAULT NULL COMMENT '资源状态，如 运行中、已停止、创建中',
  `description` text COMMENT '资源描述，如 CentOS 7.4 操作系统',
  `tags` varchar(500) DEFAULT NULL COMMENT '资源标签集合，用于分类和筛选',
  `security_group_ids` varchar(500) DEFAULT NULL COMMENT '安全组 ID 列表',
  `private_ip_address` varchar(500) DEFAULT NULL COMMENT '私有 IP 地址列表',
  `public_ip_address` varchar(500) DEFAULT NULL COMMENT '公网 IP 地址列表',
  `ip_addr` varchar(45) DEFAULT NULL COMMENT '单个公网 IP 地址',
  `port` bigint DEFAULT '22' COMMENT '端口号',
  `username` varchar(191) DEFAULT 'root' COMMENT '用户名',
  `encrypted_password` varchar(500) DEFAULT NULL COMMENT '加密后的密码',
  `key` longtext COMMENT '秘钥',
  `mode` varchar(191) DEFAULT 'password' COMMENT '认证方式',
  `creation_time` varchar(30) DEFAULT NULL COMMENT '创建时间，ISO 8601 格式',
  `load_balancer_type` varchar(50) DEFAULT NULL COMMENT '负载均衡类型, 例: nlb, alb, clb',
  `bandwidth_capacity` bigint DEFAULT NULL COMMENT '带宽容量上限, 单位 Mb, 例: 50',
  `address_type` varchar(50) DEFAULT NULL COMMENT '地址类型, 公网或内网',
  `dns_name` varchar(255) DEFAULT NULL COMMENT 'DNS 解析地址',
  `bandwidth_package_id` varchar(100) DEFAULT NULL COMMENT '绑定的带宽包 ID',
  `cross_zone_enabled` tinyint(1) DEFAULT NULL COMMENT '是否启用跨可用区',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_resource_elbs_instance_name` (`instance_name`),
  UNIQUE KEY `idx_resource_elbs_hash` (`hash`),
  UNIQUE KEY `idx_resource_elbs_ip_addr` (`ip_addr`),
  KEY `idx_resource_elbs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `resource_rds`
--

DROP TABLE IF EXISTS `resource_rds`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `resource_rds` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `instance_name` varchar(100) DEFAULT NULL COMMENT '资源实例名称，支持模糊搜索',
  `hash` varchar(200) DEFAULT NULL COMMENT '用于资源更新的哈希值',
  `vendor` longtext COMMENT '云厂商名称，1=个人，2=阿里云，3=华为云，4=腾讯云，5=AWS',
  `create_by_order` tinyint(1) DEFAULT NULL COMMENT '是否由工单创建，工单创建的资源不会被自动更新删除',
  `image` varchar(100) DEFAULT NULL COMMENT '镜像名称',
  `vpc_id` varchar(100) DEFAULT NULL COMMENT '专有网络 VPC ID',
  `zone_id` varchar(100) DEFAULT NULL COMMENT '实例所属可用区 ID，如 cn-hangzhou-g',
  `env` varchar(50) DEFAULT NULL COMMENT '环境标识，如 dev、stage、prod',
  `pay_type` varchar(50) DEFAULT NULL COMMENT '付费类型，按量付费或包年包月',
  `status` varchar(50) DEFAULT NULL COMMENT '资源状态，如 运行中、已停止、创建中',
  `description` text COMMENT '资源描述，如 CentOS 7.4 操作系统',
  `tags` varchar(500) DEFAULT NULL COMMENT '资源标签集合，用于分类和筛选',
  `security_group_ids` varchar(500) DEFAULT NULL COMMENT '安全组 ID 列表',
  `private_ip_address` varchar(500) DEFAULT NULL COMMENT '私有 IP 地址列表',
  `public_ip_address` varchar(500) DEFAULT NULL COMMENT '公网 IP 地址列表',
  `ip_addr` varchar(45) DEFAULT NULL COMMENT '单个公网 IP 地址',
  `port` bigint DEFAULT '22' COMMENT '端口号',
  `username` varchar(191) DEFAULT 'root' COMMENT '用户名',
  `encrypted_password` varchar(500) DEFAULT NULL COMMENT '加密后的密码',
  `key` longtext COMMENT '秘钥',
  `mode` varchar(191) DEFAULT 'password' COMMENT '认证方式',
  `creation_time` varchar(30) DEFAULT NULL COMMENT '创建时间，ISO 8601 格式',
  `engine` varchar(50) DEFAULT NULL COMMENT '数据库引擎类型, 例: mysql, mariadb, postgresql',
  `db_instance_net_type` varchar(50) DEFAULT NULL COMMENT '实例网络类型, 例: Internet(外网), Intranet(内网)',
  `db_instance_class` varchar(100) DEFAULT NULL COMMENT '实例规格, 例: rds.mys2.small',
  `db_instance_type` varchar(50) DEFAULT NULL COMMENT '实例类型, 例: Primary(主实例), Readonly(只读实例)',
  `engine_version` varchar(50) DEFAULT NULL COMMENT '数据库版本, 例: 8.0, 5.7',
  `master_instance_id` varchar(100) DEFAULT NULL COMMENT '主实例 ID',
  `db_instance_status` varchar(50) DEFAULT NULL COMMENT '实例状态',
  `replicate_id` varchar(100) DEFAULT NULL COMMENT '复制实例 ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_resource_rds_ip_addr` (`ip_addr`),
  UNIQUE KEY `idx_resource_rds_instance_name` (`instance_name`),
  UNIQUE KEY `idx_resource_rds_hash` (`hash`),
  KEY `idx_resource_rds_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `role_apis`
--

DROP TABLE IF EXISTS `role_apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `role_apis` (
  `api_id` bigint NOT NULL COMMENT '主键ID',
  `role_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`api_id`,`role_id`),
  KEY `fk_role_apis_role` (`role_id`),
  CONSTRAINT `fk_role_apis_api` FOREIGN KEY (`api_id`) REFERENCES `apis` (`id`),
  CONSTRAINT `fk_role_apis_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `role_menus`
--

DROP TABLE IF EXISTS `role_menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `role_menus` (
  `menu_id` bigint NOT NULL COMMENT '主键ID',
  `role_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`menu_id`,`role_id`),
  KEY `fk_role_menus_role` (`role_id`),
  CONSTRAINT `fk_role_menus_menu` FOREIGN KEY (`menu_id`) REFERENCES `menus` (`id`),
  CONSTRAINT `fk_role_menus_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `roles`
--

DROP TABLE IF EXISTS `roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `desc` varchar(255) DEFAULT NULL COMMENT '角色描述',
  `role_type` tinyint(1) NOT NULL COMMENT '角色类型 1系统角色 2自定义角色',
  `is_default` tinyint(1) DEFAULT '0' COMMENT '是否为默认角色 0否 1是',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_del` (`name`),
  KEY `idx_roles_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `terraform_configs`
--

DROP TABLE IF EXISTS `terraform_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `terraform_configs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `region` longtext,
  `name` longtext,
  `instance` longblob,
  `vpc` longblob,
  `security` longblob,
  `env` longtext,
  `pay_type` longtext,
  `description` longtext,
  `tags` longtext,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tree_node_ops_admins`
--

DROP TABLE IF EXISTS `tree_node_ops_admins`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tree_node_ops_admins` (
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`tree_node_id`,`user_id`),
  KEY `fk_tree_node_ops_admins_user` (`user_id`),
  CONSTRAINT `fk_tree_node_ops_admins_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`),
  CONSTRAINT `fk_tree_node_ops_admins_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tree_node_rd_admins`
--

DROP TABLE IF EXISTS `tree_node_rd_admins`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tree_node_rd_admins` (
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`tree_node_id`,`user_id`),
  KEY `fk_tree_node_rd_admins_user` (`user_id`),
  CONSTRAINT `fk_tree_node_rd_admins_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`),
  CONSTRAINT `fk_tree_node_rd_admins_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tree_node_rd_members`
--

DROP TABLE IF EXISTS `tree_node_rd_members`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tree_node_rd_members` (
  `tree_node_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`tree_node_id`,`user_id`),
  KEY `fk_tree_node_rd_members_user` (`user_id`),
  CONSTRAINT `fk_tree_node_rd_members_tree_node` FOREIGN KEY (`tree_node_id`) REFERENCES `tree_nodes` (`id`),
  CONSTRAINT `fk_tree_node_rd_members_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `tree_nodes`
--

DROP TABLE IF EXISTS `tree_nodes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tree_nodes` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `title` varchar(50) DEFAULT NULL COMMENT '节点名称',
  `pid` bigint DEFAULT NULL COMMENT '父节点 ID',
  `level` bigint DEFAULT NULL COMMENT '节点层级',
  `is_leaf` bigint DEFAULT NULL COMMENT '是否为叶子节点 0为非叶子节点 1为叶子节点',
  `desc` text COMMENT '节点描述',
  PRIMARY KEY (`id`),
  KEY `idx_tree_nodes_deleted_at` (`deleted_at`),
  KEY `idx_tree_nodes_pid` (`pid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_apis`
--

DROP TABLE IF EXISTS `user_apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_apis` (
  `api_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`api_id`,`user_id`),
  KEY `fk_user_apis_user` (`user_id`),
  CONSTRAINT `fk_user_apis_api` FOREIGN KEY (`api_id`) REFERENCES `apis` (`id`),
  CONSTRAINT `fk_user_apis_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_menus`
--

DROP TABLE IF EXISTS `user_menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_menus` (
  `menu_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`menu_id`,`user_id`),
  KEY `fk_user_menus_user` (`user_id`),
  CONSTRAINT `fk_user_menus_menu` FOREIGN KEY (`menu_id`) REFERENCES `menus` (`id`),
  CONSTRAINT `fk_user_menus_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `user_roles`
--

DROP TABLE IF EXISTS `user_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_roles` (
  `role_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`role_id`,`user_id`),
  KEY `fk_user_roles_user` (`user_id`),
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`),
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` bigint DEFAULT NULL COMMENT '创建时间',
  `updated_at` bigint DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint DEFAULT '0' COMMENT '删除时间',
  `username` varchar(100) NOT NULL COMMENT '用户登录名',
  `password` varchar(255) NOT NULL COMMENT '用户登录密码',
  `real_name` varchar(100) DEFAULT NULL COMMENT '用户真实姓名',
  `desc` text COMMENT '用户描述',
  `mobile` varchar(20) DEFAULT NULL COMMENT '手机号',
  `fei_shu_user_id` varchar(50) DEFAULT NULL COMMENT '飞书用户ID',
  `account_type` tinyint DEFAULT '1' COMMENT '账号类型 1普通用户 2服务账号',
  `home_path` varchar(255) DEFAULT '/' COMMENT '登录后的默认首页',
  `enable` tinyint DEFAULT '1' COMMENT '用户状态 1正常 2冻结',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_username_del` (`username`),
  UNIQUE KEY `idx_mobile_del` (`mobile`),
  UNIQUE KEY `idx_feishu_del` (`fei_shu_user_id`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2025-01-26 10:52:57
