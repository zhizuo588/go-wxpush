package main

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os" // 新增：用于读取环境变量
	"strings"
	"time"
)

// ... (RequestParams 等结构体保持不变)
type RequestParams struct {
	Title      string `json:"title" form:"title"`
	Content    string `json:"content" form:"content"`
	AppID      string `json:"appid" form:"appid"`
	Secret     string `json:"secret" form:"secret"`
	UserID     string `json:"userid" form:"userid"`
	TemplateID string `json:"template_id" form:"template_id"`
	BaseURL    string `json:"base_url" form:"base_url"`
	Timezone   string `json:"tz" form:"tz"`
}

var (
	cliTitle      string
	cliContent    string
	cliAppID      string
	cliSecret     string
	cliUserID     string
	cliTemplateID string
	cliBaseURL    string
	startPort     string
	cliTimezone   string
)

// ... (AccessTokenResponse 等结构体保持不变)
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type TemplateMessageRequest struct {
	ToUser     string                 `json:"touser"`
	TemplateID string                 `json:"template_id"`
	URL        string                 `json:"url"`
	Data       map[string]interface{} `json:"data"`
}

type WechatAPIResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

func main() {
	// 修改点：从环境变量获取默认值，如果环境变量没有，再看命令行参数
	// 这样在 Render 上部署会非常方便
	flag.StringVar(&cliTitle, "title", os.Getenv("TITLE"), "消息标题")
	flag.StringVar(&cliContent, "content", os.Getenv("CONTENT"), "消息内容")
	flag.StringVar(&cliAppID, "appid", os.Getenv("APPID"), "AppID")
	flag.StringVar(&cliSecret, "secret", os.Getenv("SECRET"), "AppSecret")
	flag.StringVar(&cliUserID, "userid", os.Getenv("USERID"), "openid")
	flag.StringVar(&cliTemplateID, "template_id", os.Getenv("TEMPLATE_ID"), "模板ID")
	flag.StringVar(&cliBaseURL, "base_url", os.Getenv("BASE_URL"), "跳转url")
	flag.StringVar(&cliTimezone, "tz", "Asia/Shanghai", "时区")
	
	// 重要：Render 会自动分配一个 $PORT 环境变量
	envPort := os.Getenv("PORT")
	flag.StringVar(&startPort, "port", envPort, "端口")

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `go-wxpush is running...✅`)
	})
	http.HandleFunc("/wxsend", handleWxSend)
	http.HandleFunc("/detail", handleDetail)

	// 端口逻辑优化
	port := "5566"
	if startPort != "" {
		port = startPort
	}
	
	fmt.Println("Server is running on port: " + port)
	// 在 Render 上必须监听 0.0.0.0 而不是 127.0.0.1
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

// ... (handleDetail, handleWxSend, getAccessToken, sendTemplateMessage 函数保持你原来的逻辑即可)
// 注意：确保 handleDetail 里的 htmlContent 定义在 handleDetail 之前
