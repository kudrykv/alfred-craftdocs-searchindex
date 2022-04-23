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
	EntityType   string
	DocumentID   string
	DocumentName string
}

func (b *Block) IsDocument() bool {
	return b.EntityType == "document"
}

// buildMatchQuery creates a FTS5 match query that searches both content
// and exactMatchContent (only set for documents). Content is used for
// regular full-text search and including exactMatchContent allows for
// searches like "mylist" that will match a document named "My List".
// This works slightly better than in Craft since we're "globbing" the
// term so that it will also match "My List Of Things" (unlike Craft).
//
// Example output:
//
// 	{content exactMatchContent} : ("my" "todo") OR ("my" "todo"*) OR ("my"* "todo"*)
// 	{content exactMatchContent} : ("mytodo") OR ("mytodo"*)
//
func buildMatchQuery(terms []string) string {
	if len(terms) == 0 {
		return ""
	}

	var quotedTerms []string
	for _, term := range terms {
		// Quote the term to avoid FTS5 bareword input sanitization.
		// https://www.sqlite.org/fts5.html#fts5_strings
		term = strings.ReplaceAll(term, "\"", " ")
		quotedTerms = append(quotedTerms, fmt.Sprintf("%q", term))
	}
	// Create different permutations of the match phrase (with and
	// without "globbing") in an attempt to give more weight (rank)
	// to non-"globbed" terms. The last form where every term is
	// followed by * will return all results, however, including the
	// former increases the weight of more exact results and more
	// closely matches the search results produced by Craft.
	//
	// Further permutations can include + between terms which
	// matches terms immediately following each other, but this does
	// not seem to affect the weights too much.
	//
	// In many cases, this matches the results returned by Craft but
	// often something can be slightly out of order. Without knowing
	// the exact sort criteria used in Craft we'll have to settle
	// for "good enough". For example, they could be using create or
	// modification date or frequency of access which we do not have
	// access to.
	matchPhrases := []string{
		strings.Join(quotedTerms, " "),       // '"term1" "term2"'
		strings.Join(quotedTerms, " ") + "*", // '"term1" "term2"*'
	}
	// Avoid unnecessarily repeating the result produced previously.
	if len(quotedTerms) > 1 {
		matchPhrases = append(matchPhrases, strings.Join(quotedTerms, "* ")+"*") // '"term1"* "term2"*'
	}

	matchQuery := fmt.Sprintf("{content exactMatchContent} : (%s)", strings.Join(matchPhrases, ") OR ("))

	return matchQuery
}

func (b *BlockRepo) Search(ctx context.Context, terms []string) ([]Block, error) {
	matchQuery := buildMatchQuery(terms)
	log.Printf("Searching with matchQuery: '%s'", matchQuery)

	blocks := make([]Block, 0, 40)
	for _, space := range b.spaces {
		limit := searchResultLimit - len(blocks)
		if limit == 0 {
			break
		}
		log.Printf("Searching %s, limit %d", space.ID, limit)

		var rows *sql.Rows
		var err error
		if len(matchQuery) > 0 {
			query := `
				SELECT id, content, entityType, documentId
				FROM BlockSearch(?)
				ORDER BY rank + customRank
				LIMIT ?
			`
			rows, err = space.DB.QueryContext(ctx, query, matchQuery, limit)
		} else {
			// No search terms were provided, fallback to listing
			// all results according to whatever custom rank Craft
			// has assigned them. Documents have a much lower
			// customRank than blocks, so with 40+ documents all
			// results will be documents. The results seems to be
			// sorted in an order approximating recently accessed,
			// edited or created.
			query := "SELECT id, content, entityType, documentId FROM BlockSearch ORDER BY customRank LIMIT ?"
			rows, err = space.DB.QueryContext(ctx, query, limit)
		}
		if err != nil {
			return nil, types.NewError("failed to query database", err)
		}

		for rows.Next() {
			block := Block{SpaceID: space.ID}

			if err = rows.Scan(&block.ID, &block.Content, &block.EntityType, &block.DocumentID); err != nil {
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
		placeholders := make([]string, 0, len(ids))
		for _, k := range b {
			if k.IsDocument() {
				// This is a document, no need to fetch title.
				continue
			}
			ids = append(ids, k.DocumentID)
			placeholders = append(placeholders, "?"+strconv.Itoa(len(ids)))
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
		if block.IsDocument() {
			backfilled[i].DocumentName = "[Document]"
		} else {
			backfilled[i].DocumentName = "[Block] " + docIDs[docKey{spaceID: block.SpaceID, docID: block.DocumentID}]
		}
	}

	return backfilled, nil
}
