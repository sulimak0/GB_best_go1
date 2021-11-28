package main

import (
	"context"
	"go.uber.org/zap"
	"lesson1/internal/crawler"
	"lesson1/internal/models"
	"lesson1/internal/requester"
	"lesson1/internal/service"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't init logger: %s", err)
	}
	defer func() {
		err := logger.Sync()
		if err != nil {
			log.Fatalf("can't sync logger: %s", err)
		}
	}()
	slog := logger.Sugar()
	slog.Infof("init logger")

	cfg := models.Config{
		MaxDepth:       1,
		MaxResults:     10000,
		MaxErrors:      10000,
		Url:            "https://golang.org",
		RequestTimeout: 5,
		GlobalTimeout:  30,
	}
	slog.Infow("read config", "config", cfg)
	slog.Infow("process id", "id", os.Getpid())

	r, err := requester.NewRequester(time.Duration(cfg.RequestTimeout)*time.Second, nil, slog)
	if err != nil {
		slog.Errorf("requester initialize error: %s", err)
		return
	}
	cr, err := crawler.NewCrawler(r, cfg.MaxDepth, slog)
	if err != nil {
		slog.Errorf("crawler initialize error: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.GlobalTimeout)*time.Second)

	srv, err := service.NewService(cfg, cr, cancel, ctx, slog)
	if err != nil {
		slog.Errorf("service initialize error: %s", err)
		return
	}

	srv.Run()

	sigCh := make(chan os.Signal, 1)                       //Создаем канал для приема сигналов
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGUSR1) //Подписываемся на сигнал SIGINT и SIGUSR1

	for {
		select {
		case <-ctx.Done():
			slog.Warnw("ctx done")
			return

		case s := <-sigCh:
			switch s {
			case syscall.SIGUSR1:
				slog.Infow("SIGURSR1")
				cr.AddDepth(2)
			case syscall.SIGTERM:
				slog.Infow("SIGTERM")
				cancel()
			}
		}
	}
}
