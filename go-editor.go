// main.go - Edit3 Go Gin Server
package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/cors"
    "gopkg.in/yaml.v3"
)

const (
    DataDir = "./data"
    Port    = ":3003"
)

type FileResponse struct {
    Content  string `json:"content"`
    Filename string `json:"filename"`
}

type SaveRequest struct {
    Content string `json:"content"`
}

type SaveResponse struct {
    Success   bool   `json:"success"`
    Message   string `json:"message"`
    Commit    string `json:"commit"`
    Timestamp string `json:"timestamp"`
}

type HistoryItem struct {
    Hash      string `json:"hash"`
    Timestamp string `json:"timestamp"`
    Message   string `json:"message"`
}

type HistoryResponse struct {
    History []HistoryItem `json:"history"`
}

func initGit() {
    // Check if git is initialized
    cmd := exec.Command("git", "rev-parse", "--git-dir")
    cmd.Dir = DataDir
    if err := cmd.Run(); err != nil {
        // Initialize git
        exec.Command("git", "init").Dir = DataDir
        exec.Command("git", "config", "user.email", "edit3@local").Dir = DataDir
        exec.Command("git", "config", "user.name", "Edit3 User").Dir = DataDir
    }
}

func ensureDataDir() {
    if _, err := os.Stat(DataDir); os.IsNotExist(err) {
        os.MkdirAll(DataDir, 0755)
    }
}

func validateContent(content string, fileType string) error {
    switch fileType {
    case "json":
        var js interface{}
        return json.Unmarshal([]byte(content), &js)
    case "yaml", "yml":
        var y interface{}
        return yaml.Unmarshal([]byte(content), &y)
    case "xml":
        return xml.Unmarshal([]byte(content), new(interface{}))
    }
    return nil
}

func getFileType(filename string) string {
    ext := filepath.Ext(filename)
    return strings.TrimPrefix(ext, ".")
}

func main() {
    // Setup
    ensureDataDir()
    initGit()

    // Gin setup
    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()
    r.Use(cors.Default())

    // Serve HTML
    r.StaticFile("/", "./static/index.html")
    r.Static("/static", "./static")

    // API Routes
    r.GET("/api/file/:filename", getFile)
    r.POST("/api/file/:filename", saveFile)
    r.GET("/api/history/:filename", getHistory)
    r.POST("/api/restore/:filename/:hash", restoreVersion)
    r.GET("/api/files", listFiles)

    fmt.Println(`
╔══════════════════════════════════════════╗
║         Edit3 - Visual Data Editor        ║
║            Go Gin Edition                 ║
║                                          ║
║  Server running on http://localhost:3003 ║
║                                          ║
║  Usage:                                  ║
║  edit3 file.json                        ║
║  edit3 file.yaml                        ║
║  edit3 file.xml                         ║
╚══════════════════════════════════════════╝
    `)

    r.Run(Port)
}

func getFile(c *gin.Context) {
    filename := c.Param("filename")
    filepath := filepath.Join(DataDir, filename)

    // Check if file exists, create default if not
    if _, err := os.Stat(filepath); os.IsNotExist(err) {
        createDefaultFile(filepath, filename)
    }

    content, err := ioutil.ReadFile(filepath)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, FileResponse{
        Content:  string(content),
        Filename: filename,
    })
}

func createDefaultFile(filepath, filename string) {
    var defaultContent string
    fileType := getFileType(filename)

    switch fileType {
    case "json":
        data := map[string]interface{}{
            "name":    "New File",
            "created": time.Now().Format(time.RFC3339),
        }
        bytes, _ := json.MarshalIndent(data, "", "  ")
        defaultContent = string(bytes)

    case "yaml", "yml":
        defaultContent = fmt.Sprintf("name: New File\ncreated: %s\n", time.Now().Format(time.RFC3339))

    case "xml":
        defaultContent = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<root>
  <name>New File</name>
  <created>%s</created>
</root>`, time.Now().Format(time.RFC3339))
    }

    ioutil.WriteFile(filepath, []byte(defaultContent), 0644)

    // Git commit
    cmd := exec.Command("git", "add", filename)
    cmd.Dir = DataDir
    cmd.Run()

    cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Initial: %s", filename))
    cmd.Dir = DataDir
    cmd.Run()
}

func saveFile(c *gin.Context) {
    filename := c.Param("filename")
    filepath := filepath.Join(DataDir, filename)

    var req SaveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Validate content
    fileType := getFileType(filename)
    if err := validateContent(req.Content, fileType); err != nil {
        c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid %s format: %v", strings.ToUpper(fileType), err)})
        return
    }

    // Save file
    if err := ioutil.WriteFile(filepath, []byte(req.Content), 0644); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Git commit
    timestamp := time.Now().Format(time.RFC3339)

    cmd := exec.Command("git", "add", filename)
    cmd.Dir = DataDir
    cmd.Run()

    cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Update %s: %s", filename, timestamp))
    cmd.Dir = DataDir
    cmd.Run()

    // Get commit hash
    cmd = exec.Command("git", "rev-parse", "HEAD")
    cmd.Dir = DataDir
    output, _ := cmd.Output()
    hash := strings.TrimSpace(string(output))[:7]

    c.JSON(200, SaveResponse{
        Success:   true,
        Message:   "File saved and committed",
        Commit:    hash,
        Timestamp: timestamp,
    })
}

func getHistory(c *gin.Context) {
    filename := c.Param("filename")

    cmd := exec.Command("git", "log", "--pretty=format:%h|%ai|%s", "-n", "20", "--", filename)
    cmd.Dir = DataDir
    output, err := cmd.Output()

    if err != nil || len(output) == 0 {
        c.JSON(200, HistoryResponse{History: []HistoryItem{}})
        return
    }

    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    history := make([]HistoryItem, 0)

    for _, line := range lines {
        parts := strings.Split(line, "|")
        if len(parts) == 3 {
            history = append(history, HistoryItem{
                Hash:      parts[0],
                Timestamp: parts[1],
                Message:   parts[2],
            })
        }
    }

    c.JSON(200, HistoryResponse{History: history})
}

func restoreVersion(c *gin.Context) {
    filename := c.Param("filename")
    hash := c.Param("hash")
    filepath := filepath.Join(DataDir, filename)

    // Get file content at specific commit
    cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", hash, filename))
    cmd.Dir = DataDir
    output, err := cmd.Output()

    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Save as current version
    if err := ioutil.WriteFile(filepath, output, 0644); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Commit the restore
    cmd = exec.Command("git", "add", filename)
    cmd.Dir = DataDir
    cmd.Run()

    cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("Restored to version %s", hash))
    cmd.Dir = DataDir
    cmd.Run()

    c.JSON(200, gin.H{
        "success": true,
        "content": string(output),
        "message": fmt.Sprintf("Restored to version %s", hash),
    })
}

func listFiles(c *gin.Context) {
    files, err := ioutil.ReadDir(DataDir)
    if err != nil {
        c.JSON(200, gin.H{"files": []string{}})
        return
    }

    validExtensions := map[string]bool{
        ".json": true,
        ".yaml": true,
        ".yml":  true,
        ".xml":  true,
    }

    var fileList []string
    for _, file := range files {
        if !file.IsDir() {
            ext := filepath.Ext(file.Name())
            if validExtensions[ext] {
                fileList = append(fileList, file.Name())
            }
        }
    }

    c.JSON(200, gin.H{"files": fileList})
}

// go.mod
/*
module edit3

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gin-contrib/cors v1.4.0
    gopkg.in/yaml.v3 v3.0.1
)
*/

// Dockerfile
/*
FROM golang:1.21-alpine AS builder

# Install git
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -o edit3 .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/edit3 .
COPY --from=builder /app/static ./static

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 3003

# Run application
CMD ["./edit3"]
*/

// static/index.html
const HTML_CONTENT = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit3 - Go Visual Editor</title>
    
    <!-- Ace Editor -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/ace/1.32.2/ace.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/ace/1.32.2/ext-language_tools.js"></script>
    
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
            height: 100vh;
            display: flex;
            flex-direction: column;
            color: #fff;
        }
        
        .header {
            background: rgba(0, 0, 0, 0.2);
            backdrop-filter: blur(20px);
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .header h1 {
            font-size: 1.5rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        .logo {
            width: 32px;
            height: 32px;
            background: linear-gradient(135deg, #00C9FF 0%, #92FE9D 100%);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: bold;
            color: #1e3c72;
        }
        
        .file-badge {
            background: rgba(255, 255, 255, 0.1);
            padding: 0.5rem 1rem;
            border-radius: 20px;
            font-size: 0.9rem;
            backdrop-filter: blur(10px);
        }
        
        .controls {
            display: flex;
            gap: 1rem;
        }
        
        button {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255, 255, 255, 0.2);
            color: white;
            padding: 0.6rem 1.5rem;
            border-radius: 10px;
            cursor: pointer;
            transition: all 0.3s ease;
            font-weight: 600;
            font-size: 0.9rem;
        }
        
        button:hover {
            background: rgba(255, 255, 255, 0.2);
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0, 0, 0, 0.2);
        }
        
        .save-btn {
            background: linear-gradient(135deg, #00C9FF 0%, #92FE9D 100%);
            border: none;
            color: #1e3c72;
        }
        
        .container {
            flex: 1;
            display: flex;
            padding: 1.5rem;
            gap: 1.5rem;
            overflow: hidden;
        }
        
        .panel {
            flex: 1;
            background: rgba(255, 255, 255, 0.05);
            backdrop-filter: blur(20px);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 20px;
            overflow: hidden;
            display: flex;
            flex-direction: column;
        }
        
        .panel-header {
            background: rgba(0, 0, 0, 0.2);
            padding: 1rem 1.5rem;
            font-weight: 600;
            font-size: 0.9rem;
            text-transform: uppercase;
            letter-spacing: 1px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        
        .panel-content {
            flex: 1;
            position: relative;
        }
        
        #editor {
            width: 100%;
            height: 100%;
            font-size: 14px;
        }
        
        #visualEditor {
            padding: 2rem;
            font-family: 'Fira Code', monospace;
            color: #fff;
            overflow: auto;
            height: 100%;
        }
        
        .tree-view {
            line-height: 2;
            font-size: 14px;
        }
        
        .tree-key {
            color: #00C9FF;
            font-weight: bold;
        }
        
        .tree-value {
            color: #92FE9D;
        }
        
        .tree-string {
            color: #FFB75E;
        }
        
        .tree-number {
            color: #FF6B6B;
        }
        
        .tree-boolean {
            color: #C792EA;
        }
        
        .tree-null {
            color: #89DDFF;
        }
        
        .error-box {
            background: linear-gradient(135deg, #F93822 0%, #F9642D 100%);
            color: white;
            padding: 1.5rem;
            margin: 1rem;
            border-radius: 15px;
            font-weight: 600;
        }
        
        .toast {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            background: linear-gradient(135deg, #00C9FF 0%, #92FE9D 100%);
            color: #1e3c72;
            padding: 1rem 2rem;
            border-radius: 15px;
            font-weight: 600;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
            animation: slideUp 0.3s ease;
            z-index: 1000;
        }
        
        @keyframes slideUp {
            from {
                transform: translateY(100px);
                opacity: 0;
            }
            to {
                transform: translateY(0);
                opacity: 1;
            }
        }
        
        .history-modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.8);
            backdrop-filter: blur(10px);
            z-index: 999;
        }
        
        .history-modal.show {
            display: flex;
            align-items: center;
            justify-content: center;
        }
        
        .history-content {
            background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
            border-radius: 20px;
            width: 600px;
            max-height: 80vh;
            overflow: hidden;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
        }
        
        .history-header {
            background: rgba(0, 0, 0, 0.2);
            padding: 1.5rem;
            font-weight: 600;
            font-size: 1.2rem;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        
        .history-list {
            max-height: 500px;
            overflow-y: auto;
            padding: 1rem;
        }
        
        .history-item {
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 15px;
            padding: 1rem 1.5rem;
            margin-bottom: 1rem;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        
        .history-item:hover {
            background: rgba(255, 255, 255, 0.1);
            transform: translateX(10px);
        }
        
        .history-version {
            font-weight: 600;
            color: #00C9FF;
            margin-bottom: 0.5rem;
        }
        
        .history-time {
            font-size: 0.85rem;
            color: rgba(255, 255, 255, 0.7);
        }
        
        .history-hash {
            font-family: monospace;
            font-size: 0.8rem;
            color: rgba(255, 255, 255, 0.5);
            margin-top: 0.5rem;
        }
        
        pre {
            background: rgba(0, 0, 0, 0.3);
            padding: 1rem;
            border-radius: 10px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>
            <div class="logo">G</div>
            Edit3 - Go Visual Editor
        </h1>
        <div class="file-badge" id="fileName">Loading...</div>
        <div class="controls">
            <button onclick="saveFile()" class="save-btn">💾 Save & Commit</button>
            <button onclick="showHistory()">📜 History</button>
            <button onclick="formatCode()">✨ Format</button>
            <button onclick="reloadFile()">🔄 Reload</button>
        </div>
    </div>
    
    <div class="container">
        <div class="panel">
            <div class="panel-header">🖊️ Ace Editor</div>
            <div class="panel-content">
                <div id="editor"></div>
            </div>
        </div>
        
        <div class="panel">
            <div class="panel-header">👁️ Visual Preview</div>
            <div class="panel-content">
                <div id="visualEditor"></div>
            </div>
        </div>
    </div>
    
    <div class="history-modal" id="historyModal">
        <div class="history-content">
            <div class="history-header">
                Version History
                <button onclick="hideHistory()" style="background: none; border: none; font-size: 1.5rem;">×</button>
            </div>
            <div class="history-list" id="historyList"></div>
        </div>
    </div>
    
    <script>
        let editor;
        let currentFile = '';
        let fileType = '';
        
        // Get filename from URL
        const urlParams = new URLSearchParams(window.location.search);
        currentFile = urlParams.get('file') || 'example.json';
        document.getElementById('fileName').textContent = currentFile;
        
        // Detect file type
        if (currentFile.endsWith('.json')) fileType = 'json';
        else if (currentFile.endsWith('.yaml') || currentFile.endsWith('.yml')) fileType = 'yaml';
        else if (currentFile.endsWith('.xml')) fileType = 'xml';
        
        // Initialize Ace Editor
        editor = ace.edit("editor");
        editor.setTheme("ace/theme/dracula");
        editor.session.setMode("ace/mode/" + fileType);
        editor.setOptions({
            enableBasicAutocompletion: true,
            enableLiveAutocompletion: true,
            fontSize: 14,
            showPrintMargin: false,
            wrap: true
        });
        
        // Load file
        loadFile();
        
        // Update visual on change
        editor.on('change', debounce(updateVisual, 500));
        
        function debounce(func, wait) {
            let timeout;
            return function executedFunction(...args) {
                const later = () => {
                    clearTimeout(timeout);
                    func(...args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        }
        
        async function loadFile() {
            try {
                const response = await fetch('/api/file/' + currentFile);
                const data = await response.json();
                editor.setValue(data.content, -1);
                updateVisual();
            } catch (error) {
                console.error('Error loading file:', error);
            }
        }
        
        function updateVisual() {
            const content = editor.getValue();
            const visualDiv = document.getElementById('visualEditor');
            
            try {
                let html = '';
                
                if (fileType === 'json') {
                    const data = JSON.parse(content);
                    html = '<div class="tree-view">' + renderJSON(data, 0) + '</div>';
                } else if (fileType === 'yaml' || fileType === 'yml') {
                    html = '<div class="tree-view"><pre>' + escapeHtml(content) + '</pre></div>';
                } else if (fileType === 'xml') {
                    html = '<div class="tree-view"><pre>' + highlightXML(content) + '</pre></div>';
                }
                
                visualDiv.innerHTML = html;
            } catch (error) {
                visualDiv.innerHTML = '<div class="error-box">⚠️ Parse Error: ' + error.message + '</div>';
            }
        }
        
        function renderJSON(obj, indent) {
            let html = '';
            const spaces = '  '.repeat(indent);
            
            if (obj === null) {
                return '<span class="tree-null">null</span>';
            } else if (typeof obj === 'boolean') {
                return '<span class="tree-boolean">' + obj + '</span>';
            } else if (typeof obj === 'number') {
                return '<span class="tree-number">' + obj + '</span>';
            } else if (typeof obj === 'string') {
                return '<span class="tree-string">"' + escapeHtml(obj) + '"</span>';
            } else if (Array.isArray(obj)) {
                if (obj.length === 0) return '[]';
                html += '[\\n';
                obj.forEach((item, i) => {
                    html += spaces + '  ' + renderJSON(item, indent + 1);
                    if (i < obj.length - 1) html += ',';
                    html += '\\n';
                });
                html += spaces + ']';
                return html;
            } else if (typeof obj === 'object') {
                const keys = Object.keys(obj);
                if (keys.length === 0) return '{}';
                html += '{\\n';
                keys.forEach((key, i) => {
                    html += spaces + '  <span class="tree-key">"' + key + '"</span>: ';
                    html += renderJSON(obj[key], indent + 1);
                    if (i < keys.length - 1) html += ',';
                    html += '\\n';
                });
                html += spaces + '}';
                return html;
            }
            return html;
        }
        
        function highlightXML(xml) {
            return xml
                .replace(/</g, '&lt;')
                .replace(/>/g, '&gt;')
                .replace(/(&lt;\\/?)(\\w+)/g, '$1<span class="tree-key">$2</span>')
                .replace(/(\\w+)(=)(".*?")/g, '<span class="tree-string">$1</span>$2<span class="tree-value">$3</span>');
        }
        
        function escapeHtml(text) {
            const map = {
                '&': '&amp;',
                '<': '&lt;',
                '>': '&gt;',
                '"': '&quot;',
                "'": '&#039;'
            };
            return text.replace(/[&<>"']/g, m => map[m]);
        }
        
        async function saveFile() {
            try {
                const content = editor.getValue();
                const response = await fetch('/api/file/' + currentFile, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    showToast('✅ File saved and committed!');
                } else {
                    alert('Error: ' + (data.error || 'Unknown error'));
                }
            } catch (error) {
                alert('Error saving file: ' + error.message);
            }
        }
        
        async function showHistory() {
            const modal = document.getElementById('historyModal');
            modal.classList.add('show');
            
            try {
                const response = await fetch('/api/history/' + currentFile);
                const data = await response.json();
                
                const listDiv = document.getElementById('historyList');
                listDiv.innerHTML = '';
                
                if (data.history && data.history.length > 0) {
                    data.history.forEach((item, index) => {
                        const div = document.createElement('div');
                        div.className = 'history-item';
                        div.innerHTML = \`
                            <div class="history-version">Version #\${data.history.length - index}</div>
                            <div class="history-time">\${item.timestamp}</div>
                            <div class="history-hash">Commit: \${item.hash}</div>
                        \`;
                        div.onclick = () => restoreVersion(item.hash);
                        listDiv.appendChild(div);
                    });
                } else {
                    listDiv.innerHTML = '<div style="text-align: center; color: rgba(255,255,255,0.5);">No history available</div>';
                }
            } catch (error) {
                console.error('Error loading history:', error);
            }
        }
        
        function hideHistory() {
            document.getElementById('historyModal').classList.remove('show');
        }
        
        async function restoreVersion(hash) {
            if (confirm('Restore this version? Current changes will be saved as a new commit.')) {
                try {
                    const response = await fetch('/api/restore/' + currentFile + '/' + hash, {
                        method: 'POST'
                    });
                    const data = await response.json();
                    
                    if (data.success) {
                        editor.setValue(data.content, -1);
                        updateVisual();
                        hideHistory();
                        showToast('✅ Version restored!');
                    }
                } catch (error) {
                    alert('Error restoring version: ' + error.message);
                }
            }
        }
        
        function formatCode() {
            const content = editor.getValue();
            let formatted = content;
            
            try {
                if (fileType === 'json') {
                    const data = JSON.parse(content);
                    formatted = JSON.stringify(data, null, 2);
                    editor.setValue(formatted, -1);
                    showToast('✨ Code formatted!');
                } else {
                    showToast('🎨 Format available for JSON only');
                }
            } catch (error) {
                alert('Error: Cannot format invalid JSON');
            }
        }
        
        function reloadFile() {
            loadFile();
            showToast('🔄 File reloaded!');
        }
        
        function showToast(message) {
            const toast = document.createElement('div');
            toast.className = 'toast';
            toast.textContent = message;
            document.body.appendChild(toast);
            
            setTimeout(() => toast.remove(), 3000);
        }
    </script>
</body>
</html>`;

// edit3.sh - Launch script
const LAUNCH_SCRIPT = `#!/bin/bash

# Edit3 - Go Visual Editor Launch Script

FILE=$1

if [ -z "$FILE" ]; then
    echo "Usage: edit3 <filename>"
    echo "Supported formats: .json, .yaml, .yml, .xml"
    exit 1
fi

# Check file extension
if [[ ! "$FILE" =~ \.(json|yaml|yml|xml)$ ]]; then
    echo "Error: Unsupported file format"
    echo "Supported formats: .json, .yaml, .yml, .xml"
    exit 1
fi

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker is not running"
    exit 1
fi

# Build and run container
echo "Starting Edit3 Go editor..."
docker build -t edit3-editor .
docker run -d \\
    --name edit3 \\
    -p 3003:3003 \\
    -v $(pwd)/data:/app/data \\
    edit3-editor

# Wait for server to start
echo -n "Waiting for server..."
for i in {1..10}; do
    if curl -s http://localhost:3003 > /dev/null; then
        echo " Ready!"
        break
    fi
    echo -n "."
    sleep 1
done

# Open in browser
if command -v xdg-open > /dev/null; then
    xdg-open "http://localhost:3003?file=$FILE"
elif command -v open > /dev/null; then
    open "http://localhost:3003?file=$FILE"
else
    echo "Editor running at: http://localhost:3003?file=$FILE"
fi

echo ""
echo "Edit3 is running. Press Ctrl+C to stop."
echo "Editing: $FILE"
echo ""

# Cleanup on exit
trap 'echo "Stopping Edit3..."; docker stop edit3 && docker rm edit3; exit' INT TERM

# Keep script running
while true; do sleep 1; done
`;`