package toolkit

import (
	"fmt"
	"testing"
)

func TestUuid(t *testing.T) {
	if uuid := Uuid(); uuid == "" {
		t.Errorf("uuid generate result is null")
	} else {
		t.Logf("uuid generate result: %s", uuid)
	}
}

func TestId(t *testing.T) {
	if id := Id(0); id == 0 {
		t.Errorf("snowflake generate result is null")
	} else {
		t.Logf("snowflake id generate result: %d", id)
	}
}

func TestGenRandomStr(t *testing.T) {
	fmt.Println("随机18位字符串：" + GenRandomStr(18))
}

func TestGenNum(t *testing.T) {
	fmt.Println("产品编码：" + GenNum("P"))
	fmt.Println("设备编码：" + GenNum("D"))
}
