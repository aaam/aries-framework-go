/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package didexchange

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hyperledger/aries-framework-go/pkg/didcomm/dispatcher"
	"github.com/hyperledger/aries-framework-go/pkg/didcomm/protocol/didexchange"
	"github.com/hyperledger/aries-framework-go/pkg/wallet"
)

// provider contains dependencies for the DID exchange protocol and is typically created by using aries.Context()
type provider interface {
	Service(id string) (interface{}, error)
	CryptoWallet() wallet.Crypto
}

// Client enable access to didexchange api
// TODO add support for Accept Exchange Request & Accept Invitation
//  using events & callback (#198 & #238)
type Client struct {
	didexchangeSvc dispatcher.Service
	wallet         wallet.Crypto
}

// New return new instance of didexchange client
func New(ctx provider) (*Client, error) {
	svc, err := ctx.Service(didexchange.DIDExchange)
	if err != nil {
		return nil, err
	}
	didexchangeSvc, ok := svc.(dispatcher.Service)
	if !ok {
		return nil, errors.New("cast service to DIDExchange Service failed")
	}
	return &Client{didexchangeSvc: didexchangeSvc, wallet: ctx.CryptoWallet()}, nil
}

// CreateInvitation create invitation
func (c *Client) CreateInvitation() (*InvitationRequest, error) {
	// TODO remove nil check after provide default implementation for wallet
	pubKey := ""
	if c.wallet != nil {
		keyInfo, err := c.wallet.CreateSigningKey(nil)
		if err != nil {
			return nil, fmt.Errorf("failed CreateSigningKey: %w", err)
		}
		pubKey = keyInfo.GetVerificationKey()
	}
	return &InvitationRequest{Invitation: &didexchange.Invitation{
		ID:              uuid.New().String(),
		Label:           "agent", // TODO get the value from config #175
		RecipientKeys:   []string{pubKey},
		ServiceEndpoint: "https://example.com/endpoint", // TODO get the value from config #175
	}}, nil
}

// QueryConnections queries connections matching given parameters
func (c *Client) QueryConnections(request *QueryConnectionsParams) ([]*QueryConnectionResult, error) {
	// TODO sample response, to be implemented as part of #226
	return []*QueryConnectionResult{
		{ConnectionID: uuid.New().String(), CreatedTime: time.Now()},
		{ConnectionID: uuid.New().String(), CreatedTime: time.Now()},
	}, nil
}

// QueryConnectionByID fetches single connection record for given id
func (c *Client) QueryConnectionByID(id string) (*QueryConnectionResult, error) {
	// TODO sample response, to be implemented as part of #226
	return &QueryConnectionResult{
		ConnectionID: uuid.New().String(), CreatedTime: time.Now(),
	}, nil
}