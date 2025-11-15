package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

var (
	priorityIndicators = []string{"FF", "GG", "QU"}
	primaryAddresses   = []string{"ZBTJZPZX", "KSFOZPZX", "KLAXZPZX", "EDDFZPZX"}
	originators        = []string{"ZBTJYOYX", "KSFOYOYX", "SELOZKE"}
	originatorLines    = []string{"141604 ZBACZQZX", "150551 ZBTJUOBK", "210930 ZGGGZQZX"}
	airports           = []string{"ZBTJ", "ZGGG", "KSFO", "KLAX", "EDDF", "RJTT", "EGLL", "ZSPD"}
	statusValues       = []string{"parsed", "header_error", "body_error", "publish_error", "repository_error"}
	bodyCategories     = []string{"ARR", "DEP", "CNL", "DLA", "FPL"}
)

func main() {
	natsURL := flag.String("nats-url", "nats://127.0.0.1:4222", "NATS server URL (empty skips publish)")
	subject := flag.String("subject", "telegram.raw", "Subject to publish telegrams to")
	noNATS := flag.Bool("no-nats", false, "Skip NATS publish even if --nats-url is provided")
	count := flag.Int("count", 10, "Number of telegrams to publish")
	category := flag.String("category", "mixed", "ARR|DEP|CNL|DLA|FPL|mixed")
	status := flag.String("status", "body_error", "parsed|header_error|body_error|publish_error|repository_error|random")
	errorReason := flag.String("error-reason", "synthetic test payload", "Metadata header describing why message is in raw state")
	dryRun := flag.Bool("dry-run", false, "Print telegrams instead of publishing to NATS")
	useJetStream := flag.Bool("jetstream", false, "Publish via JetStream")
	jsStream := flag.String("stream", "", "JetStream stream (optional when --jetstream)")
	jsSubject := flag.String("js-subject", "", "Override subject for JetStream publish (defaults to --subject)")
	headerFormat := flag.String("header-format", "json", "Metadata header encoding: json|none")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	var nc *nats.Conn
	var js nats.JetStreamContext
	var err error

	if !*dryRun && *natsURL != "" && !*noNATS {
		nc, err = nats.Connect(*natsURL)
		if err != nil {
			log.Fatalf("connect nats: %v", err)
		}
		defer nc.Drain()

		if *useJetStream {
			opts := []nats.JSOpt{}
			if *jsStream != "" {
				opts = append(opts, nats.PublishAsyncMaxPending(256))
			}
			js, err = nc.JetStream()
			if err != nil {
				log.Fatalf("init jetstream: %v", err)
			}
			_ = opts
		}
	}

	categories := bodyCategories
	if strings.ToLower(*category) != "mixed" {
		categories = []string{strings.ToUpper(*category)}
	}

	statuses := statusValues
	if strings.ToLower(*status) != "random" {
		statuses = []string{strings.ToLower(*status)}
	}

	for i := 0; i < *count; i++ {
		cat := categories[rand.Intn(len(categories))]
		payload := buildTelegram(cat)
		payload.Status = statuses[rand.Intn(len(statuses))]
		payload.ErrorReason = *errorReason
		payload.Metadata = map[string]string{
			"message_id": payload.MessageID,
			"category":   payload.Category,
			"comments":   fmt.Sprintf("seeded iteration=%d", i),
			"status":     payload.Status,
		}

		if *dryRun {
			blob, _ := json.MarshalIndent(payload, "", "  ")
			fmt.Println(string(blob))
			fmt.Println("---")
			continue
		}

		if nc != nil && !*noNATS {
			data := []byte(payload.Content)
			msg := &nats.Msg{Subject: *subject, Data: data, Header: nats.Header{}}
			msg.Header.Set("Nats-Msg-Id", payload.UUID)
			if strings.ToLower(*headerFormat) == "json" {
				headerJSON, _ := json.Marshal(payload.Metadata)
				msg.Header.Set("x-telegram-meta", string(headerJSON))
			}
			msg.Header.Set("x-telegram-uuid", payload.UUID)
			msg.Header.Set("x-telegram-status", payload.Status)
			msg.Header.Set("x-telegram-error", payload.ErrorReason)

			if js != nil {
				pubSubject := *jsSubject
				if pubSubject == "" {
					pubSubject = *subject
				}
				msg.Subject = pubSubject
				if _, err := js.PublishMsg(msg); err != nil {
					log.Fatalf("jetstream publish: %v", err)
				}
			} else {
				if err := nc.PublishMsg(msg); err != nil {
					log.Fatalf("nats publish: %v", err)
				}
			}
		}
	}

	if !*dryRun {
		if nc != nil && !*noNATS {
			log.Printf("Published %d telegram(s) to %s", *count, *subject)
		}
	}
}

type telegram struct {
	UUID        string            `json:"uuid"`
	MessageID   string            `json:"message_id"`
	Category    string            `json:"category"`
	Status      string            `json:"status"`
	ErrorReason string            `json:"error_reason"`
	Content     string            `json:"content"`
	ReceivedAt  time.Time         `json:"received_at"`
	Metadata    map[string]string `json:"metadata"`
}

func buildTelegram(category string) *telegram {
	now := time.Now().UTC()
	messageID := fmt.Sprintf("%s%04d", category, rand.Intn(9000)+1000)
	headerTime := now.Format("020304")
	priority := priorityIndicators[rand.Intn(len(priorityIndicators))]
	primary := primaryAddresses[rand.Intn(len(primaryAddresses))]
	originLine := originatorLines[rand.Intn(len(originatorLines))]
	originator := originators[rand.Intn(len(originators))]

	body := buildBody(category)

	content := strings.Join([]string{
		fmt.Sprintf("ZCZC %s %s", messageID, headerTime),
		fmt.Sprintf("%s %s", priority, primary),
		originLine,
		originator,
		body,
		"NNNN",
	}, "\n")

	return &telegram{
		UUID:       uuid.NewString(),
		MessageID:  messageID,
		Category:   category,
		Content:    content,
		ReceivedAt: now,
	}
}

func buildBody(category string) string {
	flight := fmt.Sprintf("%s%04d", []string{"CCA", "SWA", "DLH", "AAL", "JAE"}[rand.Intn(5)], rand.Intn(9000)+1000)
	dep := airports[rand.Intn(len(airports))]
	arr := airports[rand.Intn(len(airports))]
	depTime := time.Now().UTC().Add(time.Duration(rand.Intn(240)) * time.Minute).Format("1504")
	arrTime := time.Now().UTC().Add(time.Duration(rand.Intn(360)) * time.Minute).Format("1504")

	switch category {
	case "ARR":
		if rand.Intn(2) == 0 {
			return fmt.Sprintf("(ARR-%s-%s-%s%s)", flight, dep, arr, arrTime)
		}
		return fmt.Sprintf("(ARR-%s/%s-%s-%s%s)", flight, randomSSR(), dep, arr, arrTime)
	case "DEP":
		return fmt.Sprintf("(DEP-%s/%s-%s%s-%s)", flight, randomSSR(), dep, depTime, arr)
	case "CNL":
		return fmt.Sprintf("(CNL-%s-%s-%s)", flight, dep, arr)
	case "DLA":
		return fmt.Sprintf("(DLA-%s-%s%s-%s)", flight, dep, depTime, arr)
	default: // FPL
		return fmt.Sprintf(`(FPL-%s-IS
-%s/H
-%s
-%s%s
-K%04dS%04d %s
-%s%s %s
-%s)`,
			flight,
			randomAircraft(),
			randomSSR(),
			dep,
			depTime,
			rand.Intn(9000)+500,
			rand.Intn(8000)+400,
			randomRoute(),
			arr,
			arrTime,
			randomAirportPair(),
			randomOtherInfo(),
		)
	}
}

func randomSSR() string {
	return []string{"A0132", "A5633", "SXIRPZJWY/LB101", "SHID/C"}[rand.Intn(4)]
}

func randomAircraft() string {
	return []string{"A332", "B788", "MA60", "A359"}[rand.Intn(4)]
}

func randomRoute() string {
	routes := []string{
		"PIAKS G330 PIMOL A539 BTO W82 DOGAR",
		"CG J1 FZ",
		"EBAYY Q11 BSR J65 BCE Q135 KICNE",
	}
	return routes[rand.Intn(len(routes))]
}

func randomAirportPair() string {
	return fmt.Sprintf("%s %s", airports[rand.Intn(len(airports))], airports[rand.Intn(len(airports))])
}

func randomOtherInfo() string {
	return []string{
		"PBN/A1B2B3B4B5D1L1 NAV/ABAS REG/B6513 EET/ZBPE0112 SEL/KMAL PER/C RIF/FRT N640 ZBYN RMK/TCAS EQUIPPED",
		"REG/B3710 SEL/ RMK/TCAS",
		"NAV/RNAV1 RNAV5 RNP4 RMK/AGCS EQUIPPED",
	}[rand.Intn(3)]
}
