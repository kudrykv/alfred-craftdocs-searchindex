package service

import (
	"context"
	"fmt"

	"github.com/kudrykv/alfred-craftdocs-searchindex/app/repository"
)

type BlockService struct {
	br *repository.BlockRepo
}

func (bs *BlockService) Close() error {
	return bs.br.Close()
}

func NewBlockService(br *repository.BlockRepo) *BlockService {
	return &BlockService{br: br}
}

func (r *BlockService) Search(ctx context.Context, args []string) ([]repository.Block, error) {
	blocks, err := r.br.Search(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	blocks, err = r.br.BackfillDocumentNames(ctx, blocks)
	if err != nil {
		return nil, fmt.Errorf("backfill document names: %w", err)
	}

	return blocks, nil
}
