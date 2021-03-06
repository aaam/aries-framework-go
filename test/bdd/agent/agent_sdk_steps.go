/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package agent

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/DATA-DOG/godog"
	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/common/log"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/messaging/msghandler"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/decorator"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/transport/ws"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries"
	"github.com/hyperledger/aries-framework-go/pkg/framework/aries/defaults"
	"github.com/hyperledger/aries-framework-go/pkg/storage"
	"github.com/hyperledger/aries-framework-go/pkg/storage/leveldb"
	"github.com/hyperledger/aries-framework-go/pkg/vdri/httpbinding"
	"github.com/hyperledger/aries-framework-go/test/bdd/pkg/context"
)

const (
	dbPath = "./db"

	httpTransportProvider      = "http"
	webSocketTransportProvider = "websocket"
)

var logger = log.New("aries-framework/tests")

// SDKSteps contains steps for agent from client SDK
type SDKSteps struct {
	bddContext *context.BDDContext
}

// NewSDKSteps returns new agent from client SDK
func NewSDKSteps(ctx *context.BDDContext) *SDKSteps {
	return &SDKSteps{bddContext: ctx}
}

func (a *SDKSteps) createAgent(agentID, inboundHost, inboundPort, scheme string) error {
	opts := append([]aries.Option{}, aries.WithStoreProvider(a.getStoreProvider(agentID)))

	return a.create(agentID, inboundHost, inboundPort, scheme, opts...)
}

func (a *SDKSteps) createAgentWithRegistrar(agentID, inboundHost, inboundPort, scheme string) error {
	msgRegistrar := msghandler.NewRegistrar()
	a.bddContext.MessageRegistrar[agentID] = msgRegistrar

	opts := append([]aries.Option{}, aries.WithStoreProvider(a.getStoreProvider(agentID)),
		aries.WithMessageServiceProvider(msgRegistrar))

	return a.create(agentID, inboundHost, inboundPort, scheme, opts...)
}

// CreateAgentWithHTTPDIDResolver creates agent with HTTP DID resolver
func (a *SDKSteps) CreateAgentWithHTTPDIDResolver(
	agents, inboundHost, inboundPort, endpointURL, acceptDidMethod string) error {
	var opts []aries.Option

	for _, agentID := range strings.Split(agents, ",") {
		httpVDRI, err := httpbinding.New(a.bddContext.Args[endpointURL],
			httpbinding.WithAccept(func(method string) bool { return method == acceptDidMethod }))
		if err != nil {
			return fmt.Errorf("failed from httpbinding new ")
		}

		storeProv := a.getStoreProvider(agentID)

		opts = append(opts, aries.WithVDRI(httpVDRI), aries.WithStoreProvider(storeProv))

		if err := a.create(agentID, inboundHost, inboundPort, "http", opts...); err != nil {
			return err
		}
	}

	return nil
}

func (a *SDKSteps) getStoreProvider(agentID string) storage.Provider {
	storeProv := leveldb.NewProvider(dbPath + "/" + agentID + uuid.New().String())
	return storeProv
}

func (a *SDKSteps) createEdgeAgent(agentID, scheme, routeOpt string) error {
	var opts []aries.Option

	storeProv := a.getStoreProvider(agentID)

	if routeOpt != decorator.TransportReturnRouteAll {
		return errors.New("only 'all' transport route return option is supported")
	}

	opts = append(opts,
		aries.WithStoreProvider(storeProv),
		aries.WithTransportReturnRoute(routeOpt),
	)

	switch scheme {
	case webSocketTransportProvider:
		opts = append(opts, aries.WithOutboundTransports(ws.NewOutbound()))
	default:
		return fmt.Errorf("invalid transport provider type : %s (only websocket is supported)", scheme)
	}

	return a.createFramework(agentID, opts...)
}

func (a *SDKSteps) create(agentID, inboundHost, inboundPort, scheme string, opts ...aries.Option) error {
	const (
		portAttempts  = 5
		listenTimeout = 2 * time.Second
	)

	if inboundPort == "random" {
		inboundPort = strconv.Itoa(mustGetRandomPort(portAttempts))
	}

	inboundAddr := fmt.Sprintf("%s:%s", inboundHost, inboundPort)

	switch scheme {
	case webSocketTransportProvider:
		inbound, err := ws.NewInbound(inboundAddr, "ws://"+inboundAddr)
		if err != nil {
			return fmt.Errorf("failed to create websocket: %w", err)
		}

		opts = append(opts, aries.WithInboundTransport(inbound), aries.WithOutboundTransports(ws.NewOutbound()))
	case httpTransportProvider:
		opts = append(opts, defaults.WithInboundHTTPAddr(inboundAddr, "http://"+inboundAddr))
	default:
		return fmt.Errorf("invalid transport provider type : %s", scheme)
	}

	err := a.createFramework(agentID, opts...)
	if err != nil {
		return fmt.Errorf("failed to create new agent: %w", err)
	}

	if err := listenFor(fmt.Sprintf("%s:%s", inboundHost, inboundPort), listenTimeout); err != nil {
		return err
	}

	logger.Debugf("Agent %s start listening on %s:%s", agentID, inboundHost, inboundPort)

	return nil
}

func (a *SDKSteps) createFramework(agentID string, opts ...aries.Option) error {
	agent, err := aries.New(opts...)
	if err != nil {
		return fmt.Errorf("failed to create new agent: %w", err)
	}

	ctx, err := agent.Context()
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}

	a.bddContext.AgentCtx[agentID] = ctx
	a.bddContext.Messengers[agentID] = agent.Messenger()

	return nil
}

// RegisterSteps registers agent steps
func (a *SDKSteps) RegisterSteps(s *godog.Suite) {
	s.Step(`^"([^"]*)" agent is running on "([^"]*)" port "([^"]*)" with "([^"]*)" as the transport provider$`,
		a.createAgent)
	s.Step(`^"([^"]*)" edge agent is running with "([^"]*)" as the outbound transport provider `+
		`and "([^"]*)" as the transport return route option`, a.createEdgeAgent)
	s.Step(`^"([^"]*)" agent is running on "([^"]*)" port "([^"]*)" `+
		`with http-binding did resolver url "([^"]*)" which accepts did method "([^"]*)"$`, a.CreateAgentWithHTTPDIDResolver)
	s.Step(`^"([^"]*)" agent with message registrar is running on "([^"]*)" port "([^"]*)" `+
		`with "([^"]*)" as the transport provider$`, a.createAgentWithRegistrar)
}

func mustGetRandomPort(n int) int {
	for ; n > 0; n-- {
		port, err := getRandomPort()
		if err != nil {
			continue
		}

		return port
	}

	panic("cannot acquire the random port")
}

func getRandomPort() (int, error) {
	const network = "tcp"

	addr, err := net.ResolveTCPAddr(network, "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP(network, addr)
	if err != nil {
		return 0, err
	}

	if err := listener.Close(); err != nil {
		return 0, err
	}

	return listener.Addr().(*net.TCPAddr).Port, nil
}

func listenFor(host string, d time.Duration) error {
	timeout := time.After(d)

	for {
		select {
		case <-timeout:
			return errors.New("timeout: server is not available")
		default:
			conn, err := net.Dial("tcp", host)
			if err != nil {
				continue
			}

			return conn.Close()
		}
	}
}
