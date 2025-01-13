package main

import (
	"context"
	"fmt"
	"forwarder/extractor"
	"forwarder/logger"
	"forwarder/sink"
	"forwarder/source"
	"go.uber.org/zap"
	"os"
	"path/filepath"

	"forwarder/config"
	"forwarder/transform"
)

func main() {
	serviceCfg, err := config.LoadConfig(filepath.Join(os.Getenv("CONFIG_PATH"), "app.yml"))
	if err != nil {
		fmt.Printf("config.LoadConfig fail to load config %v", err)
		os.Exit(1)
	}

	log, err := logger.InitLogger(serviceCfg.Logger.Mode)
	if err != nil {
		fmt.Printf("logger.InitLogger failed to init logger %v", err)
		os.Exit(1)
	}

	// Create Input using Factory
	s, err := source.Create(serviceCfg.Source.Type, serviceCfg, log)
	if err != nil {
		log.Error("source.Create failed to create input source", zap.String("topic", serviceCfg.Source.Type), zap.Error(err))
		return
	}
	messages := s.Consume(context.Background())

	// Initialize Transformer and Output
	transformer := transform.NewMarkdownTransform()
	producer := sink.NewDifyProducer(serviceCfg.Sink)

	// Process Messages
	log.Info("Processing messages...")
	for message := range messages {

		title, extractedMessages, err := extractor.ExtractHTML(message.Type, message.Content)
		if err != nil {
			log.Error("extractor.ExtractHTML Failed to extract message", zap.Error(err))
			continue
		}
		log.Debug("Successfully extracted message", zap.Any("messages", extractedMessages))
		transformedMessage, err := transformer.Transform(extractedMessages)
		if err != nil {
			log.Error("transformer.Transform Failed to transform message", zap.Error(err))
			continue
		}

		if err := producer.Produce(title, transformedMessage); err != nil {
			log.Error("producer.Produce Failed to send message", zap.Error(err))
		} else {
			log.Debug("Successfully sent message")
		}
	}
}
