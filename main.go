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
	username := fmt.Sprintf("%s%s", userPre, randomStr)
	password := fmt.Sprintf("%s%s", passwordPre, timeStr)
	createdAt := now.Format("2006-01-02 15:04:05")

	content := fmt.Sprintf(`姓：%s
名：%s
邮箱：%s
用户名：%s
密码：%s
手机号：%s

`,
		userPre,
		userPre,
		email,
		username,
		password,
		numStr,
	)

	displayText := fmt.Sprintf("姓：%s\n名：%s\n邮箱：%s\n用户名：%s\n密码：%s\n手机号：%s",
		userPre, userPre, email, username, password, numStr)

	account := map[string]string{
		"name":          userPre,
		"firstName":     userPre,
		"first_name":    userPre,
		"lastName":      userPre,
		"last_name":     userPre,
		"email":         email,
		"username":      username,
		"password":      password,
		"phone":         numStr,
		"mobile_number": numStr,
		"created_at":    createdAt,
		"display":       displayText,
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

	jsonDir := dir

	http.HandleFunc("/api/latest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		jsonPath := filepath.Join(jsonDir, jsonFile)
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
			json.NewEncoder(w).Encode(map[string]int{"rand_len": randLen})
		case "POST":
			var body map[string]int
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
				return
			}
			if n, ok := body["rand_len"]; ok && n > 0 {
				randLen = n
			}
			json.NewEncoder(w).Encode(map[string]int{"rand_len": randLen})
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
			fmt.Printf("> HTTP服务: http://localhost:%s\n", currentPort)
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
    background: #0f0f1a;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 20px;
  }
  .container {
    width: 100%;
    max-width: 560px;
    background: #1a1a2e;
    border-radius: 24px;
    padding: 40px 32px;
    box-shadow: 0 20px 60px rgba(0,0,0,0.5);
    border: 1px solid #2a2a4a;
  }
  .header {
    text-align: center;
    margin-bottom: 32px;
  }
  .header h1 {
    font-size: 26px;
    font-weight: 700;
    background: linear-gradient(135deg, #667eea, #764ba2);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }
  .header p {
    color: #888;
    font-size: 14px;
    margin-top: 6px;
  }
  .generate-area {
    text-align: center;
    margin-bottom: 28px;
  }
  .btn-generate {
    width: 100%;
    padding: 16px;
    font-size: 18px;
    font-weight: 600;
    border: none;
    border-radius: 14px;
    background: linear-gradient(135deg, #667eea, #764ba2);
    color: #fff;
    cursor: pointer;
    transition: all 0.25s ease;
    letter-spacing: 1px;
    position: relative;
    overflow: hidden;
  }
  .btn-generate:hover { transform: translateY(-2px); box-shadow: 0 8px 30px rgba(102,126,234,0.4); }
  .btn-generate:active { transform: translateY(0); }
  .btn-generate:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }
  .btn-generate .spinner {
    display: none;
    width: 22px; height: 22px;
    border: 3px solid rgba(255,255,255,0.3);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    margin: 0 auto;
  }
  .btn-generate.loading .spinner { display: block; }
  .btn-generate.loading .btn-text { display: none; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .count-badge {
    display: inline-block;
    margin-top: 10px;
    padding: 4px 14px;
    background: #2a2a4a;
    border-radius: 20px;
    font-size: 13px;
    color: #aaa;
  }
  .config-toggle {
    margin-top: 12px;
    font-size: 13px;
    color: #666;
    cursor: pointer;
    user-select: none;
    transition: color 0.2s;
    display: inline-block;
  }
  .config-toggle:hover { color: #667eea; }
  .config-panel {
    margin-top: 12px;
    padding: 16px 20px;
    background: #12122a;
    border-radius: 12px;
    border: 1px solid #2a2a4a;
    display: none;
    text-align: left;
  }
  .config-panel.open { display: block; }
  .config-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .config-row label {
    font-size: 13px;
    color: #888;
    white-space: nowrap;
  }
  .config-row input[type="number"] {
    width: 80px;
    padding: 6px 10px;
    background: #1a1a2e;
    border: 1px solid #3a3a5a;
    border-radius: 8px;
    color: #e0e0e0;
    font-size: 14px;
    font-family: "SF Mono", "Consolas", monospace;
    outline: none;
    transition: border-color 0.2s;
  }
  .config-row input[type="number"]:focus { border-color: #667eea; }
  .config-row input[type="number"]::-webkit-inner-spin-button { opacity: 1; }
  .config-hint {
    font-size: 12px;
    color: #555;
    margin-left: auto;
  }
  .config-save-btn {
    padding: 6px 16px;
    background: #667eea;
    border: none;
    border-radius: 8px;
    color: #fff;
    font-size: 13px;
    cursor: pointer;
    transition: background 0.2s;
  }
  .config-save-btn:hover { background: #5a6fd6; }
  .config-save-btn:disabled { opacity: 0.4; cursor: not-allowed; }
  .card {
    background: #12122a;
    border-radius: 16px;
    padding: 24px;
    margin-top: 8px;
    border: 1px solid #2a2a4a;
    min-height: 80px;
    transition: all 0.3s ease;
  }
  .card.has-data { border-color: #667eea; }
  .card-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #555;
    font-size: 14px;
    min-height: 80px;
  }
  .card-empty .icon { font-size: 32px; margin-bottom: 8px; opacity: 0.4; }
  .field {
    display: flex;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid #1e1e3a;
    font-size: 14px;
  }
  .field:last-child { border-bottom: none; }
  .field-label {
    width: 72px;
    color: #888;
    flex-shrink: 0;
  }
  .field-value {
    flex: 1;
    color: #e0e0e0;
    font-family: "SF Mono", "Fira Code", "Consolas", monospace;
    font-size: 13px;
    word-break: break-all;
    user-select: all;
    padding-right: 8px;
  }
  .field-copy {
    background: none;
    border: 1px solid #3a3a5a;
    color: #888;
    padding: 3px 10px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 12px;
    flex-shrink: 0;
    transition: all 0.2s;
  }
  .field-copy:hover { background: #2a2a4a; color: #fff; border-color: #667eea; }
  .field-copy.copied { background: #2d6a4f; border-color: #52b788; color: #52b788; }
  .footer {
    margin-top: 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .time-info { font-size: 12px; color: #555; }
  .toast {
    position: fixed;
    bottom: 30px;
    left: 50%;
    transform: translateX(-50%) translateY(80px);
    background: #2d6a4f;
    color: #fff;
    padding: 10px 24px;
    border-radius: 10px;
    font-size: 14px;
    opacity: 0;
    transition: all 0.35s ease;
    pointer-events: none;
  }
  .toast.show { opacity: 1; transform: translateX(-50%) translateY(0); }
  .toast.error { background: #9b2226; }
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
    <div class="config-toggle" id="configToggle" onclick="toggleConfig()">设置 ⚙</div>
    <div class="config-panel" id="configPanel">
      <div class="config-row">
        <label for="randLenInput">随机字母长度</label>
        <input type="number" id="randLenInput" min="1" max="32" value="13">
        <span class="config-hint">1–32</span>
        <button class="config-save-btn" id="configSaveBtn" onclick="saveConfig()">应用</button>
      </div>
    </div>
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
  </div>
</div>

<div class="toast" id="toast"></div>

<script>
const fields = [
  { key: "first_name", label: "姓" },
  { key: "last_name", label: "名" },
  { key: "email", label: "邮箱", copyPrefix: true },
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

function toggleConfig() {
  const panel = document.getElementById("configPanel");
  panel.classList.toggle("open");
}

function loadConfig() {
  fetch("/api/config")
    .then(r => r.json())
    .then(data => {
      if (data.rand_len) {
        document.getElementById("randLenInput").value = data.rand_len;
      }
    })
    .catch(() => {});
}

function saveConfig() {
  const input = document.getElementById("randLenInput");
  const val = parseInt(input.value);
  if (!val || val < 1 || val > 32) {
    showToast("请输入 1–32 之间的数字", true);
    return;
  }
  const btn = document.getElementById("configSaveBtn");
  btn.disabled = true;
  btn.textContent = "保存中…";

  fetch("/api/config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rand_len: val })
  })
    .then(r => r.json())
    .then(data => {
      if (data.rand_len) {
        document.getElementById("randLenInput").value = data.rand_len;
        showToast("已更新为 " + data.rand_len + " 位随机字母");
      }
    })
    .catch(() => showToast("保存失败", true))
    .finally(() => {
      btn.disabled = false;
      btn.textContent = "应用";
    });
}

loadConfig();
</script>
</body>
</html>`
