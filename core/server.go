package core

import (
	aiService "apipig/app/ai/service"
	reviewService "apipig/app/apps/code-review/service"
	"apipig/global"
	"apipig/initialize"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Server interface {
	ServeAsync(string, *fiber.App) error
}

func RunServer() {
	// init routers
	app := initialize.Routers()
	address := fmt.Sprintf(":%d", global.CONFIG.System.Port)

	// 启动定时任务
	go RunJobs()

	// kill daemon exit
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-quit
		fmt.Println("Shutdown Server ...")
		if err := app.Shutdown(); err != nil {
			fmt.Println(err)
			log.Fatalf("Server Shutdown: %s", err)
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := reviewService.ShutdownReview(shutdownCtx); err != nil {
			fmt.Println("Code review service shutdown:", err)
		}
		if err := aiService.ShutdownAI(shutdownCtx); err != nil {
			fmt.Println("AI service shutdown:", err)
		}
		cancel()
		fmt.Println("Server exit")
	}()

	// start app
	time.Sleep(10 * time.Microsecond)
	err := app.Listen(address)
	if err != nil {
		panic(err)
	}
}
