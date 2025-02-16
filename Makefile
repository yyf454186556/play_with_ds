# 定义默认任务
all:
	@echo "Starting Golang backend and frontend in parallel..."
	$(MAKE) backend &
	$(MAKE) frontend &
	wait

# 启动 Golang 后端
backend:
	@echo "Starting Golang backend..."
	go run .

# 安装前端依赖并启动前端开发服务器
frontend:
	@echo "Installing frontend dependencies..."
	cd play_with_ds_front && npm install
	@echo "Starting frontend development server..."
	cd frontend && npm run dev