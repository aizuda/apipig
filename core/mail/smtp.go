package mail

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
)

type Smtp struct {
	address  string
	username string
	auth     smtp.Auth
}

type SmtpConfig struct {
	Identity string `json:"identity"`
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     string `json:"port"`
}

type SmtpSender struct {
	Html           bool     `json:"html"`
	Subject        string   `json:"subject"`
	Content        string   `json:"content"`
	ReplyToAddress string   `json:"replyToAddress"`
	To             []string `json:"to"`
	Cc             []string `json:"cc"`
	Bcc            []string `json:"bcc"`
}

func NewSmtp(identity, username, password, host, port string) Smtp {
	return Smtp{
		address:  fmt.Sprintf("%s:%s", host, port),
		username: username,

		// 认证信息
		auth: smtp.PlainAuth(identity, username, password, host),
	}
}

func SendMail(config string, sender SmtpSender) error {
	var sc SmtpConfig
	err := json.Unmarshal([]byte(config), &sc)
	if err != nil {
		return err
	}
	return NewSmtp(sc.Identity, sc.Username, sc.Password, sc.Host, sc.Port).SendMail(sender.Html,
		sender.Subject, sender.Content, sender.ReplyToAddress, sender.To, sender.Cc, sender.Bcc)
}

// SendMail 发送邮件
// html 发送内容是否为 Html
// subject 主题
// content 内容
// replyToAddress 回信地址
// to 收件人
// cc 抄送人
// bcc 密送人
func (s Smtp) SendMail(html bool, subject, content, replyToAddress string, to, cc, bcc []string) error {
	var sendTo = to
	var bt bytes.Buffer

	// 收件人
	bt.WriteString("To:")
	bt.WriteString(strings.Join(to, ","))
	bt.WriteString("\n")

	// 主题
	bt.WriteString("Subject: ")
	bt.WriteString(subject)
	bt.WriteString("\n")

	// 邮件头部
	bt.WriteString("From: ")
	bt.WriteString(s.username)
	bt.WriteString("\n")

	// 回信地址
	if len(replyToAddress) > 0 {
		bt.WriteString("Reply-To: ")
		bt.WriteString(replyToAddress)
		bt.WriteString("\n")
	}

	// 抄送人
	if len(cc) > 0 {
		sendTo = append(sendTo, cc...)
		bt.WriteString("Cc: ")
		bt.WriteString(strings.Join(cc, ","))
		bt.WriteString("\n")
	}

	// 密送人
	if len(bcc) > 0 {
		sendTo = append(sendTo, bcc...)
		bt.WriteString("Bcc: ")
		bt.WriteString(strings.Join(bcc, ","))
		bt.WriteString("\n")
	}

	// 内容类型
	bt.WriteString("Content-Type: text/")
	if html {
		bt.WriteString("html")
	} else {
		bt.WriteString("plain")
	}
	bt.WriteString("; charset=\"UTF-8\"")
	bt.WriteString("\n\n")
	bt.WriteString(content)
	return smtp.SendMail(s.address, s.auth, s.username, sendTo, bt.Bytes())
}
