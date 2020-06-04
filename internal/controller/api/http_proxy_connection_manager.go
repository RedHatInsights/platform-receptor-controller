package api

import (
	"fmt"
	//"os"
	"strings"

	"github.com/RedHatInsights/platform-receptor-controller/internal/config"
	"github.com/RedHatInsights/platform-receptor-controller/internal/controller"
	"github.com/RedHatInsights/platform-receptor-controller/internal/platform/logger"
)

type RedisConnectionLocator struct {
	Locator controller.RedisLocator
	Cfg     *config.Config
}

func (rcl *RedisConnectionLocator) GetConnection(account string, node_id string) controller.Receptor {
	var conn controller.Receptor

	/*
		var url string
		url = os.Getenv("GATEWAY_URL")
		if len(url) == 0 {
			logger.Log.Printf("GATEWAY_URL env var is not set\n")
		}
		logger.Log.Printf("GATEWAY_URL: %s\n", url)
	*/

	var podName string
	var err error

	if podName, err = rcl.Locator.GetConnection(account, node_id); err != nil {
		// FIXME: log error, return an error
		return nil
	}
	fmt.Println("get by account/nodeid result:", podName)
	fmt.Printf("get by account/nodeid result (type):%T\n", podName)
	fmt.Println("get by account/nodeid err:", err)

	if podName == "" {
		return nil
	}

	url := fmt.Sprintf("%s://%s:%s", rcl.Cfg.ReceptorProxyScheme, podName, rcl.Cfg.ReceptorProxyPort)

	conn = &ReceptorHttpProxy{
		Url:           url,
		AccountNumber: account,
		NodeID:        node_id,
		ClientID:      rcl.Cfg.JobReceiverClientID,
		PSK:           rcl.Cfg.JobReceiverPSK,
	}

	return conn
}

func (rcl *RedisConnectionLocator) GetConnectionsByAccount(account string) map[string]controller.Receptor {

	connectionsPerAccount := make(map[string]controller.Receptor)

	accountConnections, err := rcl.Locator.Lookup(account)
	if err != nil {
		// FIXME: Update connectionlocator interface methods to return error
		logger.Log.Warnf("Error during lookup for account: %s", account)
		return nil
	}

	for _, conn := range accountConnections {
		s := strings.Split(conn, ":")
		nodeID := s[0]
		proxy := rcl.GetConnection(account, nodeID)
		// FIXME: Is this a good key?
		connectionsPerAccount[account+":"+nodeID] = proxy
	}

	return connectionsPerAccount
}

func (rcl *RedisConnectionLocator) GetAllConnections() map[string]map[string]controller.Receptor {

	connectionMap := make(map[string]map[string]controller.Receptor)

	connections, err := rcl.Locator.Lookup("connections")
	if err != nil {
		// FIXME: Update connectionlocator interface methods to return error
		logger.Log.Warn("Error during lookup for all connections")
		return nil
	}

	for _, conn := range connections {
		s := strings.Split(conn, ":")
		account, nodeID := s[0], s[1]
		proxy := rcl.GetConnection(account, nodeID)
		if _, exists := connectionMap[account]; !exists {
			connectionMap[account] = make(map[string]controller.Receptor)
		}
		connectionMap[account][nodeID] = proxy
	}

	return connectionMap
}
