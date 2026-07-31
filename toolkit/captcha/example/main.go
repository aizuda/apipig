package main

import (
	"apipig/toolkit/captcha"
	"fmt"
	"image/color"
	"net/http"
)

func main() {
	http.HandleFunc("/r", func(w http.ResponseWriter, r *http.Request) {
		data, _ := captcha.New(&captcha.Options{
			Curve:           2,   // 两条弧线
			Length:          4,   // 长度为4的验证码
			Width:           160, // 图片宽
			Height:          50,  // 图片高
			BackgroundColor: color.White,
		})
		fmt.Println(data.Text)
		data.WriteImage(w)
	})
	http.ListenAndServe(":8085", nil)
}
