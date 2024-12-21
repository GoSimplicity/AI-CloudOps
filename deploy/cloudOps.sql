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

--
-- Table structure for table `apis`
--

DROP TABLE IF EXISTS `apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `apis` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(50) NOT NULL COMMENT 'API名称',
  `path` varchar(255) NOT NULL COMMENT 'API路径',
  `method` tinyint(1) NOT NULL COMMENT 'HTTP请求方法(1:GET,2:POST,3:PUT,4:DELETE)',
  `description` varchar(500) DEFAULT NULL COMMENT 'API描述',
  `version` varchar(20) DEFAULT 'v1' COMMENT 'API版本',
  `category` tinyint(1) NOT NULL COMMENT 'API分类(1:系统,2:业务)',
  `is_public` tinyint(1) DEFAULT '0' COMMENT '是否公开(0:否,1:是)',
  `create_time` bigint DEFAULT NULL COMMENT '创建时间',
  `update_time` bigint DEFAULT NULL COMMENT '更新时间',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '是否删除(0:否,1:是)',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `apis`
--

LOCK TABLES `apis` WRITE;
/*!40000 ALTER TABLE `apis` DISABLE KEYS */;
/*!40000 ALTER TABLE `apis` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `bind_ecs`
--

LOCK TABLES `bind_ecs` WRITE;
/*!40000 ALTER TABLE `bind_ecs` DISABLE KEYS */;
/*!40000 ALTER TABLE `bind_ecs` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `bind_elb`
--

LOCK TABLES `bind_elb` WRITE;
/*!40000 ALTER TABLE `bind_elb` DISABLE KEYS */;
/*!40000 ALTER TABLE `bind_elb` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `bind_elbs`
--

LOCK TABLES `bind_elbs` WRITE;
/*!40000 ALTER TABLE `bind_elbs` DISABLE KEYS */;
/*!40000 ALTER TABLE `bind_elbs` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `bind_rds`
--

LOCK TABLES `bind_rds` WRITE;
/*!40000 ALTER TABLE `bind_rds` DISABLE KEYS */;
/*!40000 ALTER TABLE `bind_rds` ENABLE KEYS */;
UNLOCK TABLES;

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
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (4,'p','1','/*','DELETE','','',''),(1,'p','1','/*','GET','','',''),(6,'p','1','/*','HEAD','','',''),(7,'p','1','/*','OPTIONS','','',''),(5,'p','1','/*','PATCH','','',''),(2,'p','1','/*','POST','','',''),(3,'p','1','/*','PUT','','','');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `first_upgrade_users`
--

DROP TABLE IF EXISTS `first_upgrade_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `first_upgrade_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_first_upgrade_users_user` (`user_id`),
  CONSTRAINT `fk_first_upgrade_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_first_upgrade_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `first_upgrade_users`
--

LOCK TABLES `first_upgrade_users` WRITE;
/*!40000 ALTER TABLE `first_upgrade_users` DISABLE KEYS */;
/*!40000 ALTER TABLE `first_upgrade_users` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_apps_name` (`name`),
  UNIQUE KEY `idx_k8s_apps_cluster` (`cluster`),
  KEY `idx_k8s_apps_deleted_at` (`deleted_at`),
  KEY `fk_k8s_projects_k8s_apps` (`k8s_project_id`),
  CONSTRAINT `fk_k8s_projects_k8s_apps` FOREIGN KEY (`k8s_project_id`) REFERENCES `k8s_projects` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_apps`
--

LOCK TABLES `k8s_apps` WRITE;
/*!40000 ALTER TABLE `k8s_apps` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_apps` ENABLE KEYS */;
UNLOCK TABLES;

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
  `env` longtext COMMENT '集群环境，例如 prod, stage, dev, rc, press',
  `version` longtext COMMENT '集群版本',
  `api_server_addr` longtext COMMENT 'API Server 地址',
  `kube_config_content` text COMMENT 'kubeConfig 内容',
  `action_timeout_seconds` bigint DEFAULT NULL COMMENT '操作超时时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_k8s_clusters_name` (`name`),
  UNIQUE KEY `idx_k8s_clusters_name_zh` (`name_zh`),
  KEY `idx_k8s_clusters_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_clusters`
--

LOCK TABLES `k8s_clusters` WRITE;
/*!40000 ALTER TABLE `k8s_clusters` DISABLE KEYS */;
INSERT INTO `k8s_clusters` VALUES (1,'2024-12-19 16:18:47.668','2024-12-19 16:18:47.668',0,'Cluster-1','集群-1',1,'100m','200m','256Mi','512Mi','','prod','v1.32.0','https://api.cluster1.example.com','\napiVersion: v1\nclusters:\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJSVVvMDZCMEplSFl3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TkRBNU16QXdPREEyTWpkYUZ3MHpOREE1TWpnd09ERXhNamRhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUUREOW02ZWZQQytMazZ2clkwVEUyMVJieU9WZGMvVmR3c0FMZ011ci9zMlR0MzVYV2I4aFczT3lrQTEKNEVPYWQ5VmNrNExnK1VnMXdDV1ozMTNPbnFmRUFEQ25OWjFiRHJGbCs0Smhya2c4M0pUanlPZStqZmdNOXFWTQptczVoSjhnT3didmQ2WmdaOFl2bHowbHZGU3hEZGdHNXhIZGdqemZYU0FMRlhIS1hweVpWbHpqaWNWT1FNRWlrClM3clBoQUEyTnNLdDljeFVHUkI1OUMzN3poNks5MjdFclNJbUlKalZ0Ynl4MXJaVnBFSW1STGRPcFI2NWZlYnoKU2h6UlVwejVxTGMxZjl5TWlnWVRFY2tRZHFlYXFJYi9aQ1RTWmw3d09ZcHJIT3NianJaR21WN243QkVGU3lsUgplYzcvalRQWTFya3JLaEw2ZWJCeVBEWm1MckJQQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJReUp2SWYremlBQ0xlYzZlUkxvbU5IdG5SUkNqQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQ1o5c0NvT28wOQpVWnRpc2w2TTVLcmh4UFhKcFhmbXdSZ0wxd3doSTVFZWgvNy9pTnBkZlEzVW11Q2RiYWJ2YUUyQnB0bnBzN0dEClM2bytZNGo5dGRpSS9jUEZ0ekhXMVNCWU95SFEwYWptMTlTbkJtempUZzZKRDNTTVJiVVEzT1Q3VUtJVWpTQ0UKSnBRbWJoU2N6aHJKVjN1QnFLR1hzRk1xTDZRcXVaUlI5SVFJRmhzcnhidWp5cnNiZXZJbkhoQUFyWkp3ZnlVQgpTbmVjVkVsMm5NWFF3MjlRMDZIRVJkMWxJcW9KMFpoNzN2MWhmUDAyYmxma0xaL2tIaVE2ZGhLWDQrc0hVams0CnNlaVg2N3JnT2ZrRytBUFNGMEhaVEZqeUpHd3h3ZDFlem9DakdZbTRZc2oxNjUrSXQxZGYzWjN3QStoRFlCWmMKQjVPdDljejh2K0JHCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    server: https://192.168.0.104:6443\n  name: kubernetes\n- cluster:\n    certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURCVENDQWUyZ0F3SUJBZ0lJVWt4L2ZkQnhLbU13RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TkRFeE1EWXdNakEzTlRWYUZ3MHpOREV4TURRd01qRXlOVFZhTUJVeApFekFSQmdOVkJBTVRDbXQxWW1WeWJtVjBaWE13Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLCkFvSUJBUURZaDA4RFF3S2lhTWtMUHBNbElHQXArS21KVERLYnBLa2Y3NTNxS1VOeDdmRGVDU3lZWlpKK1VSRUoKUEsvKy9BMTFtdUw1cnZTSVplZDE4MC9xSGhUZGxYL3dQSlVMeG1pN1hNWHU2TG01SW95a1ZzTUJlTXhNUUVnSQovU01vdnZhdTZZY3Rhd2JCSi9ZME11MDkrNS9rTVAvTXdDVG9seGlETkJnTzZLZVJidEYyTWJLc0VxWTBrc3A5CkpnNm0xdUxUZEk2YTlTZ00yVUc1YUJZSzR5MTJ0WWFtY2pmVVZiSGxkR3lQNU9YeDY4RDNFNnI5NHhVUFVjR2kKc3BFekxMZkZQWnFmZGg3bDJjUEtnY3BiV1FPRklva0dKaGVJT1lPZ3pFOFZpbG0wS1dBbmhwWkFkQXZIbnVwRApNaUpHaSt1ejRTejJ3N2dJRzZPeXA1MkVyODVmQWdNQkFBR2pXVEJYTUE0R0ExVWREd0VCL3dRRUF3SUNwREFQCkJnTlZIUk1CQWY4RUJUQURBUUgvTUIwR0ExVWREZ1FXQkJSdGpIa0trL0I2bVRsNUc4ZFJMZEFReVN3QlRqQVYKQmdOVkhSRUVEakFNZ2dwcmRXSmxjbTVsZEdWek1BMEdDU3FHU0liM0RRRUJDd1VBQTRJQkFRQ2R4WDE4VWhiTgpBcnE2UVg2K1B6WEFTS2FWWFBsWVNxV3VTS1dYUm1ZTjhvTnJZenpMZGZ0TGxmbWExWUZqaFpKaDY5Z2dTUlZ5CjlQTGU0OHRETDhFSDVMNFFOTDAwL2tZcThyaEtGei9lVC9TY1dITUMyUGZKaHM2Q3B5QzVtS3h5b0JGTE9rZlQKQ2hVOTlIalhkU0NmUmw1TXpGQWdSRVpQWWhhVHRRQWR5clZvcTdSUXZ2aFNWcVBBTmNobWJtMU5aVGIrVjhYSgo3SzFQUngrVmoraEFCRXhuSGR6d1BsYUdUVTJ0cjJUSTNGRjhkejZPdTVuV3E3aEdiNUl5Vk9RZ09QdHh0MW9UCko4ZG5aU0RDNWxjVnZtRzVQdElHcFFtYVkwMWtzMC8xejQ5ZXhEbllwb3dYMlhXZWk2eW5MU08wOEpjUHJKSHYKTGY2MlpQQUNoOU1WCi0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    server: https://127.0.0.1:59623\n  name: kind-kind\ncontexts:\n- context:\n    cluster: kubernetes\n    user: kubernetes-admin\n  name: kubernetes-admin@kubernetes\n- context:\n    cluster: kind-kind\n    user: kind-kind\n  name: kind-kind\ncurrent-context: kind-kind\nkind: Config\npreferences: {}\nusers:\n- name: kubernetes-admin\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURJVENDQWdtZ0F3SUJBZ0lJR3VmQ1haOEJYNzR3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TkRBNU16QXdPREEyTWpkYUZ3MHlOVEE1TXpBd09ERXhNekZhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQXlFdmFXMURaTkpHUE01ZkMKb1E4Wnplb0I2Nm5DQ3J3VndpM3I4blJwZG5WY0xBNGdNWVlXZnVtWm5IMEgxWEplUmdtanhGYUVDUDM1aEtQegphLzZMSDkvTnkrK2NJZXFGc1Q3NVZ2bGc3NWZVQ0Fvd0VoU3JQSHVZSDllTUVKRlV1d2xkcmZUdGxQaVFoM0MwCnhxd2lORkVSbXRmTnIrWjJTeXFLczAydmFtOWpuZW5Jdk90MzJMdWFmVWxwUHdGaGR0bkwrdTI3WXJVcTVSZDEKazhxSTBuS0NJdTVQeGxORUlyQnpEN29SbldRZEt1U0tyZFQyemI0dHdjRWlDckpLRzMwQVRZbjFOcGF5RW03egpZVkdTbjRhOFRCV1AzSUxpWDZhYjkwbEVWTGU3b3J2dmVrbTFlbzA0eDZoRWJETHlkaTg4amFsZnRjcjBqK1hlCkw1RS9hUUlEQVFBQm8xWXdWREFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0RBWURWUjBUQVFIL0JBSXdBREFmQmdOVkhTTUVHREFXZ0JReUp2SWYremlBQ0xlYzZlUkxvbU5IdG5SUgpDakFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBdkZBcElheW9TT0I4Y0oyMWp2TE1pMldmY20zRUtteG1GajFZCjYwV045TXlHTlRwM2dEbVpUWmlKRGN3TXYvNlNZUG5KalMwTVpMb3dmQkZoOXhGM3BnUEJKMEF2UkNkcGl0aFoKelVYYlZZQlkzK1p0bHBwenU3aU9WVllkbkg0ZW9tTTJJS2VDUHpTamNScloxbk52V3QrYTk3ejU2T0hOcDdVcgpkZEFXN1pRQ21LT1ZGTzVvWlk3QkdmVTNpTzJvN1o4OURvT01RSzZjMElKSHBRNkxqQTNrdHA4YVBFKytsZ2IwCjNEcjEzRmVGUHQyNmYvL1Zwd1laMjlGN0JIeUFUK2F1SWphTy9MTGdLVUt4STdrQkFBRXZuempQeXp2QnlHbzkKcE9HU0ZuRE5yTVlOWmdpWWZnbTBrc3dLaWpIdmxQaXNFanhYaUUrWFpnQzRFOXA0WkE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==\n    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBeUV2YVcxRFpOSkdQTTVmQ29ROFp6ZW9CNjZuQ0Nyd1Z3aTNyOG5ScGRuVmNMQTRnCk1ZWVdmdW1abkgwSDFYSmVSZ21qeEZhRUNQMzVoS1B6YS82TEg5L055KytjSWVxRnNUNzVWdmxnNzVmVUNBb3cKRWhTclBIdVlIOWVNRUpGVXV3bGRyZlR0bFBpUWgzQzB4cXdpTkZFUm10Zk5yK1oyU3lxS3MwMnZhbTlqbmVuSQp2T3QzMkx1YWZVbHBQd0ZoZHRuTCt1MjdZclVxNVJkMWs4cUkwbktDSXU1UHhsTkVJckJ6RDdvUm5XUWRLdVNLCnJkVDJ6YjR0d2NFaUNySktHMzBBVFluMU5wYXlFbTd6WVZHU240YThUQldQM0lMaVg2YWI5MGxFVkxlN29ydnYKZWttMWVvMDR4NmhFYkRMeWRpODhqYWxmdGNyMGorWGVMNUUvYVFJREFRQUJBb0lCQVFDcm9ETnVZNHg2ZXU5VgpxZ2hmc1d6UEFHQzg2aTBXdmF0M1E1b1ZtcUp6bW9Sc1MzNVNjUzc5ZUhUam5rOEVHb2VsUThWTUMwWC8zbi9iCnBCQ0V6UXV4T0RoRE13RjZIbGFJVmdtWStQNlN6bW9rcVhZZlNBNmlPTlZWRTRFMUFSSzFZWVVmOWV0TjV0OFEKN3dZMzVtODRuTzZVMjYybnQ3Wk5HaHJYSVEzYUNDb3pVNktwRkNjUGJtMWZaY0ZseDBzMkFpTCttVHhCQ3BJdQo5R2pJUS9sLy84QUNPcWx5Wi91VUFEa2tPQ1FtQTNvNEVVeEdaaGx5WXpJYkRpOEhXYWtLY3Jkc0p4T05MMk9GCkVYbTlrdXB4aUR1bmVLMHdLU0oxY1U5QWhjNGxqakRjWDcwN1lJaTVtNFhEc3FxYmt6MGoxVEl4TG1maDVhVzgKQTNxRlhZQnhBb0dCQVBwVlA5RjFMNStPU094N01rQjZLMHluTjNvUUhPaDM0a05BVU1LUTJ4YWlXZGs0TXdGago4WmMvSWdra004ZHA0YWhLM1JsS3lFL1J3YnJNYW5BSkR4QlZFeVgydi95UUFGNDhsbEM3UjU5T296aG10ZU0zCnR3Y0hvSHRWaUhUR29ZeXdGOE9pbk9udVJFUm1semJYT3pENDBoYWFKaCtNNnJCTFJBMDhhSHd6QW9HQkFNelUKb0hUY1h6cW0zNzAxTUUwMFVrWlpjb281SzR3L0lSVGIrc3d3Mk8razYvZE13QnA3bkxZQnBSTVZWTlhIbU5rVAorNGl4U04xYnpzU3gra0RIWDdmZWQybEFUK2g1cEpvcVV2WHAzcWtpWUtOSGxlSkFKYkEwM2pudS9kQUlORkNQCnNpWnVST1JmeW1nNnFYTXg0NXBjbnh6cUFuUGFzaXVrdE5MY1ZEbnpBb0dCQU8yOVNINkQ3Rlo3cW9YcitpMkIKMk4xVGNUeGJVUmoxd2N4Y3FGWWZlL0ppL1RGdVRnSmtDR3k3YUhlR0NpYTRSN2FzWW81Q2x6bzIydVdzZk9rcApzVVN4aHgzbTJTM2pGSFpxMDlhWUJjMGx3WjB1N2s1Nyt6YVI1N2M1NC80REppbVdrdnNZMUN6V083ODZMeUhHCkJsRGIvYW01ZTdzNitTZTBVMHkrc2Z4QkFvR0JBSU16czNBSGRLeENGZERCa0NYejNMdVpNZ2dkNUtvYUNkdXQKcUxGQW5NU3NORVdkRVBRbHQ5VFJxdVpWWkpqbkdCMzhjY00ySkFFK2ZHeDd3RnZjR1pEU1hGUzcwRE9PTDRSYwpsZlZWRDczdytrdThYK0tqeWtCYkxQbVkvMVZRM0FtNmNaZXlURWlvbnlNeWFEWVVmOER4a1MzWkt5Y0FyOTNLCnk5VEJNdVpIQW9HQVdYc1VGK25QbWdHc1l5bHpUSzl6NStxNmloelI1enhvNUVCd2ZHTlpBRXBSMVphSHdVcXIKWGVzOFBSWWp0c3FORTBsdWJNOGs1QS9hM2Q4QzhBdU5IdjBXSFRTQUVlRUlqeWkxeTVoakJyaklhWXdPNGZmTgpOK0I3Rm9OOVJQbDdUWHRkckNmck1hbklYU3JLOXdKMlpXRm1LaWNJT1lVUzBZYlNaOHh2emhBPQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=\n- name: kind-kind\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURLVENDQWhHZ0F3SUJBZ0lJRHNFcTVnRyt1VkV3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TkRFeE1EWXdNakEzTlRWYUZ3MHlOVEV4TURZd01qRXlOVFZhTUR3eApIekFkQmdOVkJBb1RGbXQxWW1WaFpHMDZZMngxYzNSbGNpMWhaRzFwYm5NeEdUQVhCZ05WQkFNVEVHdDFZbVZ5CmJtVjBaWE10WVdSdGFXNHdnZ0VpTUEwR0NTcUdTSWIzRFFFQkFRVUFBNElCRHdBd2dnRUtBb0lCQVFERDdMQTIKdUVDcE5meDBDdGE3RmFHWTF1YjF4cTdoRGdGTVhvTHk2NkxKbXBKT3lYUllHUFU1LzJHUlRId1UvRnVzSWYvQgphN2RzRjhWL0MvZStlKzZUR3BLOEZBR3ZsTmp4dkpEaENOVlRSTTR0bXpSR2lTTXo5cTJtMURWcW5GMDNDTjBzCjhXZXJIWmdjNVIyQ3A2WkYyMUdXTDIxVWU0UENvN2k5alZkNXJQNERHTWRCTkFYSVlGaXg1VWE2akd5NkpncTgKWXNDTms1Wko4Z0tIOFFIa0VtckZzWmV1NEI4MmxGUkdpWHN6Sjc2VVdvSE1TNWlJWGtSNWlrUmNJVEFvUFVxRgoyZEtiSnZxODN5RFdQVzVrVDNoMmh1Ri9ZWkppQ0xwVDJlYmRQSUxjR1lSczd0dlhlbXpBdXpVaU8xcUlBeitsClAvaSsyQmNNRDFQY0FQaUpBZ01CQUFHalZqQlVNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUsKQmdnckJnRUZCUWNEQWpBTUJnTlZIUk1CQWY4RUFqQUFNQjhHQTFVZEl3UVlNQmFBRkcyTWVRcVQ4SHFaT1hrYgp4MUV0MEJESkxBRk9NQTBHQ1NxR1NJYjNEUUVCQ3dVQUE0SUJBUURHWHJuUWduN2I4elJlYlZ4clpGTndpWktPCmFYaDZKTVNIbm0wNm9iYThkMDR2YVVCbzZwT01lOE9PNTcvclRNcWR4cXhFNTAxMlY0eXZBNng4RzgzY01KdDgKaGNkQ2ZlcjFQbE9YZ1RaVlBnV2hiVHJJSjc2M3hUQm1IOEJYT3hsQ1p4ZXAzS2xZWk9nSGxpeFJtRnYwZklMQgpwdERCdGE4R3BUcnRqaFZ5eWxHMjA4ajJwK3oxOE5Sc3A2bHdmRVZnenRFbTBFeFM2MFA5VjY1ZzhoR2M1UVpoCm5RSVhTSDVBZGxVYTYrajRuSndPOG01Rmk4NGFMbHFzR3czalJCeHZnUkxYUk81clBXWHpJL1NrL3ZpcytlTDEKbTVnMFVmQXJLQU4wc096bTFoTGR4YVZ5QUZUVU5XS204aTJCd0U3SXk2bXJ6VnZMVWJyVjNpR2s3aEZ6Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBdyt5d05yaEFxVFg4ZEFyV3V4V2htTmJtOWNhdTRRNEJURjZDOHV1aXlacVNUc2wwCldCajFPZjloa1V4OEZQeGJyQ0gvd1d1M2JCZkZmd3Yzdm52dWt4cVN2QlFCcjVUWThieVE0UWpWVTBUT0xaczAKUm9rak0vYXRwdFExYXB4ZE53amRMUEZucXgyWUhPVWRncWVtUmR0UmxpOXRWSHVEd3FPNHZZMVhlYXorQXhqSApRVFFGeUdCWXNlVkd1b3hzdWlZS3ZHTEFqWk9XU2ZJQ2gvRUI1QkpxeGJHWHJ1QWZOcFJVUm9sN015ZStsRnFCCnpFdVlpRjVFZVlwRVhDRXdLRDFLaGRuU215YjZ2TjhnMWoxdVpFOTRkb2JoZjJHU1lnaTZVOW5tM1R5QzNCbUUKYk83YjEzcHN3THMxSWp0YWlBTS9wVC80dnRnWERBOVQzQUQ0aVFJREFRQUJBb0lCQVFDYzdxTWUwV3NKbm5KKwpTSWhEQmtxUDcrTERqc2RaQVN6TkRRNzZvUCtkV0RCRTUxeEhqSVl3Vkh6RU0yMVlLZU1MOTVleVNDTjljM1VBCkZJZjJqYkpGSmczT2xIL2RNZTZyZ296Ums0Kzd5T3NVNExKNHBUUUxWVlUyd2RlZmMydSt2MXpadU90K3hvK20KNVdaRDF5RjU1dmhzd2NSaTNTUm03VmoyaTVZN29JeGhvVjl6S2dXc2VBbllkeXgzU3Z5WTBXaGZ6eENuSTUxQgovVUVTd1hDak5LVUdXUDZSNElDTDYwTGw1Z1RoOWtrMTh1S1BrWWJLSDlmRGw2MjNBTC9tSkZpZ3g1NUxJL25NCkhORUpBMjdYcnhUTCtVK2M3TXdvd0tKM2o0V210ZkZOdDFZeHhjS3l2OXB0R1FPRUVTY25oRG9zNDNkMDdaMGIKaWovQ1RIN2hBb0dCQU9UWE9EUWlrYUsrcXV3NVdQZTU2djZXWk1QeGdtQzFCbThOOGJNQm9aT2xVS3pUbS9nSgpuSFVaMWYyQ1V4UmYxTysycUJwa0RmRmw4MmZVVTBzNk0zZXc4WHNyZXdFK01MYmUrcjc3UWFaVVAvQjAyYW1UCkRwazQrakl6K0lWSGZIbXVTWkIvd08zZTNHQ1ZPeUIxb2p6SnBJYXRTdWdXU243M3ljVDFXOW10QW9HQkFOc3QKWStaMTdrRldVdVJ5T1F4WVJMYkxYbWpIQ1V3OWlhdFk3L1ZtL1VLcDB3MmxlSDJHQ2NRRCtIYzliS3FTeDJWSQpoRWIwZ1dPZldNeUpES0w2QVZtVzR2SkdXR0tSeVJ5Unl0ZU5PZk56V2dReFVtcFkrZDZzRFZYN0dacXJ5UWRNClZtODVpaTBENk1OcCtXUkpadHczWlBkZHNwOWhIMzhWeVdGUGVXM05Bb0dBR0gzSU1CdzdCZlh1Q1JZaUpYRXEKYTFEaE8rOVBDdGFVOTdIQVdtNGtRczhBa1Y2Y1pMRnlvejIrbjBFaGJ4N0toVlZCTElIazFCOGJLOU9Yam9lTwpGcE5EWlBGRVd0K3pDdjlXU3JaTlVtWFY2Z0EzZzJTUHZXcFJyS25QUVVSaldBcUZLUWZqT0JJUDkrNUF3N3FUClFIbzhONFc0YkpwbUlxeVdWWlFFM29rQ2dZQTNlTzUrNXJ3dGh6YWxvUTgxUTZYb2llSlVMSVA2NnR4TUpNOWUKMGZrcGhTZm9uVWU0cFZNVmJGZlhmaEZodnBKKzNQSzFycTZNMDBpN1E3aVNDeXFLVFRrVlRwNlNIQW5GbEZTOQpaMzRTVXRDbW5RRVo3M2tXVlg5dWtvWHhjcWNIbE5lUGdRV3F6UUY5YS9YMTN1b010R3gyZXgxNVh6Q0VqclFRClQvZ1F4UUtCZ1FES1FkSzhFalFKc3lWTUJ1ZmlJdGdaalVZK0h3YUFoMC9NeHNvZmxHYWJiUkNnckdFaUpwRmkKYVc0Y1M2dUV1SVlDcStZRVo5b3FJOHd2WUlMOGU0bGwwV1BCTDQwL2pqZFkrSmVzYmNmbmIxMW5vVCtOUXdqSQpYdWh1VER4WDBuVjJXNGE4Ni9vaUtQQnh1ZmFwSEhrWGRTckJjYmcyODFQRTJZd1VLYVQzWVE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=\n',30);
/*!40000 ALTER TABLE `k8s_clusters` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_cronjobs_name` (`name`),
  UNIQUE KEY `idx_k8s_cronjobs_k8s_project_id` (`k8s_project_id`),
  KEY `idx_k8s_cronjobs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_cronjobs`
--

LOCK TABLES `k8s_cronjobs` WRITE;
/*!40000 ALTER TABLE `k8s_cronjobs` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_cronjobs` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_instances_name` (`name`),
  UNIQUE KEY `idx_k8s_instances_k8s_app_id` (`k8s_app_id`),
  KEY `idx_k8s_instances_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_k8s_apps_k8s_instances` FOREIGN KEY (`k8s_app_id`) REFERENCES `k8s_apps` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_instances`
--

LOCK TABLES `k8s_instances` WRITE;
/*!40000 ALTER TABLE `k8s_instances` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_instances` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `k8s_pods`
--

LOCK TABLES `k8s_pods` WRITE;
/*!40000 ALTER TABLE `k8s_pods` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_pods` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_projects_name` (`name`),
  UNIQUE KEY `idx_k8s_projects_name_zh` (`name_zh`),
  UNIQUE KEY `idx_k8s_projects_cluster` (`cluster`),
  KEY `idx_k8s_projects_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_projects`
--

LOCK TABLES `k8s_projects` WRITE;
/*!40000 ALTER TABLE `k8s_projects` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_projects` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_yaml_tasks_name` (`name`),
  KEY `idx_k8s_yaml_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_yaml_tasks`
--

LOCK TABLES `k8s_yaml_tasks` WRITE;
/*!40000 ALTER TABLE `k8s_yaml_tasks` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_yaml_tasks` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_k8s_yaml_templates_name` (`name`),
  KEY `idx_k8s_yaml_templates_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `k8s_yaml_templates`
--

LOCK TABLES `k8s_yaml_templates` WRITE;
/*!40000 ALTER TABLE `k8s_yaml_templates` DISABLE KEYS */;
/*!40000 ALTER TABLE `k8s_yaml_templates` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `menus`
--

DROP TABLE IF EXISTS `menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `menus` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
  `name` varchar(50) NOT NULL COMMENT '菜单显示名称',
  `parent_id` bigint DEFAULT '0' COMMENT '上级菜单ID,0表示顶级菜单',
  `path` varchar(255) NOT NULL COMMENT '前端路由访问路径',
  `component` varchar(255) NOT NULL COMMENT '前端组件文件路径',
  `route_name` varchar(50) NOT NULL COMMENT '前端路由名称,需唯一',
  `hidden` tinyint(1) DEFAULT '0' COMMENT '菜单是否隐藏(0:显示 1:隐藏)',
  `redirect` varchar(255) DEFAULT '' COMMENT '重定向路径',
  `meta` json DEFAULT NULL COMMENT '菜单元数据',
  `create_time` bigint DEFAULT NULL COMMENT '记录创建时间戳',
  `update_time` bigint DEFAULT NULL COMMENT '记录最后更新时间戳',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '逻辑删除标记(0:未删除 1:已删除)',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=33 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `menus`
--

LOCK TABLES `menus` WRITE;
/*!40000 ALTER TABLE `menus` DISABLE KEYS */;
INSERT INTO `menus` VALUES (1,'Dashboard',0,'/','BasicLayout','',0,'','{\"icon\": \"\", \"order\": -1, \"title\": \"page.dashboard.title\"}',1734596327,1734596327,0),(2,'Welcome',1,'/system_welcome','/dashboard/SystemWelcome','',0,'','{\"icon\": \"lucide:area-chart\", \"title\": \"欢迎页\", \"affixTab\": true}',1734596327,1734596327,0),(3,'用户管理',1,'/system_user','/dashboard/SystemUser','',0,'','{\"icon\": \"lucide:user\", \"title\": \"用户管理\"}',1734596327,1734596327,0),(4,'菜单管理',1,'/system_menu','/dashboard/SystemMenu','',0,'','{\"icon\": \"lucide:menu\", \"title\": \"菜单管理\"}',1734596327,1734596327,0),(5,'接口管理',1,'/system_api','/dashboard/SystemApi','',0,'','{\"icon\": \"lucide:zap\", \"title\": \"接口管理\"}',1734596327,1734596327,0),(6,'权限管理',1,'/system_permission','','',0,'','{\"icon\": \"lucide:shield\", \"title\": \"权限管理\"}',1734596327,1734596327,0),(7,'角色权限',6,'/system_role','/dashboard/SystemRole','',0,'','{\"icon\": \"lucide:users\", \"title\": \"角色权限\"}',1734596327,1734596327,0),(8,'用户权限',6,'/system_user_role','/dashboard/SystemUserRole','',0,'','{\"icon\": \"lucide:user-cog\", \"title\": \"用户权限\"}',1734596327,1734596327,0),(9,'ServiceTree',0,'/tree','BasicLayout','',0,'','{\"icon\": \"\", \"order\": 1, \"title\": \"page.serviceTree.title\"}',1734596327,1734596327,0),(10,'服务树概览',9,'/tree_overview','/servicetree/TreeOverview','',0,'','{\"icon\": \"material-symbols:overview\", \"title\": \"服务树概览\"}',1734596327,1734596327,0),(11,'服务树节点管理',9,'/tree_node_manager','/servicetree/TreeNodeManager','',0,'','{\"icon\": \"fluent-mdl2:task-manager\", \"title\": \"服务树节点管理\"}',1734596327,1734596327,0),(12,'ECS管理',9,'/ecs_resource_operation','/servicetree/ECSResourceOperation','',0,'','{\"icon\": \"mdi:cloud-cog-outline\", \"title\": \"ECS管理\"}',1734596327,1734596327,0),(13,'Prometheus',0,'/prometheus','BasicLayout','',0,'','{\"icon\": \"\", \"order\": 2, \"title\": \"Promethues管理\"}',1734596327,1734596327,0),(14,'MonitorScrapePool',13,'/monitor_pool','/promethues/MonitorScrapePool','',0,'','{\"icon\": \"lucide:database\", \"title\": \"采集池\"}',1734596327,1734596327,0),(15,'MonitorScrapeJob',13,'/monitor_job','/promethues/MonitorScrapeJob','',0,'','{\"icon\": \"lucide:list-check\", \"title\": \"采集任务\"}',1734596327,1734596327,0),(16,'MonitorAlert',13,'/monitor_alert','/promethues/MonitorAlert','',0,'','{\"icon\": \"lucide:alert-triangle\", \"title\": \"alert告警池\"}',1734596327,1734596327,0),(17,'MonitorAlertRule',13,'/monitor_alert_rule','/promethues/MonitorAlertRule','',0,'','{\"icon\": \"lucide:badge-alert\", \"title\": \"告警规则\"}',1734596327,1734596327,0),(18,'MonitorAlertEvent',13,'/monitor_alert_event','/promethues/MonitorAlertEvent','',0,'','{\"icon\": \"lucide:bell-ring\", \"title\": \"告警事件\"}',1734596327,1734596327,0),(19,'MonitorAlertRecord',13,'/monitor_alert_record','/promethues/MonitorAlertRecord','',0,'','{\"icon\": \"lucide:box\", \"title\": \"预聚合\"}',1734596327,1734596327,0),(20,'MonitorConfig',13,'/monitor_config','/promethues/MonitorConfig','',0,'','{\"icon\": \"lucide:file-text\", \"title\": \"配置文件\"}',1734596327,1734596327,0),(21,'MonitorOnDutyGroup',13,'/monitor_onduty_group','/promethues/MonitorOnDutyGroup','',0,'','{\"icon\": \"lucide:user-round-minus\", \"title\": \"值班组\"}',1734596327,1734596327,0),(22,'MonitorOnDutyGroupTable',13,'/monitor_onduty_group_table','/promethues/MonitorOndutyGroupTable','',0,'','{\"icon\": \"material-symbols:table-outline\", \"title\": \"排班表\", \"hideInMenu\": true}',1734596327,1734596327,0),(23,'MonitorSend',13,'/monitor_send','/promethues/MonitorSend','',0,'','{\"icon\": \"lucide:send-horizontal\", \"title\": \"发送组\"}',1734596327,1734596327,0),(24,'K8s',0,'/k8s','BasicLayout','',0,'','{\"icon\": \"\", \"order\": 3, \"title\": \"k8s运维管理\"}',1734596327,1734596327,0),(25,'K8sCluster',24,'/k8s_cluster','/k8s/K8sCluster','',0,'','{\"icon\": \"lucide:database\", \"title\": \"集群管理\"}',1734596327,1734596327,0),(26,'K8sNode',24,'/k8s_node','/k8s/K8sNode','',0,'','{\"icon\": \"lucide:list-check\", \"title\": \"节点管理\", \"hideInMenu\": true}',1734596327,1734596327,0),(27,'K8sPod',24,'/k8s_pod','/k8s/K8sPod','',0,'','{\"icon\": \"lucide:bell-ring\", \"title\": \"Pod管理\"}',1734596327,1734596327,0),(28,'K8sService',24,'/k8s_service','/k8s/K8sService','',0,'','{\"icon\": \"lucide:box\", \"title\": \"Service管理\"}',1734596327,1734596327,0),(29,'K8sDeployment',24,'/k8s_deployment','/k8s/K8sDeployment','',0,'','{\"icon\": \"lucide:file-text\", \"title\": \"Deployment管理\"}',1734596327,1734596327,0),(30,'K8sConfigMap',24,'/k8s_configmap','/k8s/K8sConfigmap','',0,'','{\"icon\": \"lucide:user-round-minus\", \"title\": \"ConfigMap管理\"}',1734596327,1734596327,0),(31,'K8sYamlTemplate',24,'/k8s_yaml_template','/k8s/K8sYamlTemplate','',0,'','{\"icon\": \"material-symbols:table-outline\", \"title\": \"Yaml模板\"}',1734596327,1734596327,0),(32,'K8sYamlTask',24,'/k8s_yaml_task','/k8s/K8sYamlTask','',0,'','{\"icon\": \"lucide:send-horizontal\", \"title\": \"Yaml任务\"}',1734596327,1734596327,0);
/*!40000 ALTER TABLE `menus` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_events`
--

DROP TABLE IF EXISTS `monitor_alert_events`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_events` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `alert_name` varchar(200) DEFAULT NULL COMMENT '告警名称',
  `fingerprint` varchar(100) DEFAULT NULL COMMENT '告警唯一ID',
  `status` varchar(50) DEFAULT NULL COMMENT '告警状态（如告警中、已屏蔽、已认领、已恢复）',
  `rule_id` bigint DEFAULT NULL COMMENT '关联的告警规则ID',
  `send_group_id` bigint DEFAULT NULL COMMENT '关联的发送组ID',
  `event_times` bigint DEFAULT NULL COMMENT '触发次数',
  `silence_id` varchar(100) DEFAULT NULL COMMENT 'AlertManager返回的静默ID',
  `ren_ling_user_id` bigint DEFAULT NULL COMMENT '认领告警的用户ID',
  `labels` text COMMENT '标签组，格式为 key=v',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_alert_events_fingerprint` (`fingerprint`),
  KEY `idx_monitor_alert_events_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_events`
--

LOCK TABLES `monitor_alert_events` WRITE;
/*!40000 ALTER TABLE `monitor_alert_events` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_alert_events` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_manager_pools`
--

DROP TABLE IF EXISTS `monitor_alert_manager_pools`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_manager_pools` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT 'AlertManager实例名称，支持使用通配符*进行模糊搜索',
  `alert_manager_instances` text COMMENT '选择多个AlertManager实例',
  `user_id` bigint DEFAULT NULL COMMENT '创建该实例池的用户ID',
  `resolve_timeout` varchar(50) DEFAULT NULL COMMENT '默认恢复时间',
  `group_wait` varchar(50) DEFAULT NULL COMMENT '默认分组第一次等待时间',
  `group_interval` varchar(50) DEFAULT NULL COMMENT '默认分组等待间隔',
  `repeat_interval` varchar(50) DEFAULT NULL COMMENT '默认重复发送时间',
  `group_by` text COMMENT '分组的标签',
  `receiver` varchar(100) DEFAULT NULL COMMENT '兜底接收者',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_alert_manager_pools_name` (`name`),
  KEY `idx_monitor_alert_manager_pools_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_manager_pools`
--

LOCK TABLES `monitor_alert_manager_pools` WRITE;
/*!40000 ALTER TABLE `monitor_alert_manager_pools` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_alert_manager_pools` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_alert_rules`
--

DROP TABLE IF EXISTS `monitor_alert_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_alert_rules` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '告警规则名称，支持通配符*进行模糊搜索',
  `user_id` bigint DEFAULT NULL COMMENT '创建该告警规则的用户ID',
  `pool_id` bigint DEFAULT NULL COMMENT '关联的Prometheus实例池ID',
  `send_group_id` bigint DEFAULT NULL COMMENT '关联的发送组ID',
  `tree_node_id` bigint DEFAULT NULL COMMENT '绑定的树节点ID',
  `enable` bigint DEFAULT NULL COMMENT '是否启用告警规则：1启用，2禁用',
  `expr` text COMMENT '告警规则表达式',
  `severity` varchar(50) DEFAULT NULL COMMENT '告警级别，如critical、warning',
  `grafana_link` text COMMENT 'Grafana大盘链接',
  `for_time` varchar(50) DEFAULT NULL COMMENT '持续时间，达到此时间才触发告警',
  `labels` text COMMENT '标签组，格式为 key=v',
  `annotations` text COMMENT '注解，格式为 key=v',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_alert_rules_name` (`name`),
  KEY `idx_monitor_alert_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_alert_rules`
--

LOCK TABLES `monitor_alert_rules` WRITE;
/*!40000 ALTER TABLE `monitor_alert_rules` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_alert_rules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_on_duty_changes`
--

DROP TABLE IF EXISTS `monitor_on_duty_changes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_changes` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID，用于标识值班历史记录',
  `user_id` bigint DEFAULT NULL COMMENT '创建该换班记录的用户ID',
  `date` longtext COMMENT '计划哪一天进行换班的日期',
  `origin_user_id` bigint DEFAULT NULL COMMENT '换班前原定的值班人员用户ID',
  `on_duty_user_id` bigint DEFAULT NULL COMMENT '换班后值班人员的用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_on_duty_changes_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_on_duty_changes`
--

LOCK TABLES `monitor_on_duty_changes` WRITE;
/*!40000 ALTER TABLE `monitor_on_duty_changes` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_on_duty_changes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_on_duty_groups`
--

DROP TABLE IF EXISTS `monitor_on_duty_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '值班组名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索',
  `user_id` bigint DEFAULT NULL COMMENT '创建该值班组的用户ID',
  `shift_days` bigint DEFAULT NULL COMMENT '轮班周期，以天为单位',
  `yesterday_normal_duty_user_id` bigint DEFAULT NULL COMMENT '昨天的正常排班值班人ID，由cron任务设置',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_on_duty_groups_name` (`name`),
  KEY `idx_monitor_on_duty_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_on_duty_groups`
--

LOCK TABLES `monitor_on_duty_groups` WRITE;
/*!40000 ALTER TABLE `monitor_on_duty_groups` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_on_duty_groups` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_on_duty_histories`
--

DROP TABLE IF EXISTS `monitor_on_duty_histories`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_on_duty_histories` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID，用于标识值班历史记录',
  `date_string` varchar(50) DEFAULT NULL COMMENT '日期',
  `on_duty_user_id` bigint DEFAULT NULL COMMENT '当天值班人员的用户ID',
  `origin_user_id` bigint DEFAULT NULL COMMENT '原计划的值班人员用户ID',
  PRIMARY KEY (`id`),
  KEY `idx_monitor_on_duty_histories_deleted_at` (`deleted_at`),
  KEY `idx_monitor_on_duty_histories_on_duty_group_id` (`on_duty_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_on_duty_histories`
--

LOCK TABLES `monitor_on_duty_histories` WRITE;
/*!40000 ALTER TABLE `monitor_on_duty_histories` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_on_duty_histories` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `monitor_on_duty_users`
--

LOCK TABLES `monitor_on_duty_users` WRITE;
/*!40000 ALTER TABLE `monitor_on_duty_users` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_on_duty_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_record_rules`
--

DROP TABLE IF EXISTS `monitor_record_rules`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_record_rules` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '记录规则名称，支持使用通配符*进行模糊搜索',
  `record_name` varchar(500) DEFAULT NULL COMMENT '记录名称，支持使用通配符*进行模糊搜索',
  `user_id` bigint DEFAULT NULL COMMENT '创建该记录规则的用户ID',
  `pool_id` bigint DEFAULT NULL COMMENT '关联的Prometheus实例池ID',
  `tree_node_id` bigint DEFAULT NULL COMMENT '绑定的树节点ID',
  `enable` bigint DEFAULT NULL COMMENT '是否启用记录规则：1启用，2禁用',
  `for_time` varchar(50) DEFAULT NULL COMMENT '持续时间，达到此时间才触发记录规则',
  `expr` text COMMENT '记录规则表达式',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_record_rules_name` (`name`),
  UNIQUE KEY `idx_monitor_record_rules_record_name` (`record_name`),
  KEY `idx_monitor_record_rules_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_record_rules`
--

LOCK TABLES `monitor_record_rules` WRITE;
/*!40000 ALTER TABLE `monitor_record_rules` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_record_rules` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_scrape_jobs`
--

DROP TABLE IF EXISTS `monitor_scrape_jobs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_scrape_jobs` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '采集任务名称，支持使用通配符*进行模糊搜索',
  `user_id` bigint DEFAULT NULL COMMENT '任务关联的用户ID',
  `enable` bigint DEFAULT NULL COMMENT '是否启用采集任务：1为启用，2为禁用',
  `service_discovery_type` varchar(50) DEFAULT NULL COMMENT '服务发现类型，支持 k8s 或 http',
  `metrics_path` varchar(255) DEFAULT NULL COMMENT '监控采集的路径',
  `scheme` varchar(10) DEFAULT NULL COMMENT '监控采集的协议方案（如 http 或 https）',
  `scrape_interval` bigint DEFAULT '30' COMMENT '采集的时间间隔（秒）',
  `scrape_timeout` bigint DEFAULT '10' COMMENT '采集的超时时间（秒）',
  `pool_id` bigint DEFAULT NULL COMMENT '关联的采集池ID',
  `relabel_configs_yaml_string` text COMMENT 'relabel配置的YAML字符串',
  `refresh_interval` bigint DEFAULT NULL COMMENT '刷新目标的时间间隔（针对服务树http类型，秒）',
  `port` bigint DEFAULT NULL COMMENT '端口号（针对服务树服务发现接口）',
  `tree_node_ids` text COMMENT '服务树接口绑定的树节点ID列表，用于获取IP列表',
  `kube_config_file_path` varchar(255) DEFAULT NULL COMMENT '连接apiServer的Kubernetes配置文件路径',
  `tls_ca_file_path` varchar(255) DEFAULT NULL COMMENT 'TLS CA证书文件路径',
  `tls_ca_content` text COMMENT 'TLS CA证书内容',
  `bearer_token` text COMMENT '鉴权Token内容',
  `bearer_token_file` varchar(255) DEFAULT NULL COMMENT '鉴权Token文件路径',
  `kubernetes_sd_role` varchar(50) DEFAULT NULL COMMENT 'Kubernetes服务发现角色',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_scrape_jobs_name` (`name`),
  KEY `idx_monitor_scrape_jobs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_scrape_jobs`
--

LOCK TABLES `monitor_scrape_jobs` WRITE;
/*!40000 ALTER TABLE `monitor_scrape_jobs` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_scrape_jobs` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_scrape_pools`
--

DROP TABLE IF EXISTS `monitor_scrape_pools`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_scrape_pools` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '采集池名称，支持使用通配符*进行模糊搜索',
  `prometheus_instances` text COMMENT '选择多个Prometheus实例',
  `alert_manager_instances` text COMMENT '选择多个AlertManager实例',
  `user_id` bigint DEFAULT NULL COMMENT '创建该采集池的用户ID',
  `scrape_interval` bigint DEFAULT '30' COMMENT '采集间隔（秒）',
  `scrape_timeout` bigint DEFAULT '10' COMMENT '采集超时时间（秒）',
  `external_labels` text COMMENT 'remote_write时添加的标签组，格式为 key=v，例如 scrape_ip=1.1.1.1',
  `support_alert` bigint DEFAULT NULL COMMENT '是否支持告警：1支持，2不支持',
  `support_record` bigint DEFAULT NULL COMMENT '是否支持预聚合：1支持，2不支持',
  `remote_read_url` varchar(255) DEFAULT NULL COMMENT '远程读取的地址',
  `alert_manager_url` varchar(255) DEFAULT NULL COMMENT 'AlertManager的地址',
  `rule_file_path` varchar(255) DEFAULT NULL COMMENT '规则文件路径',
  `record_file_path` varchar(255) DEFAULT NULL COMMENT '记录文件路径',
  `remote_write_url` varchar(255) DEFAULT NULL COMMENT '远程写入的地址',
  `remote_timeout_seconds` bigint DEFAULT '5' COMMENT '远程写入的超时时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_scrape_pools_name` (`name`),
  KEY `idx_monitor_scrape_pools_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_scrape_pools`
--

LOCK TABLES `monitor_scrape_pools` WRITE;
/*!40000 ALTER TABLE `monitor_scrape_pools` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_scrape_pools` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `monitor_send_groups`
--

DROP TABLE IF EXISTS `monitor_send_groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `monitor_send_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `name` varchar(100) DEFAULT NULL COMMENT '发送组英文名称，供AlertManager配置文件使用，支持通配符*进行模糊搜索',
  `name_zh` varchar(100) DEFAULT NULL COMMENT '发送组中文名称，供告警规则选择发送组时使用，支持通配符*进行模糊搜索',
  `enable` bigint DEFAULT NULL COMMENT '是否启用发送组：1启用，2禁用',
  `user_id` bigint DEFAULT NULL COMMENT '创建该发送组的用户ID',
  `pool_id` bigint DEFAULT NULL COMMENT '关联的AlertManager实例ID',
  `on_duty_group_id` bigint DEFAULT NULL COMMENT '值班组ID',
  `fei_shu_qun_robot_token` varchar(255) DEFAULT NULL COMMENT '飞书机器人Token，对应IM群',
  `repeat_interval` varchar(50) DEFAULT NULL COMMENT '默认重复发送时间',
  `send_resolved` bigint DEFAULT NULL COMMENT '是否发送恢复通知：1发送，2不发送',
  `notify_methods` text COMMENT '通知方法，如：email, im, phone, sms',
  `need_upgrade` bigint DEFAULT NULL COMMENT '是否需要告警升级：1需要，2不需要',
  `upgrade_minutes` bigint DEFAULT NULL COMMENT '告警多久未恢复则升级（分钟）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_monitor_send_groups_name` (`name`),
  UNIQUE KEY `idx_monitor_send_groups_name_zh` (`name_zh`),
  KEY `idx_monitor_send_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `monitor_send_groups`
--

LOCK TABLES `monitor_send_groups` WRITE;
/*!40000 ALTER TABLE `monitor_send_groups` DISABLE KEYS */;
/*!40000 ALTER TABLE `monitor_send_groups` ENABLE KEYS */;
UNLOCK TABLES;

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
  `network_interfaces` varchar(500) DEFAULT NULL COMMENT '弹性网卡 ID 集合',
  `disk_ids` varchar(500) DEFAULT NULL COMMENT '云盘 ID 集合',
  `start_time` varchar(30) DEFAULT NULL COMMENT '最近启动时间, ISO 8601 标准, UTC+0 时间',
  `auto_release_time` varchar(30) DEFAULT NULL COMMENT '自动释放时间, ISO 8601 标准, UTC+0 时间',
  `last_invoked_time` varchar(30) DEFAULT NULL COMMENT '最近调用时间, ISO 8601 标准, UTC+0 时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_resource_ecs_instance_name` (`instance_name`),
  UNIQUE KEY `idx_resource_ecs_hash` (`hash`),
  UNIQUE KEY `idx_resource_ecs_ip_addr` (`ip_addr`),
  KEY `idx_resource_ecs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `resource_ecs`
--

LOCK TABLES `resource_ecs` WRITE;
/*!40000 ALTER TABLE `resource_ecs` DISABLE KEYS */;
/*!40000 ALTER TABLE `resource_ecs` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `resource_elbs`
--

LOCK TABLES `resource_elbs` WRITE;
/*!40000 ALTER TABLE `resource_elbs` DISABLE KEYS */;
/*!40000 ALTER TABLE `resource_elbs` ENABLE KEYS */;
UNLOCK TABLES;

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
  UNIQUE KEY `idx_resource_rds_instance_name` (`instance_name`),
  UNIQUE KEY `idx_resource_rds_hash` (`hash`),
  UNIQUE KEY `idx_resource_rds_ip_addr` (`ip_addr`),
  KEY `idx_resource_rds_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `resource_rds`
--

LOCK TABLES `resource_rds` WRITE;
/*!40000 ALTER TABLE `resource_rds` DISABLE KEYS */;
/*!40000 ALTER TABLE `resource_rds` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roles`
--

DROP TABLE IF EXISTS `roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `roles` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `name` varchar(50) NOT NULL COMMENT '角色名称',
  `description` varchar(255) DEFAULT NULL COMMENT '角色描述',
  `role_type` tinyint(1) NOT NULL COMMENT '角色类型(1:系统角色,2:自定义角色)',
  `is_default` tinyint(1) DEFAULT '0' COMMENT '是否为默认角色(0:否,1:是)',
  `create_time` bigint DEFAULT NULL COMMENT '创建时间',
  `update_time` bigint DEFAULT NULL COMMENT '更新时间',
  `is_deleted` tinyint(1) DEFAULT '0' COMMENT '是否删除(0:否,1:是)',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uni_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roles`
--

LOCK TABLES `roles` WRITE;
/*!40000 ALTER TABLE `roles` DISABLE KEYS */;
/*!40000 ALTER TABLE `roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `second_upgrade_users`
--

DROP TABLE IF EXISTS `second_upgrade_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `second_upgrade_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_second_upgrade_users_user` (`user_id`),
  CONSTRAINT `fk_second_upgrade_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_second_upgrade_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `second_upgrade_users`
--

LOCK TABLES `second_upgrade_users` WRITE;
/*!40000 ALTER TABLE `second_upgrade_users` DISABLE KEYS */;
/*!40000 ALTER TABLE `second_upgrade_users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `static_receive_users`
--

DROP TABLE IF EXISTS `static_receive_users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `static_receive_users` (
  `monitor_send_group_id` bigint NOT NULL COMMENT '主键ID',
  `user_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`monitor_send_group_id`,`user_id`),
  KEY `fk_static_receive_users_user` (`user_id`),
  CONSTRAINT `fk_static_receive_users_monitor_send_group` FOREIGN KEY (`monitor_send_group_id`) REFERENCES `monitor_send_groups` (`id`),
  CONSTRAINT `fk_static_receive_users_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `static_receive_users`
--

LOCK TABLES `static_receive_users` WRITE;
/*!40000 ALTER TABLE `static_receive_users` DISABLE KEYS */;
/*!40000 ALTER TABLE `static_receive_users` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `terraform_configs`
--

LOCK TABLES `terraform_configs` WRITE;
/*!40000 ALTER TABLE `terraform_configs` DISABLE KEYS */;
/*!40000 ALTER TABLE `terraform_configs` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `tree_node_ops_admins`
--

LOCK TABLES `tree_node_ops_admins` WRITE;
/*!40000 ALTER TABLE `tree_node_ops_admins` DISABLE KEYS */;
/*!40000 ALTER TABLE `tree_node_ops_admins` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `tree_node_rd_admins`
--

LOCK TABLES `tree_node_rd_admins` WRITE;
/*!40000 ALTER TABLE `tree_node_rd_admins` DISABLE KEYS */;
/*!40000 ALTER TABLE `tree_node_rd_admins` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `tree_node_rd_members`
--

LOCK TABLES `tree_node_rd_members` WRITE;
/*!40000 ALTER TABLE `tree_node_rd_members` DISABLE KEYS */;
/*!40000 ALTER TABLE `tree_node_rd_members` ENABLE KEYS */;
UNLOCK TABLES;

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
-- Dumping data for table `tree_nodes`
--

LOCK TABLES `tree_nodes` WRITE;
/*!40000 ALTER TABLE `tree_nodes` DISABLE KEYS */;
/*!40000 ALTER TABLE `tree_nodes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_apis`
--

DROP TABLE IF EXISTS `user_apis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_apis` (
  `user_id` bigint NOT NULL COMMENT '主键ID',
  `api_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`user_id`,`api_id`),
  KEY `fk_user_apis_api` (`api_id`),
  CONSTRAINT `fk_user_apis_api` FOREIGN KEY (`api_id`) REFERENCES `apis` (`id`),
  CONSTRAINT `fk_user_apis_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_apis`
--

LOCK TABLES `user_apis` WRITE;
/*!40000 ALTER TABLE `user_apis` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_apis` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_menus`
--

DROP TABLE IF EXISTS `user_menus`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_menus` (
  `user_id` bigint NOT NULL COMMENT '主键ID',
  `menu_id` bigint NOT NULL COMMENT '菜单ID',
  PRIMARY KEY (`user_id`,`menu_id`),
  KEY `fk_user_menus_menu` (`menu_id`),
  CONSTRAINT `fk_user_menus_menu` FOREIGN KEY (`menu_id`) REFERENCES `menus` (`id`),
  CONSTRAINT `fk_user_menus_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_menus`
--

LOCK TABLES `user_menus` WRITE;
/*!40000 ALTER TABLE `user_menus` DISABLE KEYS */;
INSERT INTO `user_menus` VALUES (1,1),(1,2),(1,3),(1,4),(1,5),(1,6),(1,7),(1,8),(1,9),(1,10),(1,11),(1,12),(1,13),(1,14),(1,15),(1,16),(1,17),(1,18),(1,19),(1,20),(1,21),(1,22),(1,23),(1,24),(1,25),(1,26),(1,27),(1,28),(1,29),(1,30),(1,31),(1,32);
/*!40000 ALTER TABLE `user_menus` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `user_roles`
--

DROP TABLE IF EXISTS `user_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_roles` (
  `user_id` bigint NOT NULL COMMENT '主键ID',
  `role_id` bigint NOT NULL COMMENT '主键ID',
  PRIMARY KEY (`user_id`,`role_id`),
  KEY `fk_user_roles_role` (`role_id`),
  CONSTRAINT `fk_user_roles_role` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`),
  CONSTRAINT `fk_user_roles_user` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `user_roles`
--

LOCK TABLES `user_roles` WRITE;
/*!40000 ALTER TABLE `user_roles` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `created_at` datetime(3) DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime(3) DEFAULT NULL COMMENT '更新时间',
  `deleted_at` bigint unsigned DEFAULT NULL COMMENT '删除时间',
  `username` varchar(100) NOT NULL COMMENT '用户登录名',
  `password` varchar(255) NOT NULL COMMENT '用户登录密码',
  `real_name` varchar(100) DEFAULT NULL COMMENT '用户真实姓名',
  `desc` text COMMENT '用户描述',
  `mobile` varchar(20) DEFAULT NULL COMMENT '手机号',
  `fei_shu_user_id` varchar(50) DEFAULT NULL COMMENT '飞书用户ID',
  `account_type` bigint DEFAULT '1' COMMENT '账号类型 1普通用户 2服务账号',
  `home_path` varchar(255) DEFAULT NULL COMMENT '登录后的默认首页',
  `enable` bigint DEFAULT '1' COMMENT '用户状态 1正常 2冻结',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  UNIQUE KEY `idx_users_mobile` (`mobile`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES (1,'2024-12-19 16:18:47.641','2024-12-19 16:18:47.641',0,'admin','$2a$10$Y9JKNYnKJIKyZwPu0MhZKuCu1afnvQFC8.l1kWFiRZEp4TXg1zYa2','管理员账号','','','',2,'',1);
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2024-12-19 16:25:02
