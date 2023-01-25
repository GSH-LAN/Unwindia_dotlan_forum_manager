package server

import (
	"context"
	"fmt"
	"github.com/GSH-LAN/Unwindia_common/src/go/config"
	"github.com/GSH-LAN/Unwindia_common/src/go/matchservice"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/database"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/dotlan"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/environment"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/messagequeue"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/template"
	"github.com/gammazero/workerpool"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"sync"
	"time"
)

type Server struct {
	env           *environment.Environment
	config        config.ConfigClient
	workerpool    *workerpool.WorkerPool
	subscriber    *messagequeue.Subscriber
	matchInfoChan chan *matchservice.MatchInfo
	lock          sync.Mutex
	dotlanClient  dotlan.DotlanDbClient
	dbClient      database.DatabaseClient
	stop          chan struct{}
}

func NewServer(ctx context.Context, env *environment.Environment, cfgClient config.ConfigClient, wp *workerpool.WorkerPool) (*Server, error) {
	matchInfoChan := make(chan *matchservice.MatchInfo)

	subscriber, err := messagequeue.NewSubscriber(ctx, env, matchInfoChan)
	if err != nil {
		return nil, err
	}

	dotlanClient, err := dotlan.NewClient(ctx, env, wp, cfgClient)
	if err != nil {
		return nil, err
	}

	dbClient, err := database.NewClient(ctx, env)

	srv := Server{
		env:           env,
		config:        cfgClient,
		workerpool:    wp,
		subscriber:    subscriber,
		matchInfoChan: matchInfoChan,
		lock:          sync.Mutex{},
		dotlanClient:  dotlanClient,
		dbClient:      dbClient,
		stop:          make(chan struct{}),
	}

	return &srv, nil
}

func (s *Server) Start() error {
	s.subscriber.StartConsumer()
	for {
		select {
		case <-s.stop:
			log.Info().Msg("Stopping processing, server stopped")
			return nil
		case matchInfo := <-s.matchInfoChan:
			s.workerpool.Submit(func() {
				s.matchInfoHandler(matchInfo)
			})
		}
	}
}

func (s *Server) Stop() error {
	log.Info().Msgf("Stopping server")
	close(s.stop)
	return fmt.Errorf("server Stopped")
}

func (s *Server) matchInfoHandler(matchInfo *matchservice.MatchInfo) {
	log := log.With().Str("matchId", matchInfo.MsID).Logger()

	log.Debug().Interface("matchInfo", matchInfo).Msg("Received match info")

	s.lock.Lock()
	defer s.lock.Unlock()

	commentText, err := template.ParseTemplateForMatch(s.config.GetConfig().Templates["CMS_FORUM_POST.gohtml"], matchInfo)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing template")
	}
	log.Debug().Str("commentText", commentText).Msg("parsed Template")

	dotlanForumState, err := s.dbClient.Get(context.TODO(), matchInfo.MsID)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error().Err(err).Msg("Failed to get dotlan forum state")
	}

	dotlanContext, cancel := context.WithTimeout(context.TODO(), time.Second*30)
	defer cancel()

	if dotlanForumState == nil {

		threadId, postId, err := s.dotlanClient.UpsertForumPostForMatch(dotlanContext, matchInfo, commentText)
		if err != nil {
			log.Error().Err(err).Msg("Error upserting forum post for match")
			return
		}

		dotlanForumState = &database.DotlanForumStatus{
			ID:                  matchInfo.MsID,
			DotlanForumPostID:   postId,
			DotlanForumThreadID: threadId,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Time{},
		}

		err = s.dbClient.Upsert(context.TODO(), dotlanForumState)
		if err != nil {
			log.Error().Err(err).Msg("Error upserting dotlanForumState")
			return
		}
	} else {
		log.Debug().Interface("dotlanForumState", dotlanForumState).Msg("Found dotlan forum state")

		s.dotlanClient.UpdateForumPostForMatch(dotlanContext, dotlanForumState.DotlanForumPostID, commentText)

		dotlanForumState.UpdatedAt = time.Now()

		err = s.dbClient.Upsert(context.TODO(), dotlanForumState)
		if err != nil {
			log.Error().Err(err).Msg("Error upserting dotlanForumState")
			return
		}
	}
}
