// Package main provides the entry point for the GPT-Load proxy server
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gpt-load/internal/app"
	"gpt-load/internal/container"
	"gpt-load/internal/types"
	"gpt-load/internal/utils"
)

//go:embed web/dist
var buildFS embed.FS

//go:embed web/dist/index.html
var indexPage []byte

func main() {
	// 设置静默模式，禁用项目日志输出到控制台
	os.Setenv("SILENT_MODE", "true")

	// Build the dependency injection container
	container, err := container.BuildContainer()
	if err != nil {
		// 在静默模式下，只输出关键错误信息
		fmt.Fprintf(os.Stderr, "Failed to build container: %v\n", err)
		os.Exit(1)
	}

	// Provide UI assets to the container
	if err := container.Provide(func() embed.FS { return buildFS }); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to provide buildFS: %v\n", err)
		os.Exit(1)
	}
	if err := container.Provide(func() []byte { return indexPage }); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to provide indexPage: %v\n", err)
		os.Exit(1)
	}

	// Initialize global logger
	if err := container.Invoke(func(configManager types.ConfigManager) {
		utils.SetupLogger(configManager)
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	// Create and run the application
	if err := container.Invoke(func(application *app.App, configManager types.ConfigManager) {
		if err := application.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start application: %v\n", err)
			os.Exit(1)
		}

		// 显示启动成功信息
		serverConfig := configManager.GetEffectiveServerConfig()
		// 当host为0.0.0.0，显示为localhost
		if serverConfig.Host == "0.0.0.0" {
			serverConfig.Host = "localhost"
		}
		fmt.Printf("项目已正常启动在 http://%s:%d\n", serverConfig.Host, serverConfig.Port)
		fmt.Printf("关闭命令行，程序将会被关闭")

		// Wait for interrupt signal for graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		// Create a context with timeout for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(serverConfig.GracefulShutdownTimeout)*time.Second)
		defer cancel()

		// Perform graceful shutdown
		application.Stop(shutdownCtx)

	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run application: %v\n", err)
		os.Exit(1)
	}
}
