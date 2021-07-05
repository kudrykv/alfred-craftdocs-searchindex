package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/kudrykv/alfred-craftdocs-searchindex/app/types"
)

type BlockRepo struct {
	db *sql.DB
}

func NewBlockRepo(db *sql.DB) *BlockRepo {
	return &BlockRepo{db: db}
}

type Block struct {
	ID           string
	Content      string
	DocumentID   string
	DocumentName string
}

func (b *BlockRepo) Search(ctx context.Context, terms []string) ([]Block, error) {
	parts := make([]string, 0, len(terms))
	termsIface := make([]interface{}, 0, len(terms))

	for i, term := range terms {
		parts = append(parts, "utf8lower(ifnull(content, '')) like ?"+strconv.Itoa(i+1))
		termsIface = append(termsIface, "%"+term+"%")
	}

	query := "select id, content, documentId from BlockSearch where " + strings.Join(parts, " and ") + " limit 40"
	rows, err := b.db.QueryContext(ctx, query, termsIface...)
	if err != nil {
		return nil, types.NewError("failed to query database", err)
	}

	defer func() { _ = rows.Close() }()

	blocks := make([]Block, 0, 40)

	for rows.Next() {
		var block Block

		if err = rows.Scan(&block.ID, &block.Content, &block.DocumentID); err != nil {
			return nil, types.NewError("failed to scan a row", err)
		}

		blocks = append(blocks, block)
	}

	if err = rows.Err(); err != nil {
		return nil, types.NewError("error in rows", err)
	}

	return blocks, nil
}

func (b *BlockRepo) BackfillDocumentNames(ctx context.Context, blocks []Block) ([]Block, error) {
	if len(blocks) == 0 {
		return blocks, nil
	}

	docIDs := make(map[string]string, 40)

	for _, block := range blocks {
		docIDs[block.DocumentID] = ""
	}

	ids := make([]interface{}, 0, len(docIDs))
	for k := range docIDs {
		ids = append(ids, k)
	}

	placeholders := make([]string, 0, len(ids))
	for i := range ids {
		placeholders = append(placeholders, "?"+strconv.Itoa(i+1))
	}

	query := `select documentId, content from BlockSearch where entityType = 'document' and documentId in (` + strings.Join(placeholders, ", ") + ")"
	rows, err := b.db.QueryContext(ctx, query, ids...)
	if err != nil {
		return nil, types.NewError("failed to query the database", err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var block Block

		if err = rows.Scan(&block.DocumentID, &block.Content); err != nil {
			return nil, types.NewError("failed to scan row", err)
		}

		docIDs[block.DocumentID] = block.Content
	}

	if err = rows.Err(); err != nil {
		return nil, types.NewError("error in rows", err)
	}

	for i, block := range blocks {
		blocks[i].DocumentName = docIDs[block.DocumentID]
	}

	return blocks, nil
}
