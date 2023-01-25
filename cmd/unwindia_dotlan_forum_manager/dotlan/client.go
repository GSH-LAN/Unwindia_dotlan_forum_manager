package dotlan

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/GSH-LAN/Unwindia_common/src/go/config"
	"github.com/GSH-LAN/Unwindia_common/src/go/matchservice"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/environment"
	sq "github.com/Masterminds/squirrel"
	"github.com/gammazero/workerpool"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

const (
	dotlanForumExt = "turnier"
)

type DotlanDbClient interface {
	UpsertForumPostForMatch(ctx context.Context, matchInfo *matchservice.MatchInfo, text string) (int, int, error)
	UpdateForumPostForMatch(ctx context.Context, postId int, text string) error
}

type DotlanDbClientImpl struct {
	ctx             context.Context
	db              *sqlx.DB
	env             *environment.Environment
	lock            sync.Mutex
	workerpool      *workerpool.WorkerPool
	modelFieldCache map[string]string
	config          config.ConfigClient
}

func (d *DotlanDbClientImpl) UpsertForumPostForMatch(ctx context.Context, matchInfo *matchservice.MatchInfo, text string) (threadId int, postId int, err error) {
	// begin transaction
	tx, err := d.db.Beginx()
	if err != nil {
		return
	}

	commit := false

	defer func() {
		if commit {
			err = tx.Commit()
		} else {
			err = tx.Rollback()
		}
	}()

	// check if we have an existing thread
	qry := "select threadid from forum_thread where ext_id = ? order by threadid LIMIT 1"
	log.Debug().Str("query", qry).Str("ext_id", matchInfo.MsID).Msg("prepared query for getting thread")

	stmt, err := tx.PreparexContext(ctx, qry)
	if err != nil {
		return 0, 0, err
	}

	row := stmt.QueryRowxContext(ctx, matchInfo.MsID)
	err = row.Scan(&threadId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error().Err(err).Msg("error scanning thread")
			return 0, 0, err
		}
		log.Debug().Err(err).Msg("we have no thread")

		// we have no thread, so we need to create one
		qry, args, err := sq.Insert(ForumThread{}.TableName()).
			Columns("title", "forumid", "lastposttime", "replies", "hits", "ext", "ext_id").
			Values(matchInfo.MatchTitle, d.env.DotlanContestForumThreadId, time.Now(), 1, 1, dotlanForumExt, matchInfo.MsID).
			ToSql()
		if err != nil {
			log.Error().Err(err).Msg("error creating new thread sql")
			return 0, 0, err
		}
		log.Debug().Str("query", qry).Interface("args", args).Msg("create new thread query")

		insertResult, err := tx.ExecContext(ctx, qry, args...)
		if err != nil {
			log.Error().Err(err).Msg("error creating new thread")
			return 0, 0, err
		}

		id, err := insertResult.LastInsertId()
		if err != nil {
			log.Error().Err(err).Msg("error getting last inserted threadid")
			return 0, 0, err
		}

		threadId = int(id)
		log.Info().Int("threadId", threadId).Msg("created new thread")
	} else {
		log.Info().Int("threadId", threadId).Msg("found existing thread")
	}

	log.Info().Int("threadId", threadId).Str("contestId", matchInfo.MsID).Msgf("Upserting post for match thread")

	qry = "select postid from forum_post where threadid = ? and userid = ? order by postid LIMIT 1"
	log.Debug().Str("query", qry).Int("threadid", threadId).Uint("userid", d.config.GetConfig().CmsConfig.UserId).Msg("prepared query for getting post")

	stmt, err = tx.PreparexContext(ctx, qry)
	if err != nil {
		return 0, 0, err
	}

	row = stmt.QueryRowxContext(ctx, threadId, d.config.GetConfig().CmsConfig.UserId)
	err = row.Scan(&postId)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Error().Err(err).Msg("error scanning post")
			return 0, 0, err
		}
		log.Debug().Err(err).Msg("we have no post")

		qry, args, err := sq.Insert(ForumPost{}.TableName()).
			Columns("threadid", "userid", "dateline", "htmltext").
			Values(threadId, d.config.GetConfig().CmsConfig.UserId, time.Now(), text).
			ToSql()
		if err != nil {
			log.Error().Err(err).Msg("error creating new post sql")
			return 0, 0, err
		}
		log.Debug().Str("query", qry).Interface("args", args).Msg("create new post query")

		_, err = tx.ExecContext(ctx, qry, args...)
		if err != nil {
			log.Error().Err(err).Msg("error creating new post")
			return 0, 0, err
		}

		qry = "update t_contest set comments = comments+1 where tcid = ?"
		_, err = tx.ExecContext(ctx, qry, matchInfo.MsID)
		if err != nil {
			log.Error().Err(err).Msg("error updating t_contest")
			return 0, 0, err
		}
	} else {
		//we have an existing post
		log.Debug().Int("postid", postId).Msg("found existing post which will be updated")

		qry = "update forum_post set htmltext = ? where postid = ?"
		_, err = tx.ExecContext(ctx, qry, text, postId)
		if err != nil {
			log.Error().Err(err).Msg("error updating t_contest")
			return 0, 0, err
		}
	}

	return
}

func (d *DotlanDbClientImpl) UpdateForumPostForMatch(ctx context.Context, postId int, text string) error {
	// begin transaction
	tx, err := d.db.Beginx()
	if err != nil {
		return err
	}

	commit := false

	defer func() {
		if commit {
			err = tx.Commit()
		} else {
			err = tx.Rollback()
		}
	}()

	qry := "update forum_post set htmltext = ? where postid = ?"
	_, err = tx.ExecContext(ctx, qry, text, postId)
	if err != nil {
		return err
	}
	return nil
}

func NewClient(ctx context.Context, env *environment.Environment, wp *workerpool.WorkerPool, config config.ConfigClient) (DotlanDbClient, error) {
	sqlxDsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		env.DotlanMySQLUser,
		env.DotlanMySQLPassword,
		env.DotlanMySQLHost,
		env.DotlanMySQLPort,
		env.DotlanMySQLDatabase)

	db, err := sqlx.Connect("mysql", sqlxDsn)

	if err != nil {
		return nil, err
	}

	return &DotlanDbClientImpl{
		ctx:             ctx,
		db:              db,
		env:             env,
		workerpool:      wp,
		config:          config,
		modelFieldCache: make(map[string]string),
	}, nil
}
