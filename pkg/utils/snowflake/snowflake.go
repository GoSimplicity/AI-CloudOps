package snowflake

import (
	"time"

	sf "github.com/bwmarrin/snowflake"
)

// InitializeSnowflakeNode 初始化雪花节点
func InitializeSnowflakeNode() *sf.Node {
	st := getTime()
	sf.Epoch = st.UnixNano() / 1000000 // 计算起始时间戳并赋值给sf.Epoch
	node, err := sf.NewNode(1)         // 创建新的节点
	if err != nil {
		return nil
	}
	return node
}

func getTime() time.Time {
	year := 2024
	month := time.May
	day := 13
	hour := 10
	minute := 30
	second := 0
	nanosecond := 0
	// 加载北京时间 (CST) 时区
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	return time.Date(year, month, day, hour, minute, second, nanosecond, location)
}
