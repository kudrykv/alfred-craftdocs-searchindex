package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kudrykv/alfred-craftdocs-searchindex/app/repository"
)

type BlockService struct {
	br *repository.BlockRepo
}

func NewBlockService(br *repository.BlockRepo) *BlockService {
	return &BlockService{br: br}
}

var regexCleanSpaces = regexp.MustCompile(`\s+`)

func (r *BlockService) Search(ctx context.Context, args []string) ([]repository.Block, error) {
	args = strings.Split(regexCleanSpaces.ReplaceAllString(strings.Join(args, " "), " "), " ")

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
