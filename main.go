package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	outputDir       = "./output"
	emailSuffix     = "@uuf.me"
	passwordPre     = "testbit#P"
	userPre         = "testbit"
	jsonFile        = "latest.json"
	uniqueNamesFile = "unique_names.txt"
	defaultPort     = "8080"
)

var (
	genMu     sync.Mutex
	genNumber int64
	randLen   = 13
)

func generateUniqueRandomStr(length int) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	path := filepath.Join(filepath.Dir(exePath), uniqueNamesFile)

	existMap := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err == nil {
		tempStr := ""
		for _, c := range data {
			if c == '\n' {
				if tempStr != "" {
					existMap[tempStr] = true
					tempStr = ""
				}
			} else {
				tempStr += string(c)
			}
		}
		if tempStr != "" {
			existMap[tempStr] = true
		}
	}

	letters := "abcdefghijklmnopqrstuvwxyz"
	max := big.NewInt(int64(len(letters)))

	for {
		str := make([]byte, length)
		for i := 0; i < length; i++ {
			num, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			str[i] = letters[num.Int64()]
		}
		result := string(str)

		if !existMap[result] {
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return "", err
			}
			_, _ = f.WriteString(result + "\n")
			f.Close()
			return result, nil
		}
	}
}

func generateContent() (string, map[string]string, error) {
	now := time.Now()

	dateStr := fmt.Sprintf("%02d%02d%02d", now.Year()%100, now.Month(), now.Day())
	timeStr := fmt.Sprintf("%02d%02d%02d", now.Hour(), now.Minute(), now.Second())
	numStr := dateStr + timeStr

	randomStr, err := generateUniqueRandomStr(randLen)
	if err != nil {
		return "", nil, err
	}

	email := fmt.Sprintf("%s%s%s", userPre, numStr, emailSuffix)
	emailUsername := strings.SplitN(email, "@", 2)[0]
	username := fmt.Sprintf("%s%s", userPre, randomStr)
	password := fmt.Sprintf("%s%s", passwordPre, timeStr)
	createdAt := now.Format("2006-01-02 15:04:05")

	content := fmt.Sprintf(`姓：%s
名：%s
邮箱：%s
邮箱用户名：%s
用户名：%s
密码：%s
手机号：%s

`,
		userPre,
		userPre,
		email,
		emailUsername,
		username,
		password,
		numStr,
	)

	displayText := fmt.Sprintf("姓：%s\n名：%s\n邮箱：%s\n邮箱用户名：%s\n用户名：%s\n密码：%s\n手机号：%s",
		userPre, userPre, email, emailUsername, username, password, numStr)

	account := map[string]string{
		"name":            userPre,
		"firstName":       userPre,
		"first_name":      userPre,
		"lastName":        userPre,
		"last_name":       userPre,
		"email":           email,
		"placeholder:用户名": emailUsername,
		"username":        username,
		"password":        password,
		"phone":           numStr,
		"mobile_number":   numStr,
		"created_at":      createdAt,
		"display":         displayText,
	}

	return content, account, nil
}

func generateAndSave(dir string) (map[string]string, string, error) {
	genMu.Lock()
	defer genMu.Unlock()

	now := time.Now()
	fileName := fmt.Sprintf("%04d%02d%02d.txt", now.Year(), now.Month(), now.Day())
	filePath := filepath.Join(dir, fileName)

	content, account, err := generateContent()
	if err != nil {
		return nil, "", err
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, "", err
	}
	_, err = f.WriteString(content)
	f.Close()
	if err != nil {
		return nil, "", err
	}

	jsonPath := filepath.Join(dir, jsonFile)
	jsonData, err := json.MarshalIndent(account, "", "  ")
	if err == nil {
		os.WriteFile(jsonPath, jsonData, 0644)
	}

	genNumber++

	return account, filePath, nil
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printHelp()
			return
		}
		if arg == "--cli-only" {
			runCLI()
			return
		}
	}

	dir := outputDir
	port := defaultPort
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--port" && i+1 < len(args) {
			port = args[i+1]
			i++
		} else if args[i] == "--rand-len" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err == nil && n > 0 {
				randLen = n
			}
			i++
		} else if !strings.HasPrefix(args[i], "-") {
			dir = args[i]
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}

	http.HandleFunc("/api/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		jsonPath := filepath.Join(dir, jsonFile)
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			http.Error(w, `{"error":"no data"}`, http.StatusNotFound)
			return
		}
		w.Write(data)
	})

	http.HandleFunc("/api/generate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		account, filePath, err := generateAndSave(dir)
		if err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		account["file_path"] = filePath
		account["count"] = fmt.Sprintf("%d", genNumber)
		json.NewEncoder(w).Encode(account)
	})

	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		entries, _ := os.ReadDir(dir)
		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
				files = append(files, e.Name())
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"files": files, "total": len(files)})
	})

	http.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"rand_len":   randLen,
				"output_dir": dir,
			})
		case "POST":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			if n, ok := body["rand_len"]; ok {
				if v, ok := n.(float64); ok && v > 0 {
					randLen = int(v)
				}
			}
			if d, ok := body["output_dir"]; ok {
				if s, ok := d.(string); ok && s != "" {
					if err := os.MkdirAll(s, 0755); err == nil {
						dir = s
					}
				}
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"rand_len":   randLen,
				"output_dir": dir,
			})
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(indexHTML))
	})

	var actualPort string
	go func() {
		currentPort := port
		for {
			addr := ":" + currentPort
			fmt.Printf("HTTP服务: http://localhost:%s\n", currentPort)
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				if strings.Contains(err.Error(), "address already in use") || strings.Contains(err.Error(), "Only one usage") {
					nextPort, _ := strconv.Atoi(currentPort)
					currentPort = strconv.Itoa(nextPort + 1)
					fmt.Printf("> 端口 %s 被占用，尝试端口 %s\n", addr[1:], currentPort)
					continue
				}
				fmt.Printf("HTTP服务启动失败: %v\n", err)
				return
			}
			break
		}
	}()
	actualPort = port

	fmt.Printf("> 输出目录: %s\n", dir)
	fmt.Printf("> 随机字母长度: %d\n", randLen)
	fmt.Printf("> 打开浏览器访问 http://localhost:%s 使用UI界面\n", actualPort)
	fmt.Println("> 按 Enter 切换到CLI模式生成，输入 q 退出")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if strings.ToLower(input) == "q" || strings.ToLower(input) == "exit" {
			fmt.Println("已退出")
			return
		}
		if input == "-h" || input == "--help" {
			printHelp()
			continue
		}

		account, filePath, err := generateAndSave(dir)
		if err != nil {
			fmt.Printf("生成失败: %v\n", err)
			continue
		}
		fmt.Printf("[%d] %s 生成成功 -> %s\n", genNumber, account["created_at"], filePath)
		fmt.Println(account["display"])
		fmt.Println(strings.Repeat("-", 40))
	}
}

func runCLI() {
	dir := outputDir
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--rand-len" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err == nil && n > 0 {
				randLen = n
			}
			i++
		} else if !strings.HasPrefix(args[i], "-") {
			dir = args[i]
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("创建目录失败: %v\n", err)
		return
	}
	fmt.Printf("> 输出目录: %s\n", dir)
	fmt.Printf("> 随机字母长度: %d\n", randLen)
	fmt.Println("> 按 Enter 生成一条账号，输入 q 退出")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if strings.ToLower(input) == "q" || strings.ToLower(input) == "exit" {
			fmt.Println("已退出")
			return
		}

		account, filePath, err := generateAndSave(dir)
		if err != nil {
			fmt.Printf("生成失败: %v\n", err)
			continue
		}
		fmt.Printf("[%d] %s 生成成功 -> %s\n", genNumber, account["created_at"], filePath)
		fmt.Println(account["display"])
		fmt.Println(strings.Repeat("-", 40))
	}
}

func printHelp() {
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("  账号生成器 使用帮助")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()
	fmt.Println("> 用法:")
	fmt.Println("  account_gen.exe [输出目录]")
	fmt.Println("  account_gen.exe [输出目录] --port 9090")
	fmt.Println("  account_gen.exe [输出目录] --rand-len 8")
	fmt.Println("  account_gen.exe --cli-only    仅CLI模式")
	fmt.Println("  account_gen.exe -h")
	fmt.Println()
	fmt.Println("> 参数:")
	fmt.Println("  --port PORT    指定HTTP端口 (默认 8080)")
	fmt.Println("  --rand-len N   指定随机字母长度 (默认 13)")
	fmt.Println("  --cli-only     仅命令行模式")
	fmt.Println()
	fmt.Println("> 默认同时启动Web UI: http://localhost:8080")
}

var indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>账号生成器</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background: #08080f;
    color: #e2e8f0;
    min-height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 20px;
  }
  .container {
    width: 100%;
    max-width: 540px;
    background: #12121e;
    border-radius: 28px;
    padding: 36px 28px;
    box-shadow: 0 0 0 1px rgba(99,102,241,0.06), 0 25px 60px rgba(0,0,0,0.7);
    position: relative;
    overflow: hidden;
  }
  .container::before {
    content: '';
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 1px;
    background: linear-gradient(90deg, transparent, rgba(99,102,241,0.25), transparent);
  }
  .header {
    text-align: center;
    margin-bottom: 28px;
    position: relative;
  }
  .header h1 {
    font-size: 24px;
    font-weight: 600;
    letter-spacing: -0.3px;
    color: #f1f5f9;
  }
  .header p {
    color: #64748b;
    font-size: 13px;
    margin-top: 5px;
    font-weight: 400;
  }
  .generate-area {
    text-align: center;
    margin-bottom: 24px;
  }
  .btn-generate {
    width: 100%;
    padding: 15px;
    font-size: 16px;
    font-weight: 500;
    border: none;
    border-radius: 12px;
    background: #6366f1;
    color: #fff;
    cursor: pointer;
    transition: all 0.2s ease;
    letter-spacing: 0.3px;
    position: relative;
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(99,102,241,0.25);
  }
  .btn-generate:hover { background: #5558e6; transform: translateY(-1px); box-shadow: 0 6px 28px rgba(99,102,241,0.35); }
  .btn-generate:active { transform: translateY(0); box-shadow: 0 2px 12px rgba(99,102,241,0.2); }
  .btn-generate:disabled { opacity: 0.35; cursor: not-allowed; box-shadow: none; transform: none; }
  .btn-generate .spinner {
    display: none;
    width: 20px; height: 20px;
    border: 2.5px solid rgba(255,255,255,0.25);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.6s linear infinite;
    margin: 0 auto;
  }
  .btn-generate.loading .spinner { display: block; }
  .btn-generate.loading .btn-text { display: none; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .count-badge {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    margin-top: 10px;
    padding: 5px 14px;
    background: rgba(99,102,241,0.08);
    border-radius: 20px;
    font-size: 12px;
    color: #818cf8;
    font-weight: 500;
  }
  .count-badge::before {
    content: '';
    width: 6px; height: 6px;
    border-radius: 50%;
    background: #6366f1;
    opacity: 0.6;
  }
  .config-panel {
    margin-bottom: 16px;
    padding: 16px 18px;
    background: #0e0e18;
    border-radius: 14px;
    border: 1px solid #1e1e30;
    text-align: left;
  }
  .config-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .config-row + .config-row { margin-top: 10px; }
  .config-row label {
    font-size: 12px;
    color: #64748b;
    white-space: nowrap;
    min-width: 72px;
    font-weight: 500;
  }
  .config-row input[type="number"],
  .config-row input[type="text"] {
    padding: 8px 11px;
    background: #12121e;
    border: 1px solid #1e1e32;
    border-radius: 8px;
    color: #e2e8f0;
    font-size: 13px;
    font-family: "SF Mono", "Consolas", monospace;
    outline: none;
    transition: border-color 0.2s, box-shadow 0.2s;
  }
  .config-row input[type="number"] { width: 72px; }
  .config-row input[type="text"] { flex: 1; width: auto; }
  .config-row input:focus { border-color: #6366f1; box-shadow: 0 0 0 3px rgba(99,102,241,0.1); }
  .config-row input[type="number"]::-webkit-inner-spin-button,
  .config-row input[type="number"]::-webkit-outer-spin-button { -webkit-appearance: none; margin: 0; }
  .config-row input[type="number"] { -moz-appearance: textfield; }
  .config-hint {
    font-size: 11px;
    color: #475569;
    margin-left: -4px;
  }
  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    font-weight: 600;
    color: #64748b;
    text-transform: uppercase;
    letter-spacing: 0.8px;
    margin-bottom: 12px;
  }
  .section-title::after {
    content: '';
    flex: 1;
    height: 1px;
    background: #1e1e30;
  }
  .info-bar {
    display: flex;
    gap: 2px;
    margin-bottom: 16px;
    padding: 10px 16px;
    background: #0e0e18;
    border-radius: 10px;
    border: 1px solid #1a1a2e;
    font-size: 12px;
  }
  .info-bar .info-item {
    display: flex;
    align-items: center;
    gap: 5px;
    flex: 1;
    justify-content: center;
    padding: 4px 0;
    border-right: 1px solid #1a1a2e;
  }
  .info-bar .info-item:last-child { border-right: none; }
  .info-bar .info-item .info-label { color: #475569; font-weight: 500; }
  .info-bar .info-item .info-value { color: #94a3b8; font-family: "SF Mono", "Consolas", monospace; font-size: 11px; }
  .card {
    background: #0e0e18;
    border-radius: 14px;
    padding: 20px;
    border: 1px solid #1a1a2e;
    min-height: 72px;
    transition: border-color 0.3s ease, box-shadow 0.3s ease;
  }
  .card.has-data { border-color: rgba(99,102,241,0.25); box-shadow: 0 0 0 1px rgba(99,102,241,0.06); }
  .card-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #475569;
    font-size: 13px;
    min-height: 72px;
    gap: 6px;
  }
  .card-empty .icon {
    width: 32px; height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 1.5px dashed #333;
    border-radius: 50%;
    font-size: 16px;
    color: #475569;
  }
  .field {
    display: flex;
    align-items: center;
    padding: 7px 0;
    font-size: 13px;
    gap: 10px;
  }
  .field + .field { border-top: 1px solid #161625; }
  .field-label {
    width: 68px;
    color: #64748b;
    flex-shrink: 0;
    font-size: 12px;
    font-weight: 500;
  }
  .field-value {
    flex: 1;
    color: #cbd5e1;
    font-family: "SF Mono", "Fira Code", "Consolas", monospace;
    font-size: 12px;
    word-break: break-all;
    user-select: all;
    padding: 2px 0;
  }
  .field-copy {
    background: transparent;
    border: 1px solid #1e1e32;
    color: #64748b;
    padding: 4px 10px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 11px;
    flex-shrink: 0;
    transition: all 0.15s;
  }
  .field-copy:hover { background: #1a1a2e; color: #94a3b8; border-color: #2a2a44; }
  .field-copy.copied { background: rgba(34,197,94,0.08); border-color: rgba(34,197,94,0.25); color: #4ade80; }
  .footer {
    margin-top: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }
  .footer .time-info {
    font-size: 11px;
    color: #475569;
    font-family: "SF Mono", "Consolas", monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .toast {
    position: fixed;
    bottom: 28px;
    left: 50%;
    transform: translateX(-50%) translateY(80px);
    background: #0e0e18;
    color: #e2e8f0;
    padding: 10px 22px;
    border-radius: 10px;
    font-size: 13px;
    opacity: 0;
    transition: all 0.3s ease;
    pointer-events: none;
    border: 1px solid #1e1e30;
    box-shadow: 0 12px 40px rgba(0,0,0,0.5);
    backdrop-filter: blur(8px);
    z-index: 100;
  }
  .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }
  .toast.success { border-color: rgba(34,197,94,0.3); color: #4ade80; }
  .toast.error { border-color: rgba(239,68,68,0.3); color: #f87171; }
</style>
</head>
<body>

<div class="container">
  <div class="header">
    <h1>账号生成器</h1>
    <p>点击按钮一键生成账号数据</p>
  </div>

  <div class="generate-area">
    <button class="btn-generate" id="btnGen" onclick="generate()">
      <span class="btn-text">生成账号</span>
      <span class="spinner"></span>
    </button>
    <div class="count-badge" id="countBadge">已生成 0 条</div>
  </div>

  <div class="config-panel" id="configPanel">
    <div class="config-row">
      <label for="randLenInput">随机字母长度</label>
      <input type="number" id="randLenInput" min="1" max="32" value="13" onchange="autoSaveConfig()">
      <span class="config-hint">1–32</span>
    </div>
    <div class="config-row">
      <label for="outputDirInput">输出目录</label>
      <input type="text" id="outputDirInput" value="./output" onchange="autoSaveConfig()">
    </div>
  </div>

  <div class="info-bar" id="infoBar">
    <span class="info-item">
      <span class="info-label">随机长度</span>
      <span class="info-value" id="infoRandLen">13</span>
    </span>
    <span class="info-item">
      <span class="info-label">输出目录</span>
      <span class="info-value" id="infoOutputDir">./output</span>
    </span>
  </div>

  <div class="card" id="resultCard">
    <div class="card-empty" id="emptyHint">
      <div class="icon">+</div>
      <div>点击上方按钮生成账号</div>
    </div>
    <div id="fieldsContainer" style="display:none"></div>
  </div>

  <div class="footer">
    <div class="time-info" id="timeInfo"></div>
    <div class="time-info" id="filePathInfo"></div>
  </div>
</div>

<div class="toast" id="toast"></div>

<script>
const fields = [
  { key: "first_name", label: "姓" },
  { key: "last_name", label: "名" },
  { key: "email", label: "邮箱", copyPrefix: true },
  { key: "placeholder:用户名", label: "邮箱用户名" },
  { key: "username", label: "用户名" },
  { key: "password", label: "密码" },
  { key: "phone", label: "手机号" },
];

let totalCount = 0;

function generate() {
  const btn = document.getElementById("btnGen");
  if (btn.classList.contains("loading")) return;
  btn.classList.add("loading");

  fetch("/api/generate")
    .then(r => r.json())
    .then(data => {
      if (data.error) { showToast(data.error, true); return; }
      totalCount = parseInt(data.count) || totalCount + 1;
      document.getElementById("countBadge").textContent = "已生成 " + totalCount + " 条";
      renderAccount(data);
      document.getElementById("timeInfo").textContent = data.created_at ? "生成时间: " + data.created_at : "";
      document.getElementById("filePathInfo").textContent = data.file_path ? "文件: " + data.file_path : "";
    })
    .catch(e => showToast("请求失败: " + e.message, true))
    .finally(() => btn.classList.remove("loading"));
}

function renderAccount(data) {
  const card = document.getElementById("resultCard");
  card.classList.add("has-data");
  document.getElementById("emptyHint").style.display = "none";
  const container = document.getElementById("fieldsContainer");
  container.style.display = "block";

  container.innerHTML = fields.map(f => {
    const val = data[f.key] || "";
    const copyVal = f.copyPrefix ? val.split('@')[0] : val;
    return '<div class="field">' +
      '<span class="field-label">' + f.label + '</span>' +
      '<span class="field-value">' + escapeHtml(val) + '</span>' +
      '<button class="field-copy" onclick="copyField(this, \'' + escapeHtml(copyVal) + '\')">复制</button>' +
    '</div>';
  }).join("");
}

function copyField(btn, val) {
  navigator.clipboard.writeText(val).then(() => {
    btn.textContent = "已复制";
    btn.classList.add("copied");
    setTimeout(() => { btn.textContent = "复制"; btn.classList.remove("copied"); }, 1500);
  }).catch(() => showToast("复制失败", true));
}

function escapeHtml(str) {
  const d = document.createElement("div");
  d.textContent = str;
  return d.innerHTML;
}

function showToast(msg, isError) {
  const t = document.getElementById("toast");
  t.textContent = msg;
  t.className = "toast" + (isError ? " error" : "");
  setTimeout(() => t.classList.add("show"), 10);
  setTimeout(() => t.classList.remove("show"), 2500);
}

function updateInfoBar() {
  const lenVal = document.getElementById("randLenInput").value;
  const dirVal = document.getElementById("outputDirInput").value;
  document.getElementById("infoRandLen").textContent = lenVal;
  document.getElementById("infoOutputDir").textContent = dirVal;
}

function loadConfig() {
  const saved = localStorage.getItem("account_gen_config");
  if (saved) {
    try {
      const cfg = JSON.parse(saved);
      if (cfg.rand_len) document.getElementById("randLenInput").value = cfg.rand_len;
      if (cfg.output_dir) document.getElementById("outputDirInput").value = cfg.output_dir;
      updateInfoBar();
      fetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: saved
      }).catch(() => {});
      return;
    } catch (e) {}
  }
  fetch("/api/config")
    .then(r => r.json())
    .then(data => {
      if (data.rand_len) localStorage.setItem("account_gen_config", JSON.stringify(data));
      if (data.rand_len) document.getElementById("randLenInput").value = data.rand_len;
      if (data.output_dir) document.getElementById("outputDirInput").value = data.output_dir;
      updateInfoBar();
    })
    .catch(() => {});
}

function autoSaveConfig() {
  const input = document.getElementById("randLenInput");
  const val = parseInt(input.value);
  if (!val || val < 1 || val > 32) {
    showToast("请输入 1–32 之间的数字", true);
    input.value = document.getElementById("infoRandLen").textContent;
    return;
  }
  const dirVal = document.getElementById("outputDirInput").value.trim() || "./output";

  fetch("/api/config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rand_len: val, output_dir: dirVal })
  })
    .then(r => r.json())
    .then(data => {
      localStorage.setItem("account_gen_config", JSON.stringify(data));
      if (data.rand_len) document.getElementById("randLenInput").value = data.rand_len;
      if (data.output_dir) document.getElementById("outputDirInput").value = data.output_dir;
      updateInfoBar();
    })
    .catch(() => showToast("保存失败", true));
}

loadConfig();
</script>
</body>
</html>`
