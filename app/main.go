package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	aw "github.com/deanishe/awgo"
	"github.com/mattn/go-sqlite3"
)

func main() {
	wf := aw.New()

	wf.Run(workflow(context.Background(), wf, os.Args[1:]))
}

//nolint:gochecknoinits
func init() {
	sql.Register("sqlite3_custom", &sqlite3.SQLiteDriver{
		ConnectHook: func(conn *sqlite3.SQLiteConn) error {
			err := conn.RegisterFunc("utf8lower", strings.ToLower, true)
			if err != nil {
				return fmt.Errorf("register utf8lower func: %w", err)
			}

			return nil
		},
	})

	if len(os.Getenv("alfred_workflow_bundleid")) == 0 {
		if err := os.Setenv("alfred_workflow_bundleid", "dev.kudrykv.craftsearchindex"); err != nil {
			panic(err)
		}
	}

	if len(os.Getenv("alfred_workflow_data")) == 0 {
		if err := os.Setenv("alfred_workflow_data", "./tmp/data"); err != nil {
			panic(err)
		}
	}

	if len(os.Getenv("alfred_workflow_cache")) == 0 {
		if err := os.Setenv("alfred_workflow_cache", "./tmp/cache"); err != nil {
			panic(err)
		}
	}
}
