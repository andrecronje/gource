package commands

import (
	"fmt"
	"github.com/andrecronje/lachesis/src/lachesis"
	aproxy "github.com/andrecronje/lachesis/src/proxy/app"
	"github.com/andrecronje/lachesis/tester"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//NewRunCmd returns the command that starts a Lachesis node
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run node",
		PreRunE: loadConfig,
		RunE:    runLachesis,
	}
	AddRunFlags(cmd)
	return cmd
}

func runLachesis(cmd *cobra.Command, args []string) error {
	if !config.Inapp {
		p, err := aproxy.NewSocketAppProxy(
			config.ClientAddr,
			config.ProxyAddr,
			config.Lachesis.NodeConfig.HeartbeatTimeout,
			config.Lachesis.Logger,
		)

		if err != nil {
			config.Lachesis.Logger.Error("Cannot initialize socket AppProxy:", err)
			return nil
		}

		config.Lachesis.Proxy = p
	}

	engine := lachesis.NewLachesis(&config.Lachesis)

	if err := engine.Init(); err != nil {
		config.Lachesis.Logger.Error("Cannot initialize engine:", err)
		return nil
	}

	if config.Lachesis.Test {
		p, err := engine.Store.Participants()
		if err != nil {
			return cli.NewExitError(
				fmt.Sprintf("Failed to acquire participants: %s", err),
				1)
		}
		go tester.PingNodesN(p.Sorted, p.ByPubKey, config.Lachesis.TestN, config.Lachesis.ServiceAddr)
	}

	engine.Run()

	return nil
}


//AddRunFlags adds flags to the Run command
func AddRunFlags(cmd *cobra.Command) {

	cmd.Flags().String("datadir", config.Lachesis.DataDir, "Top-level directory for configuration and data")
	cmd.Flags().String("log", config.Lachesis.LogLevel, "debug, info, warn, error, fatal, panic")

	// Network
	cmd.Flags().StringP("listen", "l", config.Lachesis.BindAddr, "Listen IP:Port for lachesis node")
	cmd.Flags().DurationP("timeout", "t", config.Lachesis.NodeConfig.TCPTimeout, "TCP Timeout")
	cmd.Flags().Int("max-pool", config.Lachesis.MaxPool, "Connection pool size max")

	// Proxy
	cmd.Flags().Bool("inapp", config.Inapp, "Use an in-app proxy")
	cmd.Flags().StringP("proxy-listen", "p", config.ProxyAddr, "Listen IP:Port for lachesis proxy")
	cmd.Flags().StringP("client-connect", "c", config.ClientAddr, "IP:Port to connect to client")

	// Service
	cmd.Flags().StringP("service-listen", "s", config.Lachesis.ServiceAddr, "Listen IP:Port for HTTP service")

	// Store
	cmd.Flags().Bool("store", config.Lachesis.Store, "Use badgerDB instead of in-mem DB")
	cmd.Flags().Int("cache-size", config.Lachesis.NodeConfig.CacheSize, "Number of items in LRU caches")

	// Node configuration
	cmd.Flags().Duration("heartbeat", config.Lachesis.NodeConfig.HeartbeatTimeout, "Time between gossips")
	cmd.Flags().Int("sync-limit", config.Lachesis.NodeConfig.SyncLimit, "Max number of events for sync")

	// Test
	cmd.Flags().Bool("test", config.Lachesis.Test, "Enable testing (sends transactions to random nodes in the network)")
	cmd.Flags().Uint64("test_n", config.Lachesis.TestN, "Number of transactions to send")
}

func loadConfig(cmd *cobra.Command, args []string) error {

	err := bindFlagsLoadViper(cmd)
	if err != nil {
		return err
	}
 	config, err = parseConfig()
	if err != nil {
		return err
	}

	config.Lachesis.Logger.Level = lachesis.LogLevel(config.Lachesis.LogLevel)
	config.Lachesis.NodeConfig.Logger = config.Lachesis.Logger

	config.Lachesis.Logger.WithFields(logrus.Fields{
		"proxy-listen":   config.ProxyAddr,
		"client-connect": config.ClientAddr,
		"inapp":          config.Inapp,

		"lachesis.datadir":        config.Lachesis.DataDir,
		"lachesis.bindaddr":       config.Lachesis.BindAddr,
		"lachesis.service-listen": config.Lachesis.ServiceAddr,
		"lachesis.maxpool":        config.Lachesis.MaxPool,
		"lachesis.store":          config.Lachesis.Store,
		"lachesis.loadpeers":      config.Lachesis.LoadPeers,
		"lachesis.log":            config.Lachesis.LogLevel,

		"lachesis.node.heartbeat":  config.Lachesis.NodeConfig.HeartbeatTimeout,
		"lachesis.node.tcptimeout": config.Lachesis.NodeConfig.TCPTimeout,
		"lachesis.node.cachesize":  config.Lachesis.NodeConfig.CacheSize,
		"lachesis.node.synclimit":  config.Lachesis.NodeConfig.SyncLimit,
	}).Debug("RUN")

	return nil
}

//Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command) error {
	// cmd.Flags() includes flags from this command and all persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
 	viper.SetConfigName("lachesis")              // name of config file (without extension)
	viper.AddConfigPath(config.Lachesis.DataDir) // search root directory
	// viper.AddConfigPath(filepath.Join(config.Lachesis.DataDir, "lachesis")) // search root directory /config
 	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		config.Lachesis.Logger.Debugf("Using config file: ", viper.ConfigFileUsed())
	} else if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		config.Lachesis.Logger.Debugf("No config file found in:", config.Lachesis.DataDir)
	} else {
		return err
	}
 	return nil
}
 //Retrieve the default environment configuration.
func parseConfig() (*CLIConfig, error) {
	conf := NewDefaultCLIConfig()
	err := viper.Unmarshal(conf)
	if err != nil {
		return nil, err
	}
	return conf, err
}
 func logLevel(l string) logrus.Level {
	switch l {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	case "panic":
		return logrus.PanicLevel
	default:
		return logrus.DebugLevel
	}
}
