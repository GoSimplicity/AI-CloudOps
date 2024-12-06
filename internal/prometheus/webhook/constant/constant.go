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

package constant

// AlertSeverity 表示告警的严重性等级
type AlertSeverity string

// AlertStatus 表示告警的状态
type AlertStatus string

var (
	CardContent = `
{
  "header": {
    "template": "%s",
    "title": {
      "content": "%s",
      "tag": "plain_text"
    }
  },
  "elements": [
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s"
          }
        }
      ]
    },
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s"
          }
        }
      ]
    },
    {
      "tag": "column_set",
      "flex_mode": "none",
      "background_style": "default",
      "columns": [
        {
          "tag": "column",
          "width": "weighted",
          "weight": 1,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "%s",
                "tag": "lark_md"
              }
            }
          ]
        },
        {
          "tag": "column",
          "width": "weighted",
          "weight": 1,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "%s",
                "tag": "lark_md"
              }
            }
          ]
        }
      ]
    },
    {
      "tag": "column_set",
      "flex_mode": "none",
      "background_style": "default",
      "columns": [
        {
          "tag": "column",
          "width": "weighted",
          "weight": 1,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "div",
              "text": {
                "content": "%s",
                "tag": "lark_md"
              }
            }
          ]
        },
        {
          "tag": "column",
          "width": "weighted",
          "weight": 1,
          "vertical_align": "top",
          "elements": [
            {
              "tag": "markdown",
              "content": "%s"
            }
          ]
        }
      ]
    },
    {
      "tag": "div",
      "fields": [
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s\n"
          }
        },
        {
          "is_short": true,
          "text": {
            "tag": "lark_md",
            "content": "%s"
          }
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "markdown",
      "content": "%s"
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "🔴 告警屏蔽按钮 [单一告警屏蔽👇]"
      }
    },
    {
      "tag": "action",
      "actions": [
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "认领告警"
          },
          "type": "primary",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定认领吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        },
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "屏蔽1小时"
          },
          "type": "default",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定屏蔽吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        },
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "屏蔽24小时"
          },
          "type": "danger",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定屏蔽吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "action",
      "actions": [
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "取消屏蔽"
          },
          "type": "primary",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定取消吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        },
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "屏蔽6小时"
          },
          "type": "default",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定屏蔽吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        },
        {
          "tag": "button",
          "text": {
            "tag": "plain_text",
            "content": "屏蔽7天"
          },
          "type": "danger",
          "url": "%s",
          "confirm": {
            "title": {
              "tag": "plain_text",
              "content": "确定屏蔽吗"
            },
            "text": {
              "tag": "plain_text",
              "content": ""
            }
          }
        }
      ]
    },
    {
      "tag": "hr"
    },
    {
      "tag": "div",
      "text": {
        "tag": "lark_md",
        "content": "🙋‍♂️ [我要反馈错误](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message-development-tutorial/introduction?from=mcb) | 📝 [录入报警处理过程](https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/im-v1/message-development-tutorial/introduction?from=mcb)"
      }
    }
  ]
}
`

	CartDataGroup = `
{
    "msg_type": "interactive",
    "card": %s
}
`
)

const (
	AlertSeverityCritical AlertSeverity = "critical" // 严重
	AlertSeverityWarning  AlertSeverity = "warning"  // 警告
	AlertSeverityInfo     AlertSeverity = "info"     // 信息

	AlertStatusFiring   AlertStatus = "firing"   // 触发中
	AlertStatusResolved AlertStatus = "resolved" // 已恢复
)

// SeverityTitleColorMap 将告警严重性映射到标题颜色
var SeverityTitleColorMap = map[AlertSeverity]string{
	AlertSeverityCritical: "red",    // 严重 - 红色
	AlertSeverityWarning:  "yellow", // 警告 - 黄色
	AlertSeverityInfo:     "blue",   // 信息 - 蓝色
}

// StatusColorMap 将告警状态映射到颜色
var StatusColorMap = map[AlertStatus]string{
	AlertStatusFiring:   "red",   // 触发中 - 红色
	AlertStatusResolved: "green", // 已恢复 - 绿色
}

// StatusChineseMap 将告警状态映射到中文描述
var StatusChineseMap = map[AlertStatus]string{
	AlertStatusFiring:   "触发中", // 触发中
	AlertStatusResolved: "已恢复", // 已恢复
}

// URL 模板常量
const (
	SendGroupURLTemplate = "http://%s/%s?id=%v"                  // 发送组 URL 模板
	RenderingURLTemplate = "http://%s/%s?fingerprint=%v"         // 渲染 URL 模板
	SilenceURLTemplate   = "http://%s/%s?fingerprint=%v&hour=%v" // 静音 URL 模板
	UnsilenceURLTemplate = "http://%s/%s?fingerprint=%v"         // 取消静音 URL 模板

	// DefaultUpgradeMinutes 默认告警升级时间（分钟）
	DefaultUpgradeMinutes = 30 // 默认告警升级时间为30分钟
)
