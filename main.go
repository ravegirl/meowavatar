package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"meowavatar/cache"
	"meowavatar/ratelimit"
	"meowavatar/services/discord"
	"meowavatar/services/github"
	"meowavatar/services/reddit"
	"meowavatar/services/steam"
	"meowavatar/services/telegram"
	"meowavatar/services/twitch"
	"meowavatar/services/twitter"
)

func main() {
	_ = godotenv.Load()

	rdb := cache.NewRedis()
	rl := ratelimit.New(rdb)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("GET /{social}/{identifier}", func(w http.ResponseWriter, r *http.Request) {
		social := r.PathValue("social")
		identifier := r.PathValue("identifier")

		var handler func(string) ([]byte, string, error)

		switch social {
		case "github":
			handler = github.Fetch
		case "discord":
			handler = discord.Fetch
		case "reddit":
			handler = reddit.Fetch
		case "twitter", "x":
			handler = twitter.Fetch
		case "twitch":
			handler = twitch.Fetch
		case "steam":
			handler = steam.Fetch
		case "telegram":
			handler = telegram.Fetch
		default:
			http.Error(w, "unsupported social: "+social, http.StatusNotFound)
			return
		}

		cacheKey := social + ":" + identifier
		rdb.ServeImage(w, r, cacheKey, handler, identifier)
	})

	addr := ":" + port()
	log.Printf("meowavatar listening on %s", addr)
	if err := http.ListenAndServe(addr, rl.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return "8080"
}
