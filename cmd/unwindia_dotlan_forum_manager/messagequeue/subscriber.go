package messagequeue

import (
	"context"
	"fmt"
	"github.com/GSH-LAN/Unwindia_common/src/go/matchservice"
	"github.com/GSH-LAN/Unwindia_common/src/go/messagebroker"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/environment"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/apache/pulsar-client-go/pulsar"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

const (
	SubscriberName = "UNWINDIA_DOTLAN_FORUM_MANAGER"
	topicBase      = "persistent://public/default/%s"
)

type Subscriber struct {
	mainContext    context.Context
	pulsarClient   pulsar.Client
	pulsarConsumer pulsar.Consumer
	topic          string
	matchInfoChan  chan<- *matchservice.MatchInfo
}

func NewSubscriber(ctx context.Context, env *environment.Environment, matchInfoChan chan *matchservice.MatchInfo) (*Subscriber, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:            env.PulsarURL,
		Authentication: env.PulsarAuth,
	})
	if err != nil {
		return nil, err
	}

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:            fmt.Sprintf(topicBase, messagebroker.TOPIC),
		SubscriptionName: SubscriberName,
		Type:             pulsar.Shared,
	})

	if err != nil {
		return nil, err
	}

	subscriber := Subscriber{
		mainContext:    ctx,
		topic:          messagebroker.TOPIC,
		pulsarClient:   client,
		pulsarConsumer: consumer,
		matchInfoChan:  matchInfoChan,
	}

	return &subscriber, nil
}

func (s *Subscriber) processMessages(messages <-chan *message.Message) {
	for msg := range messages {
		if s.mainContext.Err() != nil {
			return
		}
		msgContent := messagebroker.Message{}

		err := jsoniter.Unmarshal(msg.Payload, &msgContent)
		if err != nil {
			log.Info().Str("topic", s.topic).Interface("payload", string(msg.Payload)).Msg("Received message but error on unmarshal")
			log.Error().Err(err).Msg("Error unmarshalling message")
			continue
		}
		log.Info().Str("topic", s.topic).Interface("message", msgContent).Msgf("Received message: %+v", msgContent)

		//switch msgContent.SubType {
		//case messagebroker.UNWINDIA_MATCH_NEW.String(),
		//	messagebroker.UNWINDIA_MATCH_NEW.String(),
		//	messagebroker.UNWINDIA_MATCH_READY_A.String(),
		//	messagebroker.UNWINDIA_MATCH_READY_B.String(),
		//	messagebroker.UNWINDIA_MATCH_READY_ALL.String(),
		//	messagebroker.UNWINDIA_MATCH_FINISHED.String():

		match := matchservice.MatchInfo{}
		err = mapstructure.WeakDecode(msgContent.Data, &match)
		if err != nil {
			log.Error().Err(err).Msg("Error decoding match")
		}

		log.Info().Str("topic", s.topic).Interface("match", match).Msg("Received match")

		s.matchInfoChan <- &match
		//}
	}
}

func (s *Subscriber) StartConsumer() {
	messageChan := make(chan *message.Message)

	go func() {
		defer s.pulsarConsumer.Close()

		for s.mainContext.Err() == nil {
			msg, err := s.pulsarConsumer.Receive(s.mainContext)
			if err != nil {
				log.Error().Err(err).Msg("Error receiving message")
				continue
			} else {
				response := make(map[string]interface{})
				err = jsoniter.Unmarshal(msg.Payload(), &response)
				if err != nil {
					s.pulsarConsumer.Nack(msg)
				}
				log.Info().Msgf("[%s] Received message : %v", s.topic, response)
				messageChan <- &message.Message{
					UUID:    msg.Key(),
					Payload: msg.Payload(),
				}
			}

			err = s.pulsarConsumer.Ack(msg)
			if err != nil {
				log.Error().Err(err).Msg("Error acking message")
			}
		}
	}()

	go s.processMessages(messageChan)

	log.Info().Str("topic", s.topic).Msg("Started pulsar subscriber")
}
