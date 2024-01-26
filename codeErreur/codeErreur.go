package codeerreur

import (
	"fmt"
	"log"
	"net/http"
	"text/template"
)

func CodeErreur(w http.ResponseWriter, r *http.Request, status int, message string) {
	colorRed := "\033[31m" // Mise en place d'une couleur pour les erreurs
	// Mise en place de condition pour les différents types d'erreurs
	if status == 404 {
		w.WriteHeader(http.StatusNotFound)
		custTemplate, err := template.ParseFiles("./templates/404.html")
		if err != nil {
			log.Println("Error parsing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = custTemplate.Execute(w, nil)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		fmt.Println(string(colorRed), "[SERVER_ALERT] - 404 : Error URL...", message) // Message qui sera afficher sur le terminal avec une précision de l'erreur
	}
	if status == 400 {
		w.WriteHeader(http.StatusNotFound)
		custTemplate, err := template.ParseFiles("./templates/400.html")
		if err != nil {
			log.Println("Error parsing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = custTemplate.Execute(w, nil)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		fmt.Println(string(colorRed), "[SERVER_ALERT] - 400 : Invalid endpoint", message) // Message qui sera afficher sur le terminal avec une précision de l'erreur
	}
	if status == 500 {
		w.WriteHeader(http.StatusNotFound)
		custTemplate, err := template.ParseFiles("./templates/500.html")
		if err != nil {
			log.Println("Error parsing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = custTemplate.Execute(w, nil)
		if err != nil {
			log.Println("Error executing template:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		fmt.Println(string(colorRed), "[SERVER_ALERT] - 500 : Internal server error", message) // Message qui sera afficher sur le terminal avec une précision de l'erreur
	}
}
