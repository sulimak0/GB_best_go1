package service

import (
	"context"
	"go.uber.org/zap"
	"lesson1/internal/domain"
	"lesson1/internal/models"
)

type Service struct {
	config  models.Config
	crawler domain.Crawler
	ctx     context.Context
	cancel  context.CancelFunc
	slog    *zap.SugaredLogger
}

func NewService(cfg models.Config, cr domain.Crawler, cancel context.CancelFunc, ctx context.Context, slog *zap.SugaredLogger) (*Service, error) {

	return &Service{config: cfg, crawler: cr, ctx: ctx, cancel: cancel, slog: slog}, nil
}

func (s *Service) Run() {
	go s.crawler.Scan(s.ctx, s.config.Url, 1)
	go processResult(s.ctx, s.cancel, s.crawler, s.config, s.slog)
}

func (s *Service) IncreaseDepth(depth uint64) {
	s.crawler.AddDepth(depth)
}

func processResult(ctx context.Context, cancel func(), cr domain.Crawler, cfg models.Config, slog *zap.SugaredLogger) {
	var maxResult, maxErrors = cfg.MaxResults, cfg.MaxErrors
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cr.ChanResult():
			if msg.Err != nil {
				maxErrors--
				slog.Warnf("crawler result return err: %s", msg.Err.Error())
				if maxErrors <= 0 {
					slog.Errorw("crawler max errors")
					cancel()
					return
				}
			} else {
				maxResult--
				slog.Infof("crawler result: [url: %s] Title: %s", msg.Url, msg.Title)
				if maxResult <= 0 {
					slog.Infow("crawler max results")
					cancel()
					return
				}
			}
		}
	}
}
