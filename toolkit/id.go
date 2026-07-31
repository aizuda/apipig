package toolkit

import (
	"apipig/toolkit/snowflake"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"math/rand"
	"sync"
	"time"
)

func Uuid() string {
	u := uuid.NewV4()
	return u.String()
}

var sfn *snowflake.Node
var once sync.Once

func Id(node int64) snowflake.ID {
	once.Do(func() {
		var err error
		sfn, err = snowflake.NewNode(node)
		if err != nil {
			panic(err)
		}
	})
	return sfn.Generate()
}

func GenRandomStr(length int) string {
	const charset = "abcdefghijkmnprstuvwxyzABCDEFGHIJKLMNPQRSTUVWXYZ2356789"
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// 生成编号，规则 = 前缀 + 时间戳（精确到毫秒）
func GenNum(prefix string) string {
	ts := time.Now().Format("20060102150405.000")
	return fmt.Sprintf("%s%s%s", prefix, ts[:14], ts[15:])
}
