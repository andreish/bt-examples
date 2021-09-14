package main

import (
	"context"

	"github.com/braintree-go/braintree-go"
)

type ClientTokenGenerator struct {
	CustomerEmail     *string
	MerchantAccountID *string
	customer          *braintree.Customer
	token             *string
}

func NewClientTokenGenerator(customerEmail *string, merchantAccountId *string) *ClientTokenGenerator {
	return &ClientTokenGenerator{
		CustomerEmail:     customerEmail,
		MerchantAccountID: merchantAccountId,
	}
}

func (g *ClientTokenGenerator) generate(ctx context.Context) (err error) {
	log := LoggerFor("ClientTokenGenerator.generate")
	if nil != g.token {
		return
	}

	if nil == g.CustomerEmail && nil == g.MerchantAccountID {
		bt := getBraintree()
		log.Info("create generic client token")
		token, err := bt.ClientToken().Generate(ctx)
		g.token = &token
		return err
	}

	bt := getBraintree()
	if nil != g.CustomerEmail {

		query := new(braintree.SearchQuery)
		emailField := query.AddTextField("email")
		emailField.Is = *g.CustomerEmail

		customerSearch, err := bt.Customer().Search(ctx, query)
		log.Infof("email=%s customerSearch=%v", *g.CustomerEmail, customerSearch)
		if err != nil {
			log.Error(err)
		}
		if err == nil && nil != customerSearch && len(customerSearch.Customers) > 0 {
			g.customer = customerSearch.Customers[0]
			log.Infof("found customer: %s", g.customer.Id)
		}
	}

	req := braintree.ClientTokenRequest{}

	if nil != g.customer {
		req.CustomerID = g.customer.Id
	}
	if nil != g.MerchantAccountID {
		req.MerchantAccountID = *g.MerchantAccountID
	}
	log.Infof("bt.ClientToken().Generate customerID=%s merchantaccountid=%s ", req.CustomerID, req.MerchantAccountID)
	token, err := bt.ClientToken().GenerateWithRequest(ctx, &req)
	g.token = &token
	return err
}

func (g *ClientTokenGenerator) GetToken(ctx context.Context) (string, error) {
	log := LoggerFor("ClientTokenGenerator.GetToken")
	err := g.generate(ctx)
	log.Infof("g.token=%v", g.token)
	return *g.token, err
}

func (g *ClientTokenGenerator) GetCustomerID(ctx context.Context) (string, error) {
	log := LoggerFor("ClientTokenGenerator.GetCustomerID")
	err := g.generate(ctx)
	if nil != g.customer {
		log.Info("get a customer")
		return g.customer.Id, err
	}
	return "", nil
}
