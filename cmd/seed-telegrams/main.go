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

// SeedConfig controls how telegrams are generated and dispatched.
type SeedConfig struct {
	Count        int
	Mode         string
	IntervalMin  time.Duration
	IntervalMax  time.Duration
	Duration     time.Duration
	CategoryFlag string
	StatusFlag   string
	ErrorReason  string
	HeaderFormat string
	DryRun       bool
}

// publishFunc abstracts the publish side-effect so that it can be swapped in tests.
type publishFunc func(*telegram) error

// sleepFunc is used instead of time.Sleep so tests can stub out real sleeping.
var sleepFunc = time.Sleep

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
	mode := flag.String("mode", "burst", "Seed mode: burst|interval|mixed")
	intervalMin := flag.Duration("interval-min", time.Second, "Minimum interval between messages in interval/mixed modes")
	intervalMax := flag.Duration("interval-max", 2*time.Second, "Maximum interval between messages in interval/mixed modes")
	duration := flag.Duration("duration", 0, "Total duration for interval/mixed modes (0 = rely on --count only)")
	flag.Parse()

	// rand.Seed deprecated in Go 1.20+, using default source

	var nc *nats.Conn
	var js nats.JetStreamContext
	var err error

	if !*dryRun && *natsURL != "" && !*noNATS {
		nc, err = nats.Connect(*natsURL)
		if err != nil {
			log.Fatalf("connect nats: %v", err)
		}
		defer func() {
			if err := nc.Drain(); err != nil {
				log.Printf("failed to drain NATS connection: %v", err)
			}
		}()

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

	// Normalize and prepare configuration.
	if *intervalMax < *intervalMin {
		*intervalMax = *intervalMin
	}

	cfg := SeedConfig{
		Count:        *count,
		Mode:         strings.ToLower(*mode),
		IntervalMin:  *intervalMin,
		IntervalMax:  *intervalMax,
		Duration:     *duration,
		CategoryFlag: *category,
		StatusFlag:   *status,
		ErrorReason:  *errorReason,
		HeaderFormat: *headerFormat,
		DryRun:       *dryRun,
	}

	categories := buildCategories(cfg.CategoryFlag, bodyCategories)
	statuses := buildStatuses(cfg.StatusFlag, statusValues)

	var publisher publishFunc
	if nc != nil && !*noNATS {
		publisher = func(payload *telegram) error {
			data := []byte(payload.Content)
			msg := &nats.Msg{Subject: *subject, Data: data, Header: nats.Header{}}
			msg.Header.Set("Nats-Msg-Id", payload.UUID)
			if strings.ToLower(cfg.HeaderFormat) == "json" {
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
					return fmt.Errorf("jetstream publish: %w", err)
				}
				return nil
			}

			if err := nc.PublishMsg(msg); err != nil {
				return fmt.Errorf("nats publish: %w", err)
			}
			return nil
		}
	}

	if err := RunSeed(cfg, categories, statuses, publisher); err != nil {
		log.Fatalf("run seed: %v", err)
	}

	if !*dryRun {
		if nc != nil && !*noNATS {
			log.Printf("Published %d telegram(s) to %s", *count, *subject)
		}
	}
}

// RunSeed drives telegram generation according to the configured mode.
// It delegates message construction to buildTelegram and side-effects to the provided publisher.
func RunSeed(cfg SeedConfig, categories []string, statuses []string, publisher publishFunc) error {
	if len(categories) == 0 {
		return fmt.Errorf("no categories available")
	}
	if len(statuses) == 0 {
		return fmt.Errorf("no statuses available")
	}

	switch cfg.Mode {
	case "interval":
		return runIntervalMode(cfg, categories, statuses, publisher)
	case "mixed":
		return runMixedMode(cfg, categories, statuses, publisher)
	default:
		// Default to burst semantics.
		return runBurstMode(cfg, categories, statuses, publisher)
	}
}

func runBurstMode(cfg SeedConfig, categories []string, statuses []string, publisher publishFunc) error {
	count := cfg.Count
	if count <= 0 {
		return nil
	}
	for i := 0; i < count; i++ {
		if err := sendTelegram(i, cfg, categories, statuses, publisher); err != nil {
			return err
		}
	}
	return nil
}

func runIntervalMode(cfg SeedConfig, categories []string, statuses []string, publisher publishFunc) error {
	start := time.Now()
	sent := 0

	for cfg.Count <= 0 || sent < cfg.Count {
		if cfg.Duration > 0 && time.Since(start) >= cfg.Duration {
			break
		}

		if err := sendTelegram(sent, cfg, categories, statuses, publisher); err != nil {
			return err
		}
		sent++

		// Compute next sleep duration within [IntervalMin, IntervalMax].
		sleepDur := cfg.IntervalMin
		if cfg.IntervalMax > cfg.IntervalMin {
			delta := cfg.IntervalMax - cfg.IntervalMin
			sleepDur = cfg.IntervalMin + time.Duration(rand.Int63n(int64(delta)+1))
		}
		if sleepDur > 0 && !cfg.DryRun {
			sleepFunc(sleepDur)
		}
	}

	return nil
}

// mixed 模式：前半段使用 interval 模式，后半段使用 burst。
func runMixedMode(cfg SeedConfig, categories []string, statuses []string, publisher publishFunc) error {
	// 简单策略：如果 Count>0，前半部分 interval，后半部分 burst；否则退化为 interval。
	if cfg.Count <= 0 {
		return runIntervalMode(cfg, categories, statuses, publisher)
	}

	half := cfg.Count / 2
	if half == 0 {
		// Count==1 时直接按 burst 处理。
		return runBurstMode(cfg, categories, statuses, publisher)
	}

	intervalCfg := cfg
	intervalCfg.Count = half
	if err := runIntervalMode(intervalCfg, categories, statuses, publisher); err != nil {
		return err
	}

	burstCfg := cfg
	burstCfg.Count = cfg.Count - half
	return runBurstMode(burstCfg, categories, statuses, publisher)
}

// sendTelegram builds a single telegram, assigns status/metadata, and either prints or publishes it.
func sendTelegram(iteration int, cfg SeedConfig, categories []string, statuses []string, publisher publishFunc) error {
	cat := categories[rand.Intn(len(categories))]
	payload, intentionallyInvalid := buildTelegram(cat)

	payload.Status = chooseStatus(intentionallyInvalid, cfg.StatusFlag, statuses)
	payload.ErrorReason = cfg.ErrorReason
	payload.Metadata = map[string]string{
		"message_id": payload.MessageID,
		"category":   payload.Category,
		"comments":   fmt.Sprintf("seeded iteration=%d", iteration),
		"status":     payload.Status,
	}

	if cfg.DryRun {
		blob, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(blob))
		fmt.Println("---")
		// Still call publisher in DryRun mode for test purposes
		if publisher != nil {
			return publisher(payload)
		}
		return nil
	}

	if publisher != nil {
		return publisher(payload)
	}
	return nil
}

// buildCategories returns the effective categories list based on the CLI flag.
func buildCategories(flagValue string, all []string) []string {
	if strings.ToLower(flagValue) == "mixed" || flagValue == "" {
		return all
	}
	return []string{strings.ToUpper(flagValue)}
}

// buildStatuses returns the effective statuses list based on the CLI flag.
func buildStatuses(flagValue string, all []string) []string {
	if strings.ToLower(flagValue) == "random" || flagValue == "" {
		return all
	}
	return []string{strings.ToLower(flagValue)}
}

// chooseStatus encapsulates the status selection logic, including the special \"random\" behaviour.
func chooseStatus(intentionallyInvalid bool, statusFlag string, statuses []string) string {
	if strings.ToLower(statusFlag) != "random" {
		return statuses[rand.Intn(len(statuses))]
	}

	// random 模式：根据报文是否合法，对 parsed/body_error 做倾向性选择。
	if intentionallyInvalid {
		// 非法报文：大概率 body_error。
		if rand.Intn(100) < 80 {
			return "body_error"
		}
	} else {
		// 合法报文：大概率 parsed。
		if rand.Intn(100) < 70 {
			return "parsed"
		}
	}

	return statuses[rand.Intn(len(statuses))]
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

func buildTelegram(category string) (*telegram, bool) {
	now := time.Now().UTC()
	messageID := fmt.Sprintf("%s%04d", category, rand.Intn(9000)+1000)
	headerTime := now.Format("020304")
	priority := priorityIndicators[rand.Intn(len(priorityIndicators))]
	primary := primaryAddresses[rand.Intn(len(primaryAddresses))]
	originLine := originatorLines[rand.Intn(len(originatorLines))]
	originator := originators[rand.Intn(len(originators))]

	body, intentionallyInvalid := buildBody(category)

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
	}, intentionallyInvalid
}

func buildBody(category string) (string, bool) {
	flight := fmt.Sprintf("%s%04d", []string{"CCA", "SWA", "DLH", "AAL", "JAE"}[rand.Intn(5)], rand.Intn(9000)+1000)
	dep := airports[rand.Intn(len(airports))]
	arr := airports[rand.Intn(len(airports))]
	depTime := time.Now().UTC().Add(time.Duration(rand.Intn(240)) * time.Minute).Format("1504")
	arrTime := time.Now().UTC().Add(time.Duration(rand.Intn(360)) * time.Minute).Format("1504")

	switch category {
	case "ARR":
		if rand.Intn(2) == 0 {
			return fmt.Sprintf("(ARR-%s-%s-%s%s)", flight, dep, arr, arrTime), false
		}
		// 70% 使用简单 SSR（合法），30% 使用复杂 SSR（当前正则下非法）
		if rand.Intn(100) < 30 {
			return fmt.Sprintf("(ARR-%s/%s-%s-%s%s)", flight, randomComplexSSR(), dep, arr, arrTime), true
		}
		return fmt.Sprintf("(ARR-%s/%s-%s-%s%s)", flight, randomSimpleSSR(), dep, arr, arrTime), false
	case "DEP":
		// 70% 使用简单 SSR（合法），30% 使用复杂 SSR（当前正则下非法）
		if rand.Intn(100) < 30 {
			return fmt.Sprintf("(DEP-%s/%s-%s%s-%s)", flight, randomComplexSSR(), dep, depTime, arr), true
		}
		return fmt.Sprintf("(DEP-%s/%s-%s%s-%s)", flight, randomSimpleSSR(), dep, depTime, arr), false
	case "CNL":
		return fmt.Sprintf("(CNL-%s-%s-%s)", flight, dep, arr), false
	case "DLA":
		return fmt.Sprintf("(DLA-%s-%s%s-%s)", flight, dep, depTime, arr), false
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
		), false
	}
}

func randomSSR() string {
	return []string{"A0132", "A5633", "SXIRPZJWY/LB101", "SHID/C"}[rand.Intn(4)]
}

// 与当前 ARR/DEP 正则匹配的简单 SSR
func randomSimpleSSR() string {
	return []string{"A0132", "A5633"}[rand.Intn(2)]
}

// 故意构造为当前 ARR/DEP 正则无法解析的复杂 SSR
func randomComplexSSR() string {
	return []string{"SXIRPZJWY/LB101", "SHID/C"}[rand.Intn(2)]
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
