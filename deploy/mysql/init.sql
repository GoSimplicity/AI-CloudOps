-- 创建数据库
CREATE DATABASE IF NOT EXISTS cloudops DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;

-- 使用数据库
USE cloudops;

-- 设置时区
SET time_zone = '+08:00';

-- 创建用户（如果需要）
-- CREATE USER IF NOT EXISTS 'cloudops'@'%' IDENTIFIED BY 'cloudops123';
-- GRANT ALL PRIVILEGES ON cloudops.* TO 'cloudops'@'%';
-- FLUSH PRIVILEGES;

-- 基础表结构会由应用程序自动创建
SELECT 'MySQL初始化完成' as message;