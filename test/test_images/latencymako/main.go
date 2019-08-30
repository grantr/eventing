/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/mako/go/quickstore"
	"knative.dev/eventing/test/common"

	cloudevents "github.com/cloudevents/sdk-go"
	vegeta "github.com/tsenart/vegeta/lib"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/test/mako"
	pkgpacers "knative.dev/pkg/test/vegeta/pacers"
)

const (
	defaultEventType   = "perf-test-event-type"
	defaultEventSource = "perf-test-event-source"
)

// flags for the image
var (
	benchmark      string
	sinkURL        string
	msgSize        int
	timeout        int
	deliveredCh    chan deliveredState
	receivedCh     chan receivedState
	resultCh       chan eventStatus
	secondDuration int
	rps            int
	fatalf         func(f string, args ...interface{})
	additionalTags string
	idCounter      int64
)

// eventStatus is status of the event delivery.
type eventStatus int

const (
	sent eventStatus = iota
	received
	undelivered
	dropped
	duplicated // TODO currently unused
	corrupted  // TODO(Fredy-Z): corrupted status is not being used now
)

type requestInterceptor struct {
	before func(*http.Request)
	after  func(*http.Request, *http.Response, error)
}

func (r requestInterceptor) RoundTrip(request *http.Request) (*http.Response, error) {
	if r.before != nil {
		r.before(request)
	}
	res, err := http.DefaultTransport.RoundTrip(request)
	if r.after != nil {
		r.after(request, res, err)
	}
	return res, err
}

type state struct {
	eventId      uint64
	failed       bool
	sendTime time.Time
	publishTime time.Time
	receivedTime time.Time
}

type sentState state
type deliveredState state
type receivedState state

func init() {
	flag.StringVar(&sinkURL, "sink", "", "The sink URL for the event destination.")
	flag.IntVar(&msgSize, "msg-size", 100, "The size of each message we want to send. Generate random strings to avoid caching.")
	flag.IntVar(&secondDuration, "duration", 10, "Duration of the benchmark in seconds")
	flag.IntVar(&rps, "rps", 1000, "Maximum request per seconds")
	flag.StringVar(&additionalTags, "tags", "", "Additional benchmark tags, comma-separated.")
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// generateRandString returns a random string with the given length.
func generateRandString(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func main() {
	// parse the command line flags
	flag.Parse()

	// We want this for properly handling Kubernetes container lifecycle events.
	ctx := signals.NewContext()

	// We cron every 5 minutes, so make sure that we don't severely overrun to
	// limit how noisy a neighbor we can be.
	ctx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()

	extraTags := strings.Split(additionalTags, ",")

	// Use the benchmark key created
	ctx, q, qclose, err := mako.Setup(ctx, extraTags...)
	if err != nil {
		log.Fatalf("Failed to setup mako: %v", err)
	}

	// Use a fresh context here so that our RPC to terminate the sidecar
	// isn't subject to our timeout (or we won't shut it down when we time out)
	defer qclose(context.Background())

	// Wrap fatalf in a helper or our sidecar will live forever.
	fatalf = func(f string, args ...interface{}) {
		qclose(context.Background())
		log.Fatalf(f, args...)
	}

	// We don't know how messages are sent, so we estimate is at most the rate at maximum pace * duration of the benchmark
	pessimisticNumberOfTotalMessages := rps * secondDuration

	// We estimate that the channel reader requires at most 3 seconds to process a message
	pessimisticNumberOfMessagesInsideAChannel := rps * 3

	// Create all channels
	deliveredCh = make(chan deliveredState, pessimisticNumberOfMessagesInsideAChannel)
	receivedCh = make(chan receivedState, pessimisticNumberOfMessagesInsideAChannel)
	resultCh = make(chan eventStatus, pessimisticNumberOfTotalMessages)

	// Start the events receiver
	startCloudEventsReceiver()

	// Start the goroutine that will process the latencies and publish the data points to mako
	go processLatencies(q)

	targeter := common.NewCloudEventsTargeter(sinkURL, msgSize, defaultEventType, defaultEventSource, "binary").VegetaTargeter()

	pacer, err := pkgpacers.NewSteadyUp(
		vegeta.Rate{
			Freq: 100,
			Per:  time.Second,
		},
		vegeta.Rate{
			Freq: rps,
			Per:  time.Second,
		},
		2*time.Second,
	)

	if err != nil {
		fatalf("failed to create pacer: %v\n", err)
	}

	// sleep 30 seconds before sending the events
	// TODO(Fredy-Z): this is a bit hacky, as ideally, we need to wait for the Trigger/Subscription that uses it as a
	//                Subscriber to become ready before sending the events, but we don't have a way to coordinate between them.
	time.Sleep(30 * time.Second)

	client := http.Client{Transport: requestInterceptor{before: func(request *http.Request) {
		sendTime := time.Now()
		idCounter++
		request.Header.Set("Ce-Id", strconv.FormatInt(idCounter, 10))
		// reset time attribute to current time
		request.Header.Set("Ce-Time", sendTime.Format(time.RFC3339Nano))
	}, after: func(request *http.Request, response *http.Response, e error) {
		publishTime := time.Now()
	        sendTime, _ := time.Parse(time.RFC3339Nano, request.Header.Get("Ce-Time"))
		delState := deliveredState{sendTime: sendTime, publishTime: publishTime}
		if e != nil || response.StatusCode < 200 || response.StatusCode >= 300 {
			log.Printf("ERROR: delivery failed: %v\n", err)
			delState.failed = true
		}
		deliveredCh <- delState
	}}}

	vegetaResults := vegeta.NewAttacker(
		vegeta.Client(&client),
	).Attack(targeter, pacer, time.Duration(secondDuration)*time.Second, defaultEventType+"-attack")

	go processVegetaResult(vegetaResults)

	// count errors
	var publishErrorCount int
	var deliverErrorCount int
	for eventState := range resultCh {
		switch eventState {
		case dropped:
			deliverErrorCount++
		case undelivered:
			publishErrorCount++
		}
	}

	// publish error counts as aggregate metrics
	q.AddRunAggregate("pe", float64(publishErrorCount))
	q.AddRunAggregate("de", float64(deliverErrorCount))

	out, err := q.Store()
	if err != nil {
		fatalf("q.Store error: %v: %v", out, err)
	}
}

func startCloudEventsReceiver() {
	t, err := cloudevents.NewHTTPTransport(
		cloudevents.WithTarget(sinkURL),
		cloudevents.WithBinaryEncoding(),
	)
	if err != nil {
		fatalf("failed to create transport: %v\n", err)
	}
	c, err := cloudevents.NewClient(t,
		cloudevents.WithTimeNow(),
		cloudevents.WithUUIDs(),
	)
	if err != nil {
		fatalf("failed to create client: %v\n", err)
	}

	go c.StartReceiver(context.Background(), processReceiveEvent)
}

func processVegetaResult(vegetaResults <-chan *vegeta.Result) {
	// Discard all vegeta results and wait the end of this channel
	for _ = range vegetaResults {
	}

	close(deliveredCh)

	// Let's assume that after 5 seconds all responses are received
	time.Sleep(5 * time.Second)
	close(receivedCh)

	// Let's assume that after 3 seconds all responses are processed
	time.Sleep(3 * time.Second)
	close(resultCh)
}

func processReceiveEvent(event cloudevents.Event) {
	receivedCh <- receivedState{sendTime: event.Context.GetTime(), receivedTime: time.Now()}
}

func processLatencies(q *quickstore.Quickstore) {
	for {
		select {
		case d, ok := <-deliveredCh:
			if ok {
				if d.failed {
					resultCh <- undelivered
					if qerr := q.AddError(mako.XTime(d.sendTime), "undelivered"); qerr != nil {
						log.Printf("ERROR AddError: %v", qerr)
					}
				} else {
					sendLatency := d.publishTime.Sub(d.sendTime)
					if sendLatency < 0 {
						log.Printf("WARN: negative latency %d (%s - %s)", sendLatency, d.publishTime.Format(time.RFC3339Nano), d.sendTime.Format(time.RFC3339Nano))
					}
					// Uncomment to get CSV directly from this container log
					//fmt.Printf("%f,%d,\n", mako.XTime(timestampSent), sendLatency.Nanoseconds())
					// TODO mako accepts float64, which imo could lead to losing some precision on local tests. It should accept int64
					if qerr := q.AddSamplePoint(mako.XTime(d.sendTime), map[string]float64{"pl": sendLatency.Seconds()}); qerr != nil {
						log.Printf("ERROR AddSamplePoint: %v", qerr)
					}
				}
			}
		case r, ok := <-receivedCh:
			if ok {
				if ok {
					e2eLatency := r.receivedTime.Sub(r.sendTime)
					if e2eLatency < 0 {
						log.Printf("WARN: negative latency %d (%s - %s)", e2eLatency, r.receivedTime.Format(time.RFC3339Nano), r.sendTime.Format(time.RFC3339Nano))
					}
					// Uncomment to get CSV directly from this container log
					//fmt.Printf("%f,,%d\n", mako.XTime(timestampSent), e2eLatency.Nanoseconds())
					// TODO mako accepts float64, which imo could lead to losing some precision on local tests. It should accept int64
					if qerr := q.AddSamplePoint(mako.XTime(r.sendTime), map[string]float64{"dl": e2eLatency.Seconds()}); qerr != nil {
						log.Printf("ERROR AddSamplePoint: %v", qerr)
					}
				} else {
					resultCh <- corrupted
				}
			} else {
				return
			}
		}
	}
}
