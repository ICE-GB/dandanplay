package main

import (
    "crypto/sha256"
    "encoding/base64"
    "flag"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "os"
    "strings"
    "time"
)

// 常量定义
const (
    AppId     = "your_app_id"
    AppSecret = "your_app_secret"
)

// 提取URL中的Path部分
func extractPath(rawURL string) (string, error) {
    parsedURL, err := url.Parse(rawURL)
    if err != nil {
        return "", err
    }
    return parsedURL.Path, nil
}

// 生成 X-Signature
func generateXSignature(rawURL string) (string, int64, error) {
    // 获取当前的 Unix 时间戳
    timestamp := time.Now().Unix()

    // 提取 URL 的 Path 部分
    urlPath, err := extractPath(rawURL)
    if err != nil {
        return "", 0, err
    }

    // 拼接字符串
    dataToHash := fmt.Sprintf("%s%d%s%s", AppId, timestamp, urlPath, AppSecret)

    // 计算 SHA256 哈希值
    hash := sha256.Sum256([]byte(dataToHash))

    // 将哈希值转换为 Base64 编码格式
    base64Hash := base64.StdEncoding.EncodeToString(hash[:])

    return base64Hash, timestamp, nil
}

func main() {
    // 定义命令行参数
    method := flag.String("X", "GET", "HTTP request method")
    data := flag.String("d", "", "HTTP POST data")
    output := flag.String("o", "", "Output file")
    headers := flag.NewFlagSet("headers", flag.ContinueOnError)
    headerList := headers.StringArray("H", nil, "HTTP headers")

    // 解析命令行参数
    flag.Parse()
    headers.Parse(flag.Args())

    // 获取URL
    args := headers.Args()
    if len(args) < 1 {
        fmt.Println("Usage: go run main.go [options] <URL>")
        return
    }
    rawURL := args[0]

    // 创建 HTTP 请求
    client := &http.Client{}
    req, err := http.NewRequest(*method, rawURL, strings.NewReader(*data))
    if err != nil {
        fmt.Printf("Failed to create HTTP request: %v\n", err)
        return
    }

    // 处理头信息
    for _, header := range *headerList {
        kv := strings.SplitN(header, ":", 2)
        if len(kv) == 2 {
            req.Header.Add(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
        }
    }

    // 只在URL是以 https://api.dandanplay.net/ 开头时添加自定义 HTTP 头
    if strings.HasPrefix(rawURL, "https://api.dandanplay.net/") {
        // 生成 X-Signature
        xSignature, timestamp, err := generateXSignature(rawURL)
        if err != nil {
            fmt.Printf("Failed to generate X-Signature: %v\n", err)
            return
        }

        req.Header.Add("X-Signature", xSignature)
        req.Header.Add("X-AppId", AppId)
        req.Header.Add("X-Timestamp", fmt.Sprintf("%d", timestamp))
    }

    // 发送 HTTP 请求
    resp, err := client.Do(req)
    if err != nil {
        fmt.Printf("Failed to send HTTP request: %v\n", err)
        return
    }
    defer resp.Body.Close()

    // 输出响应
    var outputWriter io.Writer = os.Stdout
    if *output != "" {
        file, err := os.Create(*output)
        if err != nil {
            fmt.Printf("Failed to create output file: %v\n", err)
            return
        }
        defer file.Close()
        outputWriter = file
    }

    // 打印响应状态
    fmt.Fprintln(outputWriter, "Response status:", resp.Status)
    // 打印响应头
    for key, values := range resp.Header {
        for _, value := range values {
            fmt.Fprintf(outputWriter, "%s: %s\n", key, value)
        }
    }
    // 打印响应体
    io.Copy(outputWriter, resp.Body)
}
