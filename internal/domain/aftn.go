package domain

import (
	"caatsm/pkg/utils"
	"fmt"
	"time"
)

// AFTN 定义AFTN报文的结构
type AFTN struct {
	Header            Header            `json:"header"`              // 报文头部信息
	PriorityAndSender PriorityAndSender `json:"priority_and_sender"` // 优先级和发送地址信息
	TimeAndReceiver   TimeAndReceiver   `json:"time_and_receiver"`   // 时间和接收地址信息
	Text              string            `json:"text"`                // 报文内容
	ReceivedTime      time.Time         `json:"received_time"`       // 收报时间，表示电报接收到的时间
	Category          string            `json:"category"`
	BodyData          interface{}       `json:"body_data"`
}

// Header 定义AFTN报文的报头
type Header struct {
	StartSignal string `json:"start_signal"` // 启动信号，表示报文的开始，通常为固定值
	SendID      string `json:"send_id"`      // 发送编号，用于唯一标识报文
	SendTime    string `json:"send_time"`    // 发送时间，格式为DDHHMM
}

// 示例:
// Header{
//     StartSignal: "ZCZC",
//     SendID:      "TMQ2611",
//     SendTime:    "151524",
// }

// PriorityAndSender 定义优先级和发送地址
type PriorityAndSender struct {
	Priority string `json:"priority"` // 优先级
	Sender   string `json:"sender"`   // 发报地址
}

// 示例:
// PriorityAndSender{
//     Priority: "FF",
//     Sender:   "ZBTJZPZX",
// }

// TimeAndReceiver 定义时间和接收地址
type TimeAndReceiver struct {
	Time     string `json:"time"`     // 时间
	Receiver string `json:"receiver"` // 收报地址
}

// 示例:
// TimeAndReceiver{
//     Time:     "151524",
//     Receiver: "ZGGGZPZX",
// }

// Origin 定义AFTN报文的来源
type Origin struct {
	OriginCode string    `json:"origin_code"` // 发报地址代码
	FiledTime  time.Time `json:"filed_time"`  // 签发时间，表示电报生成的时间
}

func (h *Header) Validate() error {
	// Validate SendTime format (e.g., DDHHMM)
	if len(h.SendTime) != 6 {
		err := "invalid send_time format"

		utils.Logger.Error(err)
		return fmt.Errorf(err)
	}
	return nil
}

func (a *AFTN) Validate() error {
	if err := a.Header.Validate(); err != nil {
		return err
	}
	// Add more validation as needed
	return nil
}
