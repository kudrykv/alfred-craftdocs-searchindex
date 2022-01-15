package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/kudrykv/alfred-craftdocs-searchindex/app/types"
)

const (
	searchResultLimit = 40
)

type Space struct {
	ID string
	DB *sql.DB
}

type BlockRepo struct {
	spaces []Space
}

func NewBlockRepo(spaces ...Space) *BlockRepo {
	return &BlockRepo{spaces: spaces}
}

func (br *BlockRepo) Close() (err error) {
	for _, space := range br.spaces {
		err2 := space.DB.Close()
		if err == nil {
			err = err2
		}
	}
	return err
}

type Block struct {
	ID           string
	SpaceID      string
	Content      string
	DocumentID   string
	DocumentName string
}

func (b *BlockRepo) Search(ctx context.Context, terms []string) ([]Block, error) {
	parts := make([]string, 0, len(terms))
	termsIface := make([]interface{}, 0, len(terms))

	for i, term := range terms {
		parts = append(parts, "utf8lower(ifnull(content, '')) like ?"+strconv.Itoa(i+1))
		termsIface = append(termsIface, "%"+strings.ToLower(term)+"%")
	}

	blocks := make([]Block, 0, 40)
	for _, space := range b.spaces {
		limit := searchResultLimit - len(blocks)
		if limit == 0 {
			break
		}
		log.Printf("Searching %s, limit %d", space.ID, limit)

		query := fmt.Sprintf("select id, content, documentId from BlockSearch where %s limit %d", strings.Join(parts, " and "), limit)
		rows, err := space.DB.QueryContext(ctx, query, termsIface...)
		if err != nil {
			return nil, types.NewError("failed to query database", err)
		}

		for rows.Next() {
			block := Block{SpaceID: space.ID}

			if err = rows.Scan(&block.ID, &block.Content, &block.DocumentID); err != nil {
				return nil, types.NewError("failed to scan a row", err)
			}

			blocks = append(blocks, block)
		}

		if err = rows.Err(); err != nil {
			return nil, types.NewError("error in rows", err)
		}

		if err = rows.Close(); err != nil {
			return nil, types.NewError("closing rows failed", err)
		}
	}

	return blocks, nil
}

type docKey struct {
	spaceID string
	docID   string
}

func (b *BlockRepo) BackfillDocumentNames(ctx context.Context, blocks []Block) ([]Block, error) {
	if len(blocks) == 0 {
		return blocks, nil
	}

	blocksBySpace := make(map[string][]Block)
	for _, block := range blocks {
		blocksBySpace[block.SpaceID] = append(blocksBySpace[block.SpaceID], block)
	}

	docIDs := make(map[docKey]string)

	for _, space := range b.spaces {
		b := blocksBySpace[space.ID]

		ids := make([]interface{}, 0, len(b))
		for _, k := range b {
			ids = append(ids, k.DocumentID)
		}

		placeholders := make([]string, 0, len(ids))
		for i := range ids {
			placeholders = append(placeholders, "?"+strconv.Itoa(i+1))
		}

		query := `select documentId, content from BlockSearch where entityType = 'document' and documentId in (` + strings.Join(placeholders, ", ") + ")"
		rows, err := space.DB.QueryContext(ctx, query, ids...)
		if err != nil {
			return nil, types.NewError("failed to query the database", err)
		}

		for rows.Next() {
			var block Block

			if err = rows.Scan(&block.DocumentID, &block.Content); err != nil {
				return nil, types.NewError("failed to scan row", err)
			}

			docIDs[docKey{spaceID: space.ID, docID: block.DocumentID}] = block.Content
		}

		if err = rows.Err(); err != nil {
			return nil, types.NewError("error in rows", err)
		}

		if err = rows.Close(); err != nil {
			return nil, types.NewError("closing rows failed", err)
		}
	}

	// Avoid mutating data in original slice.
	backfilled := make([]Block, len(blocks))
	copy(backfilled, blocks)

	for i, block := range backfilled {
		backfilled[i].DocumentName = docIDs[docKey{spaceID: block.SpaceID, docID: block.DocumentID}]
	}

	return backfilled, nil
}
