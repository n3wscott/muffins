package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
)

type envConfig struct {
	Port int    `envconfig:"PORT" default:"8080" required:"true"`
	Sink string `envconfig:"K_SINK" required:"true"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("failed to process env var: %s", err)
	}

	c, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(env.Sink))
	if err != nil {
		log.Fatalf("failed to make cloudevents client: %v\n", err)
	}
	om := &OctoMuffin{
		client: c,
	}

	if err := om.Bake(context.Background(), time.Second*15); err != nil {
		log.Fatalf("failed to start server, %s", err.Error())
	}
}

type OctoMuffin struct {
	client cloudevents.Client
}

type ingredient struct {
	Amount string
	Name   string
}

var ingredients = []ingredient{{
	Amount: "2 cups",
	Name:   "all-purpose flour",
}, {
	Amount: "3 teaspoons",
	Name:   "baking powder",
}, {
	Amount: "½ teaspoon",
	Name:   "salt",
}, {
	Amount: "¾ cup",
	Name:   "white sugar",
}, {
	Amount: "1",
	Name:   "egg",
}, {
	Amount: "1 cup",
	Name:   "milk",
}, {
	Amount: "¼ cup",
	Name:   "vegetable oil",
}}

var coffeeShops = []string{
	"Tougo Coffee",
	"Squirrel Chops",
	"Victrola Cafe and Roastery",
}

type batchIngredient struct {
	Amount    string
	Name      string
	Batch     string
	Inventory string
}

// Lot is a part of a batch.
type lot struct {
	Name  string
	Batch string
	Lot   string
}

func randomID() string {
	token := make([]byte, 6)
	_, _ = rand.Read(token)
	return base64.URLEncoding.EncodeToString(token)
}

func (om *OctoMuffin) Bake(ctx context.Context, timeToFirstMuffin time.Duration) error {
	ticker := time.NewTicker(timeToFirstMuffin)
	defer ticker.Stop()
	for {
		batch := randomID()
		// Report Batch Ingredients.
		for _, i := range ingredients {
			event := newEvent("com.n3wscott.atlanta.octomuffin.ingredient", batch, &batchIngredient{
				Amount:    i.Amount,
				Name:      i.Name,
				Batch:     batch,
				Inventory: randomID(),
			})
			if result := om.client.Send(ctx, event); cloudevents.IsUndelivered(result) {
				log.Printf("failed to send cloudevent: %v\n", result.Error())
			}
		}
		for _, cs := range coffeeShops {
			event := newEvent("com.n3wscott.atlanta.octomuffin.lot", batch, &lot{
				Name:  "Sent to " + cs,
				Lot:   randomID(),
				Batch: batch,
			})
			if result := om.client.Send(ctx, event); cloudevents.IsUndelivered(result) {
				log.Printf("failed to send cloudevent: %v\n", result.Error())
			}
		}

		// Wait for a tick.
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			continue
		}
	}
}

func newEvent(eventType, batch string, data interface{}) cloudevents.Event {
	event := cloudevents.NewEvent() // Sets version
	event.SetType(eventType)
	event.SetSource("github.com/n3wscott/octomuffin")
	event.SetSubject(batch)
	if err := event.SetData(cloudevents.ApplicationJSON, data); err != nil {
		log.Printf("failed to cloudevents event: %v\n", err)
	}
	return event
}
