package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	stripe "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/webhook"
)

// port - default port to start application on
const port = ":8090"

// webhook event types
const (
	stripeEventTypeSubscriptionUpdated string = "customer.subscription.updated"
	// canceled subscription
	stripeEventTypeSubscriptionDeleted string = "customer.subscription.deleted"
	// card deletion event
	stripeEventTypeSourceDeleted string = "customer.source.deleted"
)

func validateSignature(payload []byte, header, secret string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, header, secret)
}

func main() {
	secret := os.Getenv("SIGNING_SECRET")
	if secret == "" {
		fmt.Println("SIGNING_SECRET env variable is required")
		os.Exit(1)
	}

	// preparing HTTP server
	srv := &http.Server{Addr: port, Handler: http.DefaultServeMux}

	// incoming stripe webhook handler
	http.HandleFunc("/stripe", func(resp http.ResponseWriter, req *http.Request) {
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		// validating signature
		event, err := validateSignature(body, req.Header.Get("Stripe-Signature"), secret)
		if err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			fmt.Printf("Failed to validate signature: %s", err)
			return
		}

		switch event.Type {
		case stripeEventTypeSubscriptionUpdated, stripeEventTypeSubscriptionDeleted:
			// subscription status change
			customerID, ok := event.Data.Obj["customer"].(string)
			if !ok {
				fmt.Println("customer key missing from event.Data.Obj")
				return
			}

			subStatus, ok := event.Data.Obj["status"].(string)
			if !ok {
				fmt.Println("status key missing from event.Data.Obj")
				return
			}

			fmt.Printf("customer %s subscription updated, current status: %s \n", customerID, subStatus)
		case stripeEventTypeSourceDeleted:
			customerID, ok := event.Data.Obj["customer"].(string)
			if !ok {
				fmt.Println("customer key missing from event.Data.Obj")
				return
			}
			fmt.Printf("card deleted for customer %s \n", customerID)
		}
	})

	fmt.Printf("Receiving Stripe webhooks on http://localhost%s/stripe \n", port)
	// starting server
	err := srv.ListenAndServe()

	if err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}
}
