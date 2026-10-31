package config

import "github.com/spf13/viper"

// CronLocalExecPolicy 在保存、调度和执行时读取同一动态本地执行门禁。
// 未注入策略时拒绝本地命令和脚本，不依赖业务包中的隐式全局配置。
type CronLocalExecPolicy func() bool

func (policy CronLocalExecPolicy) IsAllowed() bool {
	return policy != nil && policy()
}

// NewCronLocalExecPolicy 绑定已加载的配置实例；每次检查仍读取当前值和环境变量。
// 不把可撤销的执行权限冻结为应用启动时的 bool 快照。
func NewCronLocalExecPolicy() CronLocalExecPolicy {
	loaded := viper.GetViper()
	return func() bool { return loaded.GetBool(CronAllowLocalExecKey) }
}
