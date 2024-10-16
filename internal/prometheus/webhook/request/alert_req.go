package request

// AlertmanagerSilenceResponse 表示告警管理器静默响应结构体
type AlertmanagerSilenceResponse struct {
	Status string `json:"status"`
	Data   struct {
		SilenceId string `json:"silenceId"`
	} `json:"data"`
}
