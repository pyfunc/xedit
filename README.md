# xedit


### Porównanie rozwiązań:

| Cecha | Edit1 (Node.js) | Edit2 (Python) | Edit3 (Go) |
|-------|-----------------|----------------|------------|
| **Rozmiar obrazu Docker** | ~150MB | ~180MB | ~25MB |
| **Czas startu** | ~2s | ~3s | <1s |
| **Zużycie RAM** | ~50MB | ~80MB | ~15MB |
| **Edytor kodu** | CodeMirror | Monaco | Ace |
| **Wizualizacja JSON** | JSONEditor (pełna) | Podstawowa | Zaawansowana |
| **Wizualizacja YAML** | Drzewo | Preview | Preview |
| **Wizualizacja XML** | Drzewo | Preview | Kolorowanie |
| **Łatwość instalacji** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |

### 🏆 **REKOMENDACJA: Edit1 (Node.js)**

**Dlaczego Edit1 jest najlepszy:**

1. **Najprostsza instalacja** - tylko `npm install` i `npm start`
2. **Najlepszy edytor wizualny** - JSONEditor oferuje pełną edycję wizualną z tree/form/code view
3. **Najszybszy development** - hot reload, bogaty ekosystem npm
4. **Najprostsza konfiguracja** - Docker Compose gotowy do użycia

### Szybki start dla każdego rozwiązania:

#### **Edit1 (Node.js) - REKOMENDOWANE** 🟢

```bash
# Utwórz folder projektu
mkdir edit1 && cd edit1

# Utwórz package.json
cat > package.json << 'EOF'
{
  "name": "edit1-visual-editor",
  "version": "1.0.0",
  "scripts": {
    "start": "node server.js"
  },
  "dependencies": {
    "express": "^4.18.2",
    "body-parser": "^1.20.2",
    "cors": "^2.8.5"
  }
}
EOF

# Utwórz Dockerfile
cat > Dockerfile << 'EOF'
FROM node:18-alpine
RUN apk add --no-cache git
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN mkdir -p /app/data
EXPOSE 3001
CMD ["npm", "start"]
EOF

# Utwórz docker-compose.yml
cat > docker-compose.yml << 'EOF'
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
    restart: unless-stopped
EOF

# Utwórz skrypt uruchamiający
cat > edit1 << 'EOF'
#!/bin/bash
FILE=$1
if [ -z "$FILE" ]; then
    echo "Usage: edit1 <filename.json|yaml|xml>"
    exit 1
fi
docker-compose up -d --build
sleep 2
echo "Opening http://localhost:3001?file=$FILE"
open "http://localhost:3001?file=$FILE" 2>/dev/null || xdg-open "http://localhost:3001?file=$FILE" 2>/dev/null || echo "Open manually: http://localhost:3001?file=$FILE"
EOF

chmod +x edit1

# Skopiuj pliki server.js i public/index.html z artifactów powyżej

# Uruchom
./edit1 config.json
```

#### **Edit2 (Python)**

```bash
# Utwórz folder projektu  
mkdir edit2 && cd edit2

# Utwórz requirements.txt
cat > requirements.txt << 'EOF'
Flask==2.3.3
Flask-CORS==4.0.0
PyYAML==6.0.1
EOF

# Utwórz Dockerfile
cat > Dockerfile << 'EOF'
FROM python:3.11-slim
RUN apt-get update && apt-get install -y git && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
RUN mkdir -p /app/data
EXPOSE 3002
CMD ["python", "app.py"]
EOF

# Skopiuj app.py i templates/index.html z artifactów

# Uruchom
docker build -t edit2 .
docker run -d -p 3002:3002 -v $(pwd)/data:/app/data edit2
```

#### **Edit3 (Go)**

```bash
# Utwórz folder projektu
mkdir edit3 && cd edit3

# Utwórz go.mod
cat > go.mod << 'EOF'
module edit3
go 1.21
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/gin-contrib/cors v1.4.0
    gopkg.in/yaml.v3 v3.0.1
)
EOF

# Utwórz Dockerfile
cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o edit3 .

FROM alpine:latest
RUN apk --no-cache add ca-certificates git
WORKDIR /app
COPY --from=builder /app/edit3 .
COPY --from=builder /app/static ./static
RUN mkdir -p /app/data
EXPOSE 3003
CMD ["./edit3"]
EOF

# Skopiuj main.go i static/index.html z artifactów

# Uruchom
docker build -t edit3 .
docker run -d -p 3003:3003 -v $(pwd)/data:/app/data edit3
```

### Funkcjonalności wspólne dla wszystkich rozwiązań:

✅ **Edycja wizualna i tekstowa jednocześnie**
✅ **Kolorowanie składni**
✅ **Automatyczne zapisywanie do Git**
✅ **Historia wersji z możliwością przywracania**
✅ **Walidacja formatów przed zapisem**
✅ **Docker dla łatwego deploymentu**
✅ **Brak konieczności konfiguracji**

### Użycie:

```bash
# Dla każdego edytora:
edit1 config.yaml    # Node.js (port 3001)
edit2 data.json      # Python (port 3002)  
edit3 settings.xml   # Go (port 3003)

# Pliki są automatycznie:
# - Tworzone jeśli nie istnieją
# - Wersjonowane w Git
# - Zapisywane w folderze ./data
```

### Dodatkowe ulepszenia (opcjonalne):

1. **Instalacja globalna (dla Edit1)**:
```bash
npm install -g edit1-visual-editor
edit1 myfile.json
```

2. **Aliasy w ~/.bashrc**:
```bash
alias edit-json='edit1'
alias edit-yaml='edit1'
alias edit-xml='edit1'
```

3. **Integracja z VS Code**:
```json
{
  "terminal.integrated.commandsToSkipShell": ["edit1", "edit2", "edit3"]
}
```

**Wybierz Edit1 (Node.js)** jeśli chcesz najłatwiejsze rozwiązanie z najlepszym edytorem wizualnym! 🚀
