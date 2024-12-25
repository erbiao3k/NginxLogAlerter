package public

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// GeneratorMd5 生成string的md5值
func GeneratorMd5(code string) string {
	MD5 := md5.New()
	_, _ = io.WriteString(MD5, code)
	return hex.EncodeToString(MD5.Sum(nil))
}

var WecomRobotAddr = []string{
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=000",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=111",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=222",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=333",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=444",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=555",
	"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=666",
}

func Sender(data, people string) error {
	// 设置随机种子
	rand.Seed(time.Now().UnixNano())
	// 产生一个 [0, len(wecomRobotAddr)-1) 的随机整数
	randNum := rand.Intn(len(WecomRobotAddr) - 1)

	postData := fmt.Sprintf(`{"msgtype": "text", "text": {"content": "%s","mentioned_mobile_list":["%s"]}}`, data, people)

	if _, err := http.Post(WecomRobotAddr[randNum], "application/json", strings.NewReader(postData)); err != nil {
		return err
	}
	return nil
}
