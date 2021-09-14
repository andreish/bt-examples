package main

import (
	"context"
	"fmt"
	"html/template"
	"strconv"

	"net/http"
	"os"

	"github.com/braintree-go/braintree-go"
	"github.com/braintree-go/braintree-go/customfields"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	logr "github.com/sirupsen/logrus"
)

func init() {
	loadenv()
}

func LoggerFor(fn string) *logr.Entry {
	return logr.
		WithField("fn", fn)
}

func loadenv() error {
	fname, ok := os.LookupEnv("BTPAY_ENV")
	if !ok {
		fname = ".env"
	}
	return godotenv.Load(fname)
}

const (
	transactionCustomParameterStoreID                   = "store_id"
	transactionCustomParameterStoreName                 = "store_name"
	transactionCustomParameterSubscriptionStartDateTime = "subscription_start_date_time"
)

type BraintreeJS struct {
	Key template.HTML
}

func showIndex(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("showIndex")
	log.Info("--------------------------- new request --------------")
	w.WriteHeader(http.StatusOK)
	return
}

func showForm(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("showForm")
	log.Info("--------------------------- new request --------------")
	ctx := context.Background()
	bt := getBraintree()
	clientToken, err := bt.ClientToken().Generate(ctx)
	if err != nil {
		log.Fatal(err)
	}
	t := template.Must(template.ParseFiles("form.html"))
	w.WriteHeader(http.StatusOK)
	err = t.Execute(w, clientToken)
	if err != nil {
		log.Error(err)
	}
}

var brt *braintree.Braintree

func getBraintree() *braintree.Braintree {
	if nil != brt {
		return brt
	}
	log := LoggerFor("getBraintree")

	log.Infof("BRAINTREE_MERCHANT_ID=%s", os.Getenv("BRAINTREE_MERCHANT_ID"))
	log.Infof("BRAINTREE_PUBLIC_KEY=%s", os.Getenv("BRAINTREE_PUBLIC_KEY"))
	log.Info("BRAINTREE_PRIVATE_KEY=%s", os.Getenv("BRAINTREE_PRIVATE_KEY"))
	log.Info("--- create bt ----")
	brt = braintree.New(
		braintree.Sandbox,
		os.Getenv("BRAINTREE_MERCHANT_ID"),
		os.Getenv("BRAINTREE_PUBLIC_KEY"),
		os.Getenv("BRAINTREE_PRIVATE_KEY"),
	)
	return brt
}

func createTransaction(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	log := LoggerFor("createTransaction")
	bt := getBraintree()
	cc := braintree.CreditCard{
		Number:          r.FormValue("number"),
		CVV:             r.FormValue("cvv"),
		ExpirationMonth: r.FormValue("month"),
		ExpirationYear:  r.FormValue("year"),
	}
	log.Infof("cc=%v", cc)
	//paymentMethodNonce := r.PostFormValue("paymentMethodNonce")
	tx := &braintree.TransactionRequest{
		Type:   "sale",
		Amount: braintree.NewDecimal(100, 2),
		//PaymentMethodNonce: paymentMethodNonce,
		Options: &braintree.TransactionOptions{
			SubmitForSettlement: false,
		},
	}

	_, err := bt.Transaction().Create(ctx, tx)

	if err == nil {
		log.Info("Transaction success")
		_, _ = fmt.Fprintf(w, "<h1>Success!</h1>")
	} else {
		log.Errorf("Transaction failed %v", err)
		_, _ = fmt.Fprintf(w, "<h1>Something went wrong: "+err.Error()+"</h1>")
	}
}

func showSubscriptionForm(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("showSubscriptionForm")
	log.Info("--------------------------- new request --------------")
	ctx := context.Background()
	bt := getBraintree()
	clientToken, err := bt.ClientToken().Generate(ctx)
	if err != nil {
		log.Error("failed to create token")
		log.Error(err)
	}
	t := template.Must(template.ParseFiles("sform.html"))
	w.WriteHeader(http.StatusOK)
	err = t.Execute(w, clientToken)
	if err != nil {
		log.Error(err)
	}
}

func showLoginForm(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("showLoginForm")
	log.Info("--------------------------- new request --------------")

	t := template.Must(template.ParseFiles("login.html"))
	w.WriteHeader(http.StatusOK)
	err := t.Execute(w, nil)
	if err != nil {
		log.Error(err)
	}
}

func dumpVars(log *logr.Entry, vars map[string]string) {
	for k, v := range vars {
		log.Infof("%s->%s", k, v)
	}

}

func GenerateClientToken(ctx context.Context, customerEmail *string, merchantaccountid *string) (string, error) {
	log := LoggerFor("GenerateClientToken")
	if nil == customerEmail && nil == merchantaccountid {
		bt := getBraintree()
		log.Info("bt.ClientToken().Generate")
		return bt.ClientToken().Generate(ctx)
	}
	var customer *braintree.Customer
	if nil != customerEmail {
		bt := getBraintree()
		query := new(braintree.SearchQuery)
		emailField := query.AddTextField("email")
		emailField.Is = *customerEmail

		customerSearch, err := bt.Customer().Search(ctx, query)
		log.Infof("email=%s customerSearch=%v", *customerEmail, customerSearch)
		if err != nil {
			log.Error(err)
		}
		if err == nil && nil != customerSearch && len(customerSearch.Customers) > 0 {
			customer = customerSearch.Customers[0]

		}

	}
	bt := getBraintree()
	req := braintree.ClientTokenRequest{}

	if nil != customer {
		req.CustomerID = customer.Id
	}
	if nil != merchantaccountid {
		req.MerchantAccountID = *merchantaccountid
	}
	log.Infof("bt.ClientToken().Generate customerID=%s merchantaccountid=%s ", req.CustomerID, req.MerchantAccountID)
	return bt.ClientToken().GenerateWithRequest(ctx, &req)
}

func showUserSubscriptionForm(w http.ResponseWriter, r *http.Request) {

	log := LoggerFor("showUserSubscriptionForm")
	log.Info("--------------------------- new request --------------")
	log.Infof("vars: %v", mux.Vars(r))
	dumpVars(log, mux.Vars(r))
	email := r.FormValue("email")
	log.Infof("email=%s", email)

	ctx := context.Background()

	clientTokenGenerator := NewClientTokenGenerator(&email, nil)
	clientToken, err := clientTokenGenerator.GetToken(ctx)
	if nil != err {
		log.Error(err)
	}
	clientId, err := clientTokenGenerator.GetCustomerID(ctx)
	if nil != err {
		log.Error(err)
	}
	if err != nil {
		log.Fatal(err)
	}
	t := template.Must(template.ParseFiles("suform.html"))
	w.WriteHeader(http.StatusOK)
	err = t.Execute(w, map[string]string{"clientToken": clientToken,
		"clientId": clientId, "email": email})
	if err != nil {
		log.Error(err)
	}
}

func createCustomerSubscription(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("createCustomerSubscription")
	log.Info("--------------------------- new request --------------")
	ctx := context.Background()
	bt := getBraintree()
	paymentMethodNonce := r.PostFormValue("paymentMethodNonce")
	paymentMethodIndex := r.PostFormValue("paymentMethodIndex")

	if paymentMethodNonce == "" {
		log.Error("Payment method nonce is empty")
		return
	}
	log.Infof("paymentMethodNonce=%s", paymentMethodNonce)
	log.Infof("paymentMethodIndex=%s", paymentMethodIndex)
	// You can later search for the user by his ID

	customerID := r.PostFormValue("clientId")
	email := r.PostFormValue("email")
	log.Infof("search customer by email=%s customerId=%s", email, customerID)
	customer, err := bt.Customer().Find(ctx, customerID)

	if err != nil {
		log.Info("search by id failed, creating new customer")
		customerReq := braintree.CustomerRequest{
			FirstName:    "John",
			LastName:     "Doe1",
			Email:        email,
			CustomFields: customfields.CustomFields{"userid": "1234"},
			//PaymentMethodNonce: paymentMethodNonce,
		}
		// We create a credit card that after generation, gives us the PaymentMethodToken that's needed in the Subscription.Create
		customer, err = bt.Customer().Create(ctx, &customerReq)
		if err != nil {
			log.Error(err)
		}
		log.Infof("cread customerid=$s", customer.Id)
		paymentMethods := customer.PaymentMethods()
		if len(paymentMethods) == 0 {
			log.Error("no payment methods for customer")

		}
		pmReq := braintree.PaymentMethodRequest{
			CustomerId:         customer.Id,
			PaymentMethodNonce: paymentMethodNonce,
		}
		pmToken, err := bt.PaymentMethod().Create(ctx, &pmReq)
		if nil != err {
			log.WithError(err).Error("failed to create payment method ")
		}
		log.Infof("created payment method token=%s", pmToken.GetToken())

		log.Info("create transaction ---------------------------------->")
		txrq := braintree.TransactionRequest{
			OrderId:            "234234",
			PaymentMethodToken: pmToken.GetToken(),
			Amount:             braintree.NewDecimal(5555, 2),
			Type:               "sale",
			Options: &braintree.TransactionOptions{
				SubmitForSettlement: false,
			},
		}
		t, err := bt.Transaction().Create(ctx, &txrq)
		if nil != err {
			log.WithError(err).Error("creeate transaction failed")
		} else {
			log.Infof("tid=%s status=%s", t.Id, t.Status)
		}

	} else {

		log.Info("customer already created, using its payment method")
		paymentMethods := customer.PaymentMethods()
		if len(paymentMethods) == 0 {
			log.Error("no payment methods for customer")
			return
		}

		log.Info("num payment methods: [%d] ", len(paymentMethods))
		for i, pm := range paymentMethods {
			log.Infof("[%d] -> %s , url=%s", i, pm.GetToken(), pm.GetImageURL())
		}

		log.Infof("default: %s ", customer.DefaultPaymentMethod().GetToken())
		idx, _ := strconv.Atoi(paymentMethodIndex)
		selected := len(paymentMethods) - idx - 1
		pmToken := paymentMethods[selected].GetToken()
		log.Infof("selected: %s ", pmToken)

		log.Info("create transaction ---------------------------------->")
		txrq := braintree.TransactionRequest{
			OrderId:            "12345",
			PaymentMethodToken: pmToken,
			Amount:             braintree.NewDecimal(222, 2),
			Type:               "sale",
			Options: &braintree.TransactionOptions{
				SubmitForSettlement: false,
			},
		}
		t, err := bt.Transaction().Create(ctx, &txrq)
		if nil != err {
			log.WithError(err).Error("creeate transaction failed")
		} else {
			log.Infof("tid=%s status=%s", t.Id, t.Status)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Success! "))
	if err != nil {
		log.Error(err)
	}

}

func dumpSubscriptions(log *logr.Entry, subscriptions []*braintree.Subscription) {
	for i, s := range subscriptions {
		log.Infof("-------------- subscription id=%s start --------------", s.Id)
		log.Info("[%d]-> %v", i, s)
		if nil != s.Transactions {
			log.Infof("Num Transactions: %d", len(s.Transactions.Transaction))
		} else {
			log.Info("No transactions")
		}
		log.Infof("-------------- subscription id=%s end --------------", s.Id)
	}
}

func createSubscription(w http.ResponseWriter, r *http.Request) {
	log := LoggerFor("createSubscription")
	log.Info("--------------------------- new request --------------")
	ctx := context.Background()
	bt := getBraintree()
	paymentMethodNonce := r.PostFormValue("paymentMethodNonce")
	if paymentMethodNonce == "" {
		log.Error("Payment method nonce is empty")
		return
	}
	log.Infof("paymentMethodNonce=%s", paymentMethodNonce)
	// You can later search for the user by his ID
	// customer, err := bt.Customer().Find("CustomerID")
	customer, err := bt.Customer().Create(ctx, &braintree.CustomerRequest{
		// You can leave it empty, but, if you've got a user system, I recommend using the user's ID as the client ID
		// Or, createa a row for Braintree's customer ID
		ID: "",
	})
	if err != nil {
		log.Error(err)
		return
	}

	// We create a credit card that after generation, gives us the PaymentMethodToken that's needed in the Subscription.Create
	card, err := bt.CreditCard().Create(ctx, &braintree.CreditCard{
		// The created or existing customer ID
		CustomerId: customer.Id,
		// The nonce from the clinet side
		PaymentMethodNonce: paymentMethodNonce,
		Options: &braintree.CreditCardOptions{
			VerifyCard: func(b bool) *bool { return &b }(false),
		},
	})
	if err != nil {
		log.Error(err)
	}

	// Create the subscription and make the user pay
	subscription, err := bt.Subscription().Create(ctx, &braintree.SubscriptionRequest{
		PlanId: "subscriptionED",
		// The payment method token generated by the CreditCard.Create
		PaymentMethodToken: card.Token,
	})
	if err != nil {
		log.Error(err)
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(fmt.Sprintf("Success! Subscription #%s created with user ID %s", subscription.Id, customer.Id)))
	if err != nil {
		log.Error(err)
	}
}

func main() {
	log := LoggerFor("main")
	http.HandleFunc("/", showIndex)
	http.HandleFunc("/t", showForm)
	http.HandleFunc("/s", showSubscriptionForm)
	http.HandleFunc("/c", showUserSubscriptionForm)
	http.HandleFunc("/l", showLoginForm)
	http.HandleFunc("/checkout", createSubscription)
	http.HandleFunc("/createtransaction", createTransaction)
	http.HandleFunc("/createusersubscription", createCustomerSubscription)
	log.Info("starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
