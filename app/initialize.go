package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	aw "github.com/deanishe/awgo"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/config"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/repository"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/service"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/types"
)

func initialize() (*sql.DB, *config.Config, *service.BlockService, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get config: %w", err)
	}

	var db *sql.DB
	if db, err = sql.Open("sqlite3_custom", cfg.PathToIndex()); err != nil {
		return nil, nil, nil, fmt.Errorf("sql open: %w", err)
	}

	blockRepo := repository.NewBlockRepo(db)
	blockService := service.NewBlockService(blockRepo)

	return db, cfg, blockService, nil
}

func flow(ctx context.Context, args []string) (*config.Config, []repository.Block, error) {
	db, cfg, blockService, err := initialize()
	if err != nil {
		return nil, nil, fmt.Errorf("initialize: %w", err)
	}

	defer func() { _ = db.Close() }()

	blocks, err := blockService.Search(ctx, args)
	if err != nil {
		return nil, nil, fmt.Errorf("search: %w", err)
	}

	return cfg, blocks, nil
}

func workflow(ctx context.Context, wf *aw.Workflow, args []string) func() {
	return func() {
		defer wf.SendFeedback()
		defer func() {
			if wf.IsEmpty() {
				wf.NewItem("No results")
			}
		}()

		cfg, blocks, err := flow(ctx, args)
		if err != nil {
			var te types.Error
			if errors.As(err, &te) {
				wf.NewWarningItem(te.Title, err.Error())
			} else {
				wf.NewWarningItem("Unknown error", err.Error())
			}

			return
		}

		for _, block := range blocks {
			wf.
				NewItem(block.Content).
				Subtitle(block.DocumentName).
				UID(block.ID).
				Arg("craftdocs://open?blockId=" + block.ID + "&spaceId=" + cfg.SpaceID).
				Valid(true)
		}
	}
}
