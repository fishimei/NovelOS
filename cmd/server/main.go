// NovelOS 是一个 AI 辅助小说创作平台的后端服务。
// 采用分层架构设计：入口层 → 引导层 → 应用层 → 基础设施层 → 传输层。
// 核心功能分为两个流程：
// 1. Setup Flow（设置流程）：将作者的粗略想法转化为结构化的项目状态（世界观、角色、关系等）
// 2. Story Flow（故事流程）：通过 AI 辅助生成故事内容，支持实时进度推送和状态更新
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fishimei/NovelOS/internal/bootstrap"
	"github.com/fishimei/NovelOS/internal/config"
)

// main 是应用程序的入口函数。
// 职责范围仅限于进程级关注点：
// 1. 加载配置：从环境变量 NOVEL_OS_CONFIG 读取配置文件路径，使用 config.Load 解析
// 2. 构建应用：调用 bootstrap.New 创建 App 实例（包含 HTTP 服务器和路由设置）
// 3. 设置信号监听：捕获 Interrupt 和 Terminate 信号，用于优雅关闭
// 4. 启动应用：调用 app.Run 启动 HTTP 服务器并阻塞
// 5. 错误处理：如果启动或运行出错，记录日志并退出
// 注意：所有依赖注入和业务逻辑布线都留在 bootstrap 包中。
func main() {
	cfg, err := config.Load(os.Getenv("NOVEL_OS_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := bootstrap.New(cfg)
	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
