package config

// 仅向外部包集成测试开放真实解析入口，避免业务包依赖配置类型时形成测试导入环。
var LoadWebhookFromFileForTest = loadWebhookFromFile
var ClearWebhookEnvForTest = clearAllWebhookEnv
