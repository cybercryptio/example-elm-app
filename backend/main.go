package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"elm-backend/version"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var RunTimestamp time.Time
var Logger zerolog.Logger

type StatusResponse struct {
	Hostname             string `json:"hostname"`
	RunTimestampUnix     int    `json:"run_timestamp_unix"`
	RunTimestampRFC3339  string `json:"run_timestamp_rfc3339"`
	RunTimestampUnixDate string `json:"run_timestamp_unixdate"`
}

func indexPlainText(w http.ResponseWriter, hostname string, count int, extraText string) {
	if extraText != "" {
		fmt.Fprintf(w, "%s %s %d", extraText, hostname, count)
	} else {
		fmt.Fprintf(w, "%s %d", hostname, count)
	}
	fmt.Fprint(w, "\n")
}

func indexHTML(w http.ResponseWriter, hostname string, count int, extraText string) {
	extraTextBlock := ""
	if extraText != "" {
		extraTextBlock = `<h2>` + extraText + `</h2>`
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<style>
	html, body {
		height: 100%;
	}
	.center-parent {
		width: 100%;
		height: 100%;
		display: table;
		text-align: center;
	}
	.center-parent > .center-child {
		display: table-cell;
		vertical-align: middle;
	}
	</style>
	<style>
	h1 {
		font-family: Arial;
		font-size: 5em;
	}
	h2 {
		font-family: Arial;
		font-size: 2em;
	}
	</style>
	<link rel="icon" href="/favicon.ico">
	<section class="center-parent">
		<div class="center-child">
			`+extraTextBlock+`
			<h1>
				`+strconv.Itoa(count)+`
			</h1>
			<h2>`+hostname+`</h2>
		</div>
	</section>
	`)
}

func versionAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(map[string]string{
		"version": version.Version,
	})
	fmt.Fprint(w, string(data))
}

func livez(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(map[string]bool{
		"live": true,
	})
	fmt.Fprint(w, string(data))
}

func readyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(map[string]bool{
		"ready": true,
	})
	fmt.Fprint(w, string(data))
}

func status(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(StatusResponse{
		Hostname:             hostname,
		RunTimestampUnix:     int(RunTimestamp.Unix()),
		RunTimestampRFC3339:  RunTimestamp.Format(time.RFC3339),
		RunTimestampUnixDate: RunTimestamp.Format(time.UnixDate),
	})
	fmt.Fprint(w, string(data))
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func doCount(redisHost, hostname string) int {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost + ":6379",
		Password: "",
		DB:       0,
	})

	val, err := rdb.Get(ctx, "counter").Result()
	if err != nil {
		if err == redis.Nil {
			val = "0"
		} else {
			log.Error().
				Str("hostname", hostname).
				Msg(fmt.Sprintf("error=%s", err))
			return -1
		}
	}

	counter, err := strconv.Atoi(val)
	if err != nil {
		log.Error().
			Str("hostname", hostname).
			Msg(fmt.Sprintf("error=%s", err))
		return -1
	}

	err = rdb.Set(ctx, "counter", strconv.Itoa(counter+1), 0).Err()
	if err != nil {
		log.Error().
			Str("hostname", hostname).
			Msg(fmt.Sprintf("error=%s", err))
		return -1
	}

	return counter
}

func main() {
	var err error

	Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	RunTimestamp = time.Now()

	hostname, _ := os.Hostname()

	port := "80"
	envPort := os.Getenv("PORT")
	if envPort != "" {
		port = envPort
	}

	redisHost := "127.0.0.1"
	envRedisHost := os.Getenv("REDIS")
	if envRedisHost != "" {
		redisHost = envRedisHost
	}

	slowStart := 0
	envSlowStart := os.Getenv("SLOW_START")
	if envSlowStart != "" {
		slowStart, err = strconv.Atoi(envSlowStart)
		if err != nil {
			Logger.Fatal().Str("hostname", hostname).Msg(`cannot parse integer form SLOW_START value "` + envSlowStart + `", original Go error: ` + err.Error())
		}
	}

	extraText := os.Getenv("EXTRA_TEXT")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		counter := doCount(redisHost, hostname)
		// Check if User-Agent header exists
		if userAgentList, ok := r.Header["User-Agent"]; ok {
			// Check if User-Agent header has some data
			if len(userAgentList) > 0 {
				// If User-Agent starts with curl, use plain text
				if strings.HasPrefix(userAgentList[0], "curl") {
					indexPlainText(w, hostname, counter, extraText)
				} else {
					// If User-Agent header presents and not starts with curl
					// use HTML (Chrome, Safari, Firefox, ...)
					indexHTML(w, hostname, counter, extraText)
				}
			}
		} else {
			// If User-Agent header doesn't exists, use plain text
			indexPlainText(w, hostname, counter, extraText)
		}
		Logger.Info().
			Str("hostname", hostname).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("counter", counter).
			Msg(r.Method + " " + r.URL.Path)
	})
	http.HandleFunc("/api/counter", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		counter := doCount(redisHost, hostname)
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(map[string]int{
			"counter": counter,
		})
		fmt.Fprint(w, string(data))
		fmt.Fprint(w, "\n")
		Logger.Info().
			Str("hostname", hostname).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("counter", counter).
			Msg(r.Method + " " + r.URL.Path)
	})
	http.HandleFunc("/api/version", versionAPI)
	http.HandleFunc("/version", versionAPI)
	http.HandleFunc("/api/livez", livez)
	http.HandleFunc("/livez", livez)
	http.HandleFunc("/api/readyz", readyz)
	http.HandleFunc("/readyz", readyz)
	http.HandleFunc("/api/status", status)
	http.HandleFunc("/status", status)
	http.HandleFunc("/favicon.ico", faviconHandler)

	Logger.Info().Str("hostname", hostname).Msg("Starting server counter " + version.Version + ", ser ...")

	time.Sleep(time.Duration(slowStart) * time.Second)

	Logger.Info().Str("hostname", hostname).Msg("Server counter " + version.Version + " started on 0.0.0.0:" + port + ", see http://127.0.0.1:" + port)
	err = http.ListenAndServe("0.0.0.0:"+port, nil)
	if err != nil {
		Logger.Fatal().Str("hostname", hostname).Msg(err.Error())
	}
}
