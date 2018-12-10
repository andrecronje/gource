package node

import (
	"testing"
	"time"

	"github.com/Fantom-foundation/go-lachesis/src/common"
	"github.com/Fantom-foundation/go-lachesis/src/log"
	"github.com/sirupsen/logrus"
)

type Config struct {
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat"`
	TCPTimeout       time.Duration `mapstructure:"timeout"`
	CacheSize        int           `mapstructure:"cache-size"`
	SyncLimit        int64         `mapstructure:"sync-limit"`
	TestDelay        uint64        `mapstructure:"test_delay"`
	PeersCount       int           `mapstructure:"peers-count"`
	Logger           *logrus.Logger
}

func NewConfig(heartbeat time.Duration,
	timeout time.Duration,
	cacheSize int,
	syncLimit int64,
	peersCount int,
	logger *logrus.Logger) *Config {

	return &Config{
		HeartbeatTimeout: heartbeat,
		TCPTimeout:       timeout,
		CacheSize:        cacheSize,
		SyncLimit:        syncLimit,
		PeersCount:       peersCount,
		Logger:           logger,
	}
}

func DefaultConfig() *Config {
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	lachesis_log.NewLocal(logger, logger.Level.String())

	return &Config{
		HeartbeatTimeout: 10 * time.Millisecond,
		TCPTimeout:       180 * 1000 * time.Millisecond,
		CacheSize:        500,
		SyncLimit:        100,
		PeersCount:       1,
		TestDelay:        1,
		Logger:           logger,
	}
}

func TestConfig(t *testing.T) *Config {
	config := DefaultConfig()
	config.HeartbeatTimeout = time.Second * 1

	config.Logger = common.NewTestLogger(t)

	return config
}
