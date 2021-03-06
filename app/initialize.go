package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/config"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/repository"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/service"
	"github.com/kudrykv/alfred-craftdocs-searchindex/app/types"
)

func initialize() (*config.Config, *service.BlockService, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("get config: %w", err)
	}

	var spaces []repository.Space
	for _, si := range cfg.SearchIndexes() {
		db, err := sql.Open("sqlite3_custom", si.Path())
		if err != nil {
			return nil, nil, fmt.Errorf("sql open: %w", err)
		}
		spaces = append(spaces, repository.Space{
			ID: si.SpaceID,
			DB: db,
		})
	}

	blockRepo := repository.NewBlockRepo(spaces...)
	blockService := service.NewBlockService(blockRepo)

	return cfg, blockService, nil
}

func flow(ctx context.Context, args []string) (*config.Config, []repository.Block, error) {
	cfg, blockService, err := initialize()
	if err != nil {
		return nil, nil, fmt.Errorf("initialize: %w", err)
	}

	defer func() { _ = blockService.Close() }()

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

		config, blocks, err := flow(ctx, args)
		if err != nil {
			var te types.Error
			if errors.As(err, &te) {
				wf.NewWarningItem(te.Title, err.Error())
			} else {
				wf.NewWarningItem("Unknown error", err.Error())
			}

			return
		}

		if len(blocks) == 0 {
			addCreateNewDocument(wf, config, args)
		}

		// Sort all documents (across spaces) on top, whilst maintaining
		// order, primary space documents will always be on top.
		sort.SliceStable(blocks, func(i, j int) bool {
			if blocks[i].IsDocument() && !blocks[j].IsDocument() {
				return true
			}
			if !blocks[i].IsDocument() && blocks[j].IsDocument() {
				return false
			}
			return i < j
		})

		newDocumentEntryAdded := false
		for _, block := range blocks {
			// Append new document after documents but before
			// individual blocks.
			if !newDocumentEntryAdded && !block.IsDocument() {
				addCreateNewDocument(wf, config, args)
				newDocumentEntryAdded = true
			}
			wf.
				NewItem(block.Content).
				Subtitle(block.DocumentName).
				UID(block.ID).
				Arg("craftdocs://open?blockId=" + block.ID + "&spaceId=" + block.SpaceID).
				Valid(true)
		}
	}
}

func addCreateNewDocument(wf *aw.Workflow, config *config.Config, args []string) {
	name := strings.Join(args, " ")
	title := fmt.Sprintf("Create %q", name)
	url := fmt.Sprintf("craftdocs://createdocument?spaceId=%s&title=%s&content=&folderId=", config.SearchIndexes()[0].SpaceID, url.PathEscape(name))
	wf.
		NewItem(title).
		UID(title).
		Arg(url).
		Valid(true)
}
