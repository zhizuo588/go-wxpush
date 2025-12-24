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
	"os"
	"strings"
	"time"
)

// 请求参数结构体
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

// 全局变量用于存储命令行参数
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

// 微信AccessToken响应
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// 微信模板消息请求
type TemplateMessageRequest struct {
	ToUser     string                 `json:"touser"`
	TemplateID string                 `json:"template_id"`
	URL        string                 `json:"url"`
	Data       map[string]interface{} `json:"data"`
}

// 微信API响应
type WechatAPIResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// 嵌入静态HTML文件
//go:embed msg_detail.html
var htmlContent embed.FS

func main() {
	// 定义命令行参数，同时支持从环境变量读取默认值（适配 Render）
	flag.StringVar(&cliTitle, "title", os.Getenv("TITLE"), "消息标题")
	flag.StringVar(&cliContent, "content", os.Getenv("CONTENT"), "消息内容")
	flag.StringVar(&cliAppID, "appid", os.Getenv("APPID"), "AppID")
	flag.StringVar(&cliSecret, "secret", os.Getenv("SECRET"), "AppSecret")
	flag.StringVar(&cliUserID, "userid", os.Getenv("USERID"), "openid")
	flag.StringVar(&cliTemplateID, "template_id", os.Getenv("TEMPLATE_ID"), "模板ID")
	flag.StringVar(&cliBaseURL, "base_url", os.Getenv("BASE_URL"), "跳转url")
	flag.StringVar(&cliTimezone, "tz", "Asia/Shanghai", "时区")
	flag.StringVar(&startPort, "port", os.Getenv("PORT"), "端口")

	// 解析命令行参数
	flag.Parse()

	// 设置路由
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `go-wxpush is running...✅`)
	})
	http.HandleFunc("/wxsend", handleWxSend)
	http.HandleFunc("/detail", handleDetail)

	// 确定端口
	port := "5566"
	if startPort != "" {
		port = startPort
	}
	fmt.Println("Server is running on port: " + port)

	// 启动服务器（Render 环境必须监听 :port）
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

// 处理详情页面请求
func handleDetail(w http.ResponseWriter, r *http.Request) {
	htmlData, err := htmlContent.ReadFile("msg_detail.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "Failed to read embedded HTML file: %v"}`, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlData)
}

func handleWxSend(w http.ResponseWriter, r *http.Request) {
	var params RequestParams
	if r.Method == "POST" {
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&params)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "Invalid JSON format: %v"}`, err)
			return
		}
	} else if r.Method == "GET" {
		params.Title = r.URL.Query().Get("title")
		params.Content = r.URL.Query().Get("content")
		params.AppID = r.URL.Query().Get("appid")
		params.Secret = r.URL.Query().Get("secret")
		params.UserID = r.URL.Query().Get("userid")
		params.TemplateID = r.URL.Query().Get("template_id")
		params.BaseURL = r.URL.Query().Get("base_url")
		params.Timezone = r.URL.Query().Get("tz")
	}

	// 合并环境变量/命令行参数
	if params.Title == "" { params.Title = cliTitle }
	if params.Content == "" { params.Content = cliContent }
	if params.AppID == "" { params.AppID = cliAppID }
	if params.Secret == "" { params.Secret = cliSecret }
	if params.UserID == "" { params.UserID = cliUserID }
	if params.TemplateID == "" { params.TemplateID = cliTemplateID }
	if params.BaseURL == "" { params.BaseURL = cliBaseURL }
	if params.Timezone == "" { params.Timezone = cliTimezone }

	if params.AppID == "" || params.Secret == "" || params.UserID == "" || params.TemplateID == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "Missing required parameters"}`)
		return
	}

	token, err := getAccessToken(params.AppID, params.Secret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "Failed to get access token: %v"}`, err)
		return
	}

	resp, err := sendTemplateMessage(token, params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "Failed to send template message: %v"}`, err)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func getAccessToken(appid, secret string) (string, error) {
	apiUrl := "https://api.weixin.qq.com/cgi-bin/stable_token"
	requestData := map[string]interface{}{
		"grant_type": "client_credential",
		"appid":      appid,
		"secret":     secret,
	}
	jsonData, _ := json.Marshal(requestData)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Post(apiUrl, "application/json", strings.NewReader(string(jsonData)))
	if err != nil { return "", err }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tokenResp AccessTokenResponse
	json.Unmarshal(body, &tokenResp)
	return tokenResp.AccessToken, nil
}

func sendTemplateMessage(accessToken string, params RequestParams) (WechatAPIResponse, error) {
	apiUrl := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/message/template/send?access_token=%s", accessToken)
	location, _ := time.LoadLocation(params.Timezone)
	if location == nil { location = time.Local }
	timeStr := time.Now().In(location).Format("2006-01-02 15:04:05")

	requestData := TemplateMessageRequest{
		ToUser:     params.UserID,
		TemplateID: params.TemplateID,
		URL:        params.BaseURL + `/detail?title=` + url.QueryEscape(params.Title) + `&message=` + url.QueryEscape(params.Content) + `&date=` + url.QueryEscape(timeStr),
		Data: map[string]interface{}{
			"title":   map[string]string{"value": params.Title},
			"content": map[string]string{"value": params.Content},
		},
	}
	jsonData, _ := json.Marshal(requestData)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Post(apiUrl, "application/json", strings.NewReader(string(jsonData)))
	if err != nil { return WechatAPIResponse{}, err }
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var apiResp WechatAPIResponse
	json.Unmarshal(body, &apiResp)
	return apiResp, nil
}
