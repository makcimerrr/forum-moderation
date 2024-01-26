package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"forum/facebook"
	"forum/forum"
	"forum/github"

	"github.com/joho/godotenv"
)

var (
	requestsMap = make(map[string]int)
	lastReset   = time.Now() // Stocke le temps de la dernière réinitialisation
	mutex       = &sync.Mutex{}
	lastPage     = ""         // Stocke l'URL de la dernière page accédée
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load("data.env"); err != nil {
		log.Fatal("No .env file found")
	}
}

func main() {
	if err := godotenv.Load("data.env"); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Middleware pour le rate limiting
	rateLimitMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Vérifiez si une minute s'est écoulée depuis la dernière réinitialisation
		if time.Since(lastReset) > time.Minute {
			// Si oui, réinitialisez le compteur de requêtes
			mutex.Lock()
			requestsMap = make(map[string]int)
			lastReset = time.Now()
			mutex.Unlock()
			
		}
			// Vérifiez si l'adresse demandée est "/static/" et bloquez la requête si c'est le cas
		if r.URL.Path == "/static/" {
			
			log.Printf("Rate limit for acces to /static/")
			http.Error(w, "Access to /static/ is not allowed", http.StatusForbidden)
			return
		}

		// Vérifiez si l'URL change (changement de page)
		if r.URL.Path != lastPage {
			// Si oui, réinitialisez le compteur de requêtes
			mutex.Lock()
			requestsMap = make(map[string]int)
			lastPage = r.URL.Path
			lastReset = time.Now()
			mutex.Unlock()
		}

			// Utilisez l'adresse IP de l'utilisateur comme clé de suivi
			key := r.RemoteAddr

			// Verrouillez la carte pour éviter les accès concurrents
			mutex.Lock()
			defer mutex.Unlock()

			// Incrémente le compteur de requêtes pour l'adresse IP actuelle
			requestsMap[key]++

			// Si le nombre de requêtes dépasse une limite définie, renvoyer une réponse d'erreur
			if requestsMap[key] > 3 { // Par exemple, limitez à 10 requêtes par minute
				log.Printf("Rate limit exceeded for %s. Requests: %d", key, requestsMap[key])
				http.Error(w, "Rate limit exceeded. Wait 1 minute.", http.StatusTooManyRequests)
				return
			}

			// Laissez passer la requête vers le gestionnaire suivant
			next(w, r)
		}
	}

	// Login route
	http.HandleFunc("/login/github/", rateLimitMiddleware(github.GithubLoginHandler))

	// Github callback
	http.HandleFunc("/login/github/callback", github.GithubCallbackHandler)

	// Route where the authenticated user is redirected to
	http.HandleFunc("/loggedin", func(w http.ResponseWriter, r *http.Request) {
		github.LoggedinHandler(w, r, "")
	})

	http.HandleFunc("/login/facebook", facebook.HandleFacebookLogin)
	http.HandleFunc("/oauth2callback", facebook.HandleFacebookCallback)

	http.HandleFunc("/", forum.Home)
	http.HandleFunc("/404", forum.HandleNotFound)
	http.HandleFunc("/500", forum.HandleServerError)
	http.HandleFunc("/400", forum.HandleBadRequest)
	http.HandleFunc("/logorsign", forum.Logorsign)
	http.HandleFunc("/log_in", forum.Log_in)
	http.HandleFunc("/sign_up", forum.Sign_up)
	http.HandleFunc("/logout", forum.Logout)
	http.HandleFunc("/home", forum.Home)
	http.HandleFunc("/tickets", forum.AllTickets)
	http.HandleFunc("/create_discussion", forum.CreateDiscussion)
	http.HandleFunc("/create_ticket", forum.CreateTicket)
	http.HandleFunc("/create_ticket_modo", forum.CreateTicketModo)
	http.HandleFunc("/discussion/", forum.ShowDiscussion)
	http.HandleFunc("/ticket/", forum.ShowTicket)
	http.HandleFunc("/DeleteItem", forum.DeleteItem)
	http.HandleFunc("/PromoteOrDemote", forum.PromoteOrDemote)
	http.HandleFunc("/add_message/", forum.AddMessage)
	http.HandleFunc("/add_response/", forum.AddResponse)
	http.HandleFunc("/like/", forum.LikeDiscussion)
	http.HandleFunc("/dislike/", forum.DislikeDiscussion)

	// Définir le dossier "static" comme dossier de fichiers statiques
	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	fmt.Println("Voici le lien pour ouvrir la page web http://localhost:3000/")
	http.ListenAndServe(":3000", nil)
}