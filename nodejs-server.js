// server.js - Edit1 Node.js Server
const express = require('express');
const fs = require('fs').promises;
const path = require('path');
const { execSync, exec } = require('child_process');
const bodyParser = require('body-parser');
const cors = require('cors');

const app = express();
const PORT = 3001;

// Middleware
app.use(cors());
app.use(bodyParser.json({ limit: '50mb' }));
app.use(bodyParser.urlencoded({ extended: true, limit: '50mb' }));
app.use(express.static('public'));

// Initialize Git repo if not exists
async function initGit(dir) {
    try {
        execSync('git rev-parse --git-dir', { cwd: dir });
    } catch {
        execSync('git init', { cwd: dir });
        execSync('git config user.email "edit1@local"', { cwd: dir });
        execSync('git config user.name "Edit1 User"', { cwd: dir });
    }
}

// Ensure data directory exists
const DATA_DIR = path.join(__dirname, 'data');
async function ensureDataDir() {
    try {
        await fs.access(DATA_DIR);
    } catch {
        await fs.mkdir(DATA_DIR, { recursive: true });
    }
    await initGit(DATA_DIR);
}

// Routes
app.get('/', async (req, res) => {
    const html = await fs.readFile(path.join(__dirname, 'public', 'index.html'), 'utf8');
    res.send(html);
});

// Get file content
app.get('/api/file/:filename', async (req, res) => {
    try {
        const filename = req.params.filename;
        const filepath = path.join(DATA_DIR, filename);
        
        // Check if file exists, if not create with default content
        try {
            await fs.access(filepath);
        } catch {
            const ext = path.extname(filename);
            let defaultContent = '';
            
            if (ext === '.json') {
                defaultContent = JSON.stringify({ 
                    name: "New File", 
                    created: new Date().toISOString() 
                }, null, 2);
            } else if (ext === '.yaml' || ext === '.yml') {
                defaultContent = `name: New File\ncreated: ${new Date().toISOString()}`;
            } else if (ext === '.xml') {
                defaultContent = `<?xml version="1.0" encoding="UTF-8"?>\n<root>\n  <name>New File</name>\n  <created>${new Date().toISOString()}</created>\n</root>`;
            }
            
            await fs.writeFile(filepath, defaultContent);
            execSync(`git add "${filename}" && git commit -m "Initial: ${filename}"`, { cwd: DATA_DIR });
        }
        
        const content = await fs.readFile(filepath, 'utf8');
        res.json({ content, filename });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// Save file
app.post('/api/file/:filename', async (req, res) => {
    try {
        const filename = req.params.filename;
        const { content } = req.body;
        const filepath = path.join(DATA_DIR, filename);
        
        await fs.writeFile(filepath, content);
        
        // Git commit
        const timestamp = new Date().toISOString();
        execSync(`git add "${filename}" && git commit -m "Update ${filename}: ${timestamp}"`, { cwd: DATA_DIR });
        
        // Get commit hash
        const hash = execSync('git rev-parse HEAD', { cwd: DATA_DIR }).toString().trim().substring(0, 7);
        
        res.json({ 
            success: true, 
            message: 'File saved and committed',
            commit: hash,
            timestamp 
        });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// Get file history
app.get('/api/history/:filename', async (req, res) => {
    try {
        const filename = req.params.filename;
        
        // Get git log for file
        const log = execSync(
            `git log --pretty=format:'{"hash":"%h","timestamp":"%ai","message":"%s"}' -n 20 -- "${filename}"`,
            { cwd: DATA_DIR }
        ).toString();
        
        const history = log.split('\n')
            .filter(line => line)
            .map(line => JSON.parse(line));
        
        res.json({ history });
    } catch (error) {
        res.json({ history: [] });
    }
});

// Restore version
app.post('/api/restore/:filename/:hash', async (req, res) => {
    try {
        const { filename, hash } = req.params;
        
        // Get file content at specific commit
        const content = execSync(
            `git show ${hash}:"${filename}"`,
            { cwd: DATA_DIR }
        ).toString();
        
        // Save as current version
        const filepath = path.join(DATA_DIR, filename);
        await fs.writeFile(filepath, content);
        
        // Commit the restore
        execSync(
            `git add "${filename}" && git commit -m "Restored to version ${hash}"`,
            { cwd: DATA_DIR }
        );
        
        res.json({ 
            success: true, 
            content,
            message: `Restored to version ${hash}` 
        });
    } catch (error) {
        res.status(500).json({ error: error.message });
    }
});

// List files
app.get('/api/files', async (req, res) => {
    try {
        const files = await fs.readdir(DATA_DIR);
        const validFiles = files.filter(f => 
            ['.json', '.yaml', '.yml', '.xml'].some(ext => f.endsWith(ext))
        );
        res.json({ files: validFiles });
    } catch {
        res.json({ files: [] });
    }
});

// Start server
ensureDataDir().then(() => {
    app.listen(PORT, '0.0.0.0', () => {
        console.log(`
╔══════════════════════════════════════════╗
║         Edit1 - Visual Data Editor        ║
║                                          ║
║  Server running on http://localhost:${PORT} ║
║                                          ║
║  Usage:                                  ║
║  edit1 file.json                        ║
║  edit1 file.yaml                        ║
║  edit1 file.xml                         ║
╚══════════════════════════════════════════╝
        `);
    });
});

// package.json
const packageJson = {
    "name": "edit1-visual-editor",
    "version": "1.0.0",
    "description": "Visual editor for JSON, YAML, and XML files",
    "main": "server.js",
    "scripts": {
        "start": "node server.js"
    },
    "dependencies": {
        "express": "^4.18.2",
        "body-parser": "^1.20.2",
        "cors": "^2.8.5"
    },
    "engines": {
        "node": ">=14.0.0"
    }
};

// Dockerfile
const dockerfile = `
FROM node:18-alpine

# Install git
RUN apk add --no-cache git

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy application files
COPY . .

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 3001

# Start command
CMD ["npm", "start"]
`;

// docker-compose.yml
const dockerCompose = `
version: '3.8'

services:
  edit1:
    build: .
    container_name: edit1-editor
    ports:
      - "3001:3001"
    volumes:
      - ./data:/app/data
      - ./public:/app/public
    environment:
      - NODE_ENV=production
    restart: unless-stopped
`;

// edit1.sh - Launch script
const launchScript = `#!/bin/bash

# Edit1 - Visual Data Editor Launch Script

FILE=$1

if [ -z "$FILE" ]; then
    echo "Usage: edit1 <filename>"
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
echo "Starting Edit1 editor..."
docker-compose up -d --build

# Wait for server
sleep 2

# Open in browser
if command -v xdg-open > /dev/null; then
    xdg-open "http://localhost:3001?file=$FILE"
elif command -v open > /dev/null; then
    open "http://localhost:3001?file=$FILE"
else
    echo "Editor running at: http://localhost:3001?file=$FILE"
fi

echo "Edit1 is running. Press Ctrl+C to stop."
echo "Your file is being edited: $FILE"

# Keep script running
trap 'docker-compose down; exit' INT
while true; do sleep 1; done
`;

module.exports = { packageJson, dockerfile, dockerCompose, launchScript };