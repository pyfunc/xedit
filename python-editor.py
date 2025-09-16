# app.py - Edit2 Python Flask Server
from flask import Flask, render_template, request, jsonify, send_from_directory
from flask_cors import CORS
import os
import json
import yaml
import xml.etree.ElementTree as ET
from xml.dom import minidom
import subprocess
from datetime import datetime
from pathlib import Path

app = Flask(__name__)
CORS(app)

# Configuration
DATA_DIR = Path('./data')
DATA_DIR.mkdir(exist_ok=True)

def init_git():
    """Initialize Git repository if not exists"""
    try:
        subprocess.run(['git', 'rev-parse', '--git-dir'], 
                      cwd=DATA_DIR, capture_output=True, check=True)
    except:
        subprocess.run(['git', 'init'], cwd=DATA_DIR)
        subprocess.run(['git', 'config', 'user.email', 'edit2@local'], cwd=DATA_DIR)
        subprocess.run(['git', 'config', 'user.name', 'Edit2 User'], cwd=DATA_DIR)

def prettify_xml(elem):
    """Return a pretty-printed XML string"""
    rough_string = ET.tostring(elem, 'unicode')
    reparsed = minidom.parseString(rough_string)
    return reparsed.toprettyxml(indent="  ")

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/api/file/<filename>', methods=['GET'])
def get_file(filename):
    """Get file content"""
    filepath = DATA_DIR / filename
    
    try:
        # Create default file if not exists
        if not filepath.exists():
            ext = filepath.suffix
            default_content = ""
            
            if ext == '.json':
                default_content = json.dumps({
                    "name": "New File",
                    "created": datetime.now().isoformat()
                }, indent=2)
            elif ext in ['.yaml', '.yml']:
                default_content = yaml.dump({
                    "name": "New File",
                    "created": datetime.now().isoformat()
                })
            elif ext == '.xml':
                root = ET.Element("root")
                ET.SubElement(root, "n").text = "New File"
                ET.SubElement(root, "created").text = datetime.now().isoformat()
                default_content = prettify_xml(root)
            
            filepath.write_text(default_content)
            subprocess.run(['git', 'add', filename], cwd=DATA_DIR)
            subprocess.run(['git', 'commit', '-m', f'Initial: {filename}'], cwd=DATA_DIR)
        
        content = filepath.read_text()
        return jsonify({"content": content, "filename": filename})
    
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/api/file/<filename>', methods=['POST'])
def save_file(filename):
    """Save file content"""
    try:
        data = request.json
        content = data.get('content', '')
        filepath = DATA_DIR / filename
        
        # Validate content based on file type
        ext = filepath.suffix
        if ext == '.json':
            json.loads(content)  # Validate JSON
        elif ext in ['.yaml', '.yml']:
            yaml.safe_load(content)  # Validate YAML
        elif ext == '.xml':
            ET.fromstring(content)  # Validate XML
        
        # Save file
        filepath.write_text(content)
        
        # Git commit
        timestamp = datetime.now().isoformat()
        subprocess.run(['git', 'add', filename], cwd=DATA_DIR)
        subprocess.run(['git', 'commit', '-m', f'Update {filename}: {timestamp}'], cwd=DATA_DIR)
        
        # Get commit hash
        result = subprocess.run(['git', 'rev-parse', 'HEAD'], 
                              cwd=DATA_DIR, capture_output=True, text=True)
        commit_hash = result.stdout.strip()[:7]
        
        return jsonify({
            "success": True,
            "message": "File saved and committed",
            "commit": commit_hash,
            "timestamp": timestamp
        })
    
    except json.JSONDecodeError:
        return jsonify({"error": "Invalid JSON format"}), 400
    except yaml.YAMLError:
        return jsonify({"error": "Invalid YAML format"}), 400
    except ET.ParseError:
        return jsonify({"error": "Invalid XML format"}), 400
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/api/history/<filename>', methods=['GET'])
def get_history(filename):
    """Get file history from Git"""
    try:
        cmd = [
            'git', 'log', 
            '--pretty=format:{"hash":"%h","timestamp":"%ai","message":"%s"}',
            '-n', '20', '--', filename
        ]
        result = subprocess.run(cmd, cwd=DATA_DIR, capture_output=True, text=True)
        
        history = []
        for line in result.stdout.strip().split('\n'):
            if line:
                history.append(json.loads(line))
        
        return jsonify({"history": history})
    
    except:
        return jsonify({"history": []})

@app.route('/api/restore/<filename>/<hash>', methods=['POST'])
def restore_version(filename, hash):
    """Restore file to specific version"""
    try:
        # Get file content at specific commit
        cmd = ['git', 'show', f'{hash}:{filename}']
        result = subprocess.run(cmd, cwd=DATA_DIR, capture_output=True, text=True, check=True)
        content = result.stdout
        
        # Save as current version
        filepath = DATA_DIR / filename
        filepath.write_text(content)
        
        # Commit the restore
        subprocess.run(['git', 'add', filename], cwd=DATA_DIR)
        subprocess.run(['git', 'commit', '-m', f'Restored to version {hash}'], cwd=DATA_DIR)
        
        return jsonify({
            "success": True,
            "content": content,
            "message": f"Restored to version {hash}"
        })
    
    except Exception as e:
        return jsonify({"error": str(e)}), 500

@app.route('/api/files', methods=['GET'])
def list_files():
    """List all editable files"""
    try:
        valid_extensions = {'.json', '.yaml', '.yml', '.xml'}
        files = [f.name for f in DATA_DIR.iterdir() 
                if f.is_file() and f.suffix in valid_extensions]
        return jsonify({"files": files})
    
    except:
        return jsonify({"files": []})

if __name__ == '__main__':
    init_git()
    print("""
╔══════════════════════════════════════════╗
║         Edit2 - Visual Data Editor        ║
║            Python Flask Edition           ║
║                                          ║
║  Server running on http://localhost:3002 ║
║                                          ║
║  Usage:                                  ║
║  edit2 file.json                        ║
║  edit2 file.yaml                        ║
║  edit2 file.xml                         ║
╚══════════════════════════════════════════╝
    """)
    app.run(host='0.0.0.0', port=3002, debug=False)

# requirements.txt
"""
Flask==2.3.3
Flask-CORS==4.0.0
PyYAML==6.0.1
"""

# Dockerfile
"""
FROM python:3.11-slim

# Install git
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy application files
COPY . .

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 3002

# Start command
CMD ["python", "app.py"]
"""

# templates/index.html
HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit2 - Python Visual Editor</title>
    
    <!-- Monaco Editor -->
    <link rel="stylesheet" data-name="vs/editor/editor.main" href="https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.44.0/min/vs/editor/editor.main.min.css">
    
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(120deg, #a1c4fd 0%, #c2e9fb 100%);
            height: 100vh;
            display: flex;
            flex-direction: column;
        }
        
        .header {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 2px 20px rgba(0,0,0,0.1);
        }
        
        .header h1 {
            font-size: 1.5rem;
            background: linear-gradient(120deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        
        .controls {
            display: flex;
            gap: 1rem;
        }
        
        button {
            background: white;
            border: 2px solid #667eea;
            color: #667eea;
            padding: 0.5rem 1.5rem;
            border-radius: 25px;
            cursor: pointer;
            transition: all 0.3s ease;
            font-weight: 600;
        }
        
        button:hover {
            background: #667eea;
            color: white;
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }
        
        .save-btn {
            background: linear-gradient(120deg, #667eea 0%, #764ba2 100%);
            border: none;
            color: white;
        }
        
        .container {
            flex: 1;
            display: flex;
            padding: 1rem;
            gap: 1rem;
            overflow: hidden;
        }
        
        .panel {
            flex: 1;
            background: white;
            border-radius: 15px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.1);
            overflow: hidden;
            display: flex;
            flex-direction: column;
        }
        
        .panel-header {
            background: linear-gradient(120deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 1rem;
            font-weight: 600;
        }
        
        .panel-content {
            flex: 1;
            position: relative;
            overflow: auto;
        }
        
        #editor {
            width: 100%;
            height: 100%;
        }
        
        #visualEditor {
            padding: 1.5rem;
            font-family: 'Fira Code', 'Courier New', monospace;
            overflow: auto;
            height: 100%;
        }
        
        .tree-view {
            line-height: 1.8;
        }
        
        .tree-key {
            color: #2196F3;
            font-weight: bold;
        }
        
        .tree-value {
            color: #4CAF50;
        }
        
        .tree-string {
            color: #FF5722;
        }
        
        .error-panel {
            background: #f44336;
            color: white;
            padding: 1rem;
            margin: 1rem;
            border-radius: 10px;
            animation: shake 0.5s;
        }
        
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            25% { transform: translateX(-10px); }
            75% { transform: translateX(10px); }
        }
        
        .success-toast {
            position: fixed;
            top: 100px;
            right: 20px;
            background: linear-gradient(120deg, #84fab0 0%, #8fd3f4 100%);
            color: #1a1a1a;
            padding: 1rem 2rem;
            border-radius: 50px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            animation: slideIn 0.3s ease;
            z-index: 1000;
            font-weight: 600;
        }
        
        @keyframes slideIn {
            from { transform: translateX(400px); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
        
        .history-sidebar {
            position: fixed;
            right: -350px;
            top: 80px;
            width: 350px;
            height: calc(100vh - 80px);
            background: white;
            box-shadow: -5px 0 20px rgba(0,0,0,0.1);
            transition: right 0.3s ease;
            z-index: 999;
            border-radius: 15px 0 0 15px;
        }
        
        .history-sidebar.open {
            right: 0;
        }
        
        .history-header {
            background: linear-gradient(120deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 1rem;
            font-weight: 600;
        }
        
        .history-list {
            padding: 1rem;
            overflow-y: auto;
            max-height: calc(100% - 60px);
        }
        
        .history-item {
            background: #f5f5f5;
            border-radius: 10px;
            padding: 1rem;
            margin-bottom: 0.5rem;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        
        .history-item:hover {
            background: linear-gradient(120deg, #e3f2fd 0%, #f3e5f5 100%);
            transform: translateX(-5px);
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🐍 Edit2 - Python Visual Editor</h1>
        <div class="file-info">
            <span id="fileName">Loading...</span>
        </div>
        <div class="controls">
            <button onclick="saveFile()" class="save-btn">💾 Save</button>
            <button onclick="showHistory()">📜 History</button>
            <button onclick="formatCode()">✨ Format</button>
        </div>
    </div>
    
    <div class="container">
        <div class="panel">
            <div class="panel-header">Monaco Editor</div>
            <div class="panel-content">
                <div id="editor"></div>
            </div>
        </div>
        
        <div class="panel">
            <div class="panel-header">Visual Preview</div>
            <div class="panel-content">
                <div id="visualEditor"></div>
            </div>
        </div>
    </div>
    
    <div class="history-sidebar" id="historySidebar">
        <div class="history-header">
            Version History
            <button onclick="hideHistory()" style="float: right; background: none; border: none; color: white;">✕</button>
        </div>
        <div class="history-list" id="historyList"></div>
    </div>
    
    <script src="https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.44.0/min/vs/loader.min.js"></script>
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
        
        // Initialize Monaco Editor
        require.config({ paths: { vs: 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.44.0/min/vs' } });
        require(['vs/editor/editor.main'], function() {
            editor = monaco.editor.create(document.getElementById('editor'), {
                value: '',
                language: fileType,
                theme: 'vs-dark',
                automaticLayout: true,
                minimap: { enabled: false },
                fontSize: 14,
                wordWrap: 'on'
            });
            
            // Load file content
            loadFile();
            
            // Update visual editor on change
            editor.onDidChangeModelContent(debounce(updateVisual, 500));
        });
        
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
                const response = await fetch(`/api/file/${currentFile}`);
                const data = await response.json();
                editor.setValue(data.content);
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
                    html = '<div class="tree-view">' + renderObject(data) + '</div>';
                } else if (fileType === 'yaml') {
                    // Simple YAML preview
                    html = '<div class="tree-view"><pre>' + content + '</pre></div>';
                } else if (fileType === 'xml') {
                    // Simple XML preview with syntax highlighting
                    html = '<div class="tree-view"><pre>' + content.replace(/</g, '&lt;').replace(/>/g, '&gt;') + '</pre></div>';
                }
                
                visualDiv.innerHTML = html;
            } catch (error) {
                visualDiv.innerHTML = '<div class="error-panel">Parse Error: ' + error.message + '</div>';
            }
        }
        
        function renderObject(obj, indent = 0) {
            let html = '';
            const spaces = '  '.repeat(indent);
            
            if (typeof obj === 'object' && obj !== null) {
                if (Array.isArray(obj)) {
                    obj.forEach((item, i) => {
                        html += spaces + '<span class="tree-key">[' + i + ']</span>: ';
                        if (typeof item === 'object') {
                            html += '\\n' + renderObject(item, indent + 1);
                        } else {
                            html += '<span class="tree-value">' + item + '</span>\\n';
                        }
                    });
                } else {
                    Object.keys(obj).forEach(key => {
                        html += spaces + '<span class="tree-key">' + key + '</span>: ';
                        if (typeof obj[key] === 'object') {
                            html += '\\n' + renderObject(obj[key], indent + 1);
                        } else if (typeof obj[key] === 'string') {
                            html += '<span class="tree-string">"' + obj[key] + '"</span>\\n';
                        } else {
                            html += '<span class="tree-value">' + obj[key] + '</span>\\n';
                        }
                    });
                }
            }
            
            return html;
        }
        
        async function saveFile() {
            try {
                const content = editor.getValue();
                const response = await fetch(`/api/file/${currentFile}`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ content })
                });
                
                const data = await response.json();
                
                if (data.success) {
                    showToast('✅ File saved successfully!');
                } else {
                    alert('Error: ' + (data.error || 'Unknown error'));
                }
            } catch (error) {
                alert('Error saving file: ' + error.message);
            }
        }
        
        async function showHistory() {
            const sidebar = document.getElementById('historySidebar');
            sidebar.classList.add('open');
            
            try {
                const response = await fetch(`/api/history/${currentFile}`);
                const data = await response.json();
                
                const listDiv = document.getElementById('historyList');
                listDiv.innerHTML = '';
                
                data.history.forEach((item, index) => {
                    const div = document.createElement('div');
                    div.className = 'history-item';
                    div.innerHTML = `
                        <div>Version ${data.history.length - index}</div>
                        <div style="font-size: 0.9em; color: #666;">${item.timestamp}</div>
                        <div style="font-size: 0.8em; color: #999;">Commit: ${item.hash}</div>
                    `;
                    div.onclick = () => restoreVersion(item.hash);
                    listDiv.appendChild(div);
                });
            } catch (error) {
                console.error('Error loading history:', error);
            }
        }
        
        function hideHistory() {
            document.getElementById('historySidebar').classList.remove('open');
        }
        
        async function restoreVersion(hash) {
            if (confirm('Restore this version?')) {
                try {
                    const response = await fetch(`/api/restore/${currentFile}/${hash}`, {
                        method: 'POST'
                    });
                    const data = await response.json();
                    
                    if (data.success) {
                        editor.setValue(data.content);
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
            editor.getAction('editor.action.formatDocument').run();
            showToast('✨ Code formatted!');
        }
        
        function showToast(message) {
            const toast = document.createElement('div');
            toast.className = 'success-toast';
            toast.textContent = message;
            document.body.appendChild(toast);
            
            setTimeout(() => toast.remove(), 3000);
        }
    </script>
</body>
</html>"""

# edit2.sh - Launch script
LAUNCH_SCRIPT = """#!/bin/bash

# Edit2 - Python Visual Editor Launch Script

FILE=$1

if [ -z "$FILE" ]; then
    echo "Usage: edit2 <filename>"
    echo "Supported formats: .json, .yaml, .yml, .xml"
    exit 1
fi

# Check file extension
if [[ ! "$FILE" =~ \.(json|yaml|yml|xml)$ ]]; then
    echo "Error: Unsupported file format"
    exit 1
fi

# Build and run Docker container
echo "Starting Edit2 Python editor..."
docker build -t edit2-editor .
docker run -d --name edit2 -p 3002:3002 -v $(pwd)/data:/app/data edit2-editor

# Wait for server
sleep 2

# Open in browser
if command -v xdg-open > /dev/null; then
    xdg-open "http://localhost:3002?file=$FILE"
elif command -v open > /dev/null; then
    open "http://localhost:3002?file=$FILE"
else
    echo "Editor running at: http://localhost:3002?file=$FILE"
fi

echo "Edit2 is running. Press Ctrl+C to stop."
trap 'docker stop edit2 && docker rm edit2; exit' INT
while true; do sleep 1; done
"""