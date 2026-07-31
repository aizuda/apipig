package mail

import (
	"fmt"
	"testing"
)

func TestSmtp(t *testing.T) {
	username := "发信地址"
	password := "SMTP密码"
	host := "mail.qq.com"
	port := "587"
	to := []string{"收件人地址", "收件人地址1"}
	cc := []string{"抄送地址", "抄送地址1"}
	bcc := []string{"密送地址", "密送地址1"}

	subject := " apipig  Smtp Send Test"
	replyToAddress := "回信地址"

	body := `
    <html>
    <body>
    <h3>
        " apipig  SMTP Send Test Successful!"
    </h3>
    </body>
    </html>
        `
	fmt.Println("send email")
	s := NewSmtp("", username, password, host, port)
	err := s.SendMail(true, subject, body, replyToAddress, to, cc, bcc)
	if err != nil {
		fmt.Println("Send mail error!", err)
	} else {
		fmt.Println("Send mail success!")
	}
}
