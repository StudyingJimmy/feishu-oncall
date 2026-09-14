# 常用命令（Windows 下建议在 Git Bash / WSL 里执行，或直接用 go 命令）

BIN_DIR := bin
LDFLAGS := -s -w

.PHONY: init-config tidy build run-server run-worker test lint fmt clean

# 生成本地配置：仓库只提交 configs/config.example.yaml，真实的 configs/config.yaml 不入库
init-config:
	@if [ -f configs/config.yaml ]; then \
		echo "configs/config.yaml 已存在，跳过（要重新生成先删掉它）"; \
	else \
		cp configs/config.example.yaml configs/config.yaml; \
		echo "已生成 configs/config.yaml：请按注释填飞书凭证等；本地数据库/对象存储凭据可留空（自动回退默认值）"; \
	fi

tidy:
	go mod tidy

build:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/server ./cmd/server
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/worker ./cmd/worker

run-server:
	go run ./cmd/server -config configs/config.yaml

run-worker:
	go run ./cmd/worker -config configs/config.yaml

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -l -w .

clean:
	rm -rf $(BIN_DIR)
