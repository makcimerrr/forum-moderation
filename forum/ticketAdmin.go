package forum

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func AllTickets(w http.ResponseWriter, r *http.Request) {

	// Vérifiez la validité de la session
	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}

	var username string
	// Récupérer le nom d'utilisateur à partir du cookie "username"
	usernameCookie, err := r.Cookie("username")

	if err != nil {
		fmt.Println(err)
	}else {
		username = usernameCookie.Value
	} 
	
	var admin bool
	var isadmin string

	adminCookie, err := r.Cookie("access_level")

	if err != nil {
		fmt.Println(err)
	}else {
		isadmin = adminCookie.Value
	}

	if isadmin == "admin" {
			admin = true
		} else {
			admin = false 
	
		}


	var discussions []Discussion

	category := r.URL.Query().Get(`category`)

	if category == "" {
		// Récupérer toutes les discussions à partir de la base de données
		discussions, err = GetAllDiscussionsFromDB()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	} else {
		discussions, err = GetDiscussionsFromDBByCategories(category)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}


	// Récupérer les catégories pour chaque discussion
	for i, discussion := range discussions {
		category, err := GetCategoryForDiscussionFromDB(discussion.ID)
		if err == nil {
			discussions[i].Category = category
		}
	}

	// Récupérer les catégories uniques
	categories := GetUniqueCategoriesFromDiscussions(discussions)
 
	// Pour chaque discussion, vérifiez si l'utilisateur l'a aimée
	for i := range discussions {
		liked, err := CheckIfUserLikedDiscussion(username, discussions[i].ID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		discussions[i].Liked = liked

		// Pour chaque discussion, vérifiez si l'utilisateur l'a pas aimée
		disliked, err := CheckIfUserDislikedDiscussion(username, discussions[i].ID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		discussions[i].Disliked = disliked

		// Pour chaque discussion, vérifiez si l'utilisateur l'a aimée
		numberLike, err := CheckNumberOfLikesForDiscussion(discussions[i].ID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		discussions[i].NumberLike = numberLike

		numberDislike, err := CheckNumberOfDislikesForDiscussion(discussions[i].ID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		discussions[i].NumberDislike = numberDislike
	}


	var tickets []Ticket

	probleme := r.URL.Query().Get(`probleme`)

	if probleme == "" {
		// Récupérer toutes les discussions à partir de la base de données
		tickets, err = GetAllTicketsFromDB()
		if err != nil {
			http.Error(w, "Internal Server Error 1", http.StatusInternalServerError)
			return
		}
	} else {
		tickets, err = GetTicketsFromDBByCategories(probleme)
		if err != nil {
			http.Error(w, "Internal Server Error 2" , http.StatusInternalServerError)
			return
		}
	}


	// Récupérer les catégories pour chaque discussion
	for i, ticket := range tickets {
		probleme, err := GetCategoryForTicketFromDB(ticket.ID)
		if err == nil {
			tickets[i].Probleme = probleme
		}
	}

	// Récupérer les catégories uniques
	problemes := GetUniqueCategoriesFromTickets(tickets)
 

// Créer une structure de données pour passer les informations au modèle
	data := struct {
		Username    string
		Admin       bool

		Discussions []Discussion
		Categories  []string

		Tickets []Ticket
		Problemes  []string
	}{
		Username:    username,
		Admin:       admin,
		Discussions: discussions,
		Tickets: tickets,
		Categories:  categories,
		Problemes:  problemes,
	}

	tmpl := template.Must(template.ParseFiles("templates/tickets.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func CreateTicketModo(w http.ResponseWriter, r *http.Request) {


	data := struct {
		FormErrors []string
	}{
		FormErrors: nil, // Initialisez-le à nil ou avec des valeurs par défaut si nécessaire

	}

	// Vérifiez la validité de la sessionf

	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}

	 // Si une erreur est présente dans l'URL, récupérez-la
	 errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		data.FormErrors = strings.Split(errorParam, ",")
	}

	 
	if r.Method == http.MethodPost {
		sujet := r.FormValue("sujet")
		message := r.FormValue("message")
		probleme := r.FormValue("probleme") // Récupérez la catégorie du formulaire

		//const maxFileSize = 20 * (1 << 20); // 20 megabytes in bytes
		const maxFileSize = 1 << 1 // test 

		file, fileHeader, err := r.FormFile("file")
		var imageBuffer bytes.Buffer

		if file != nil {
			fmt.Println("test nil ")

			if err != nil {
				fmt.Println("Erreur lors de la récupération du fichier :", err)
				http.Error(w, "Internal Server Error: Erreur lors de la récupération du fichier", http.StatusInternalServerError)
				return
			}
			fmt.Println("test apres nil 11 ")

			defer file.Close()

			fmt.Println("test apres nil 222 ")


			// Vérification de la taille du fichier
			if err != nil || fileHeader.Size > maxFileSize {
				data.FormErrors = append(data.FormErrors, "Erreur lors de la récupération ou taille de fichier trop volumineuse.")
				http.Redirect(w, r, "/create_ticket?error="+url.QueryEscape(strings.Join(data.FormErrors, ",")), http.StatusSeeOther)
				return
			}

			fmt.Println("test apres nil3333 ")



			_, err = io.Copy(&imageBuffer, file)
			if err != nil {
				http.Error(w, "Internal Server Error: Lecture du fichier", http.StatusInternalServerError)
				fmt.Println("Erreur lors de la lecture du fichier :", err)
				return
			}

		}

		// Obtenez le nom d'utilisateur à partir du cookie "username"
		usernameCookie, err := r.Cookie("username")
		if err != nil {
			http.Redirect(w, r, "/log_in", http.StatusSeeOther)
			return
		}
		username := usernameCookie.Value

		// Ouverture de la connexion à la base de données
		db, err := sql.Open("sqlite", "database/data.db")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Insérez le nouveau ticket dans la base de données, y compris la catégorie
		stmt, err := db.Prepare("INSERT INTO ticket_modo (username, sujet, message, raison, image) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			http.Error(w, "Internal Server Error: Préparation de la requête SQL", http.StatusInternalServerError)
			fmt.Println("Erreur lors de la préparation de la requête SQL :", err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(username, sujet, message, probleme, imageBuffer.Bytes())
		if err != nil {
			http.Error(w, "Internal Server Error: Insertion dans la base de données", http.StatusInternalServerError)
			fmt.Println("Erreur lors de l'insertion dans la base de données :", err)
			return
		}

		// Récupérez l'ID de la discussion nouvellement créée
		var ticketID int
		err = db.QueryRow("SELECT last_insert_rowid()").Scan(&ticketID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Redirigez l'utilisateur vers la page de la discussion
		http.Redirect(w, r, "/ticket/"+strconv.Itoa(ticketID), http.StatusSeeOther)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/write_ticket_modo.html"))
	tmpl.Execute(w, data)
	
}

func CreateTicket(w http.ResponseWriter, r *http.Request) {


	data := struct {
		FormErrors []string
	}{
		FormErrors: nil, // Initialisez-le à nil ou avec des valeurs par défaut si nécessaire
	}

	// Vérifiez la validité de la session

	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}

	 // Si une erreur est présente dans l'URL, récupérez-la
	 errorParam := r.URL.Query().Get("error")
	if errorParam != "" {
		data.FormErrors = strings.Split(errorParam, ",")
	}

	 
	if r.Method == http.MethodPost {
		sujet := r.FormValue("sujet")
		message := r.FormValue("message")
		probleme := r.FormValue("probleme") // Récupérez la catégorie du formulaire

		//const maxFileSize = 20 * (1 << 20); // 20 megabytes in bytes
		const maxFileSize = 1 << 1 // test 

		file, fileHeader, err := r.FormFile("file")
		var imageBuffer bytes.Buffer

		if file != nil {
			fmt.Println("test nil ")

			if err != nil {
				fmt.Println("Erreur lors de la récupération du fichier :", err)
				http.Error(w, "Internal Server Error: Erreur lors de la récupération du fichier", http.StatusInternalServerError)
				return
			}
			fmt.Println("test apres nil 11 ")

			defer file.Close()

			fmt.Println("test apres nil 222 ")


			// Vérification de la taille du fichier
			if err != nil || fileHeader.Size > maxFileSize {
				data.FormErrors = append(data.FormErrors, "Erreur lors de la récupération ou taille de fichier trop volumineuse.")
				http.Redirect(w, r, "/create_ticket?error="+url.QueryEscape(strings.Join(data.FormErrors, ",")), http.StatusSeeOther)
				return
			}

			fmt.Println("test apres nil3333 ")



			_, err = io.Copy(&imageBuffer, file)
			if err != nil {
				http.Error(w, "Internal Server Error: Lecture du fichier", http.StatusInternalServerError)
				fmt.Println("Erreur lors de la lecture du fichier :", err)
				return
			}

		}

		// Obtenez le nom d'utilisateur à partir du cookie "username"
		usernameCookie, err := r.Cookie("username")
		if err != nil {
			http.Redirect(w, r, "/log_in", http.StatusSeeOther)
			return
		}
		username := usernameCookie.Value

		// Ouverture de la connexion à la base de données
		db, err := sql.Open("sqlite", "database/data.db")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Insérez le nouveau ticket dans la base de données, y compris la catégorie
		stmt, err := db.Prepare("INSERT INTO ticket_modo (username, sujet, message, raison, image) VALUES (?, ?, ?, ?, ?)")
		if err != nil {
			http.Error(w, "Internal Server Error: Préparation de la requête SQL", http.StatusInternalServerError)
			fmt.Println("Erreur lors de la préparation de la requête SQL :", err)
			return
		}
		defer stmt.Close()

		_, err = stmt.Exec(username, sujet, message, probleme, imageBuffer.Bytes())
		if err != nil {
			http.Error(w, "Internal Server Error: Insertion dans la base de données", http.StatusInternalServerError)
			fmt.Println("Erreur lors de l'insertion dans la base de données :", err)
			return
		}

		// Récupérez l'ID de la discussion nouvellement créée
		var ticketID int
		err = db.QueryRow("SELECT last_insert_rowid()").Scan(&ticketID)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Redirigez l'utilisateur vers la page de la discussion
		http.Redirect(w, r, "/ticket/"+strconv.Itoa(ticketID), http.StatusSeeOther)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/write_ticket.html"))
	tmpl.Execute(w, data)
	
}

func ShowTicket(w http.ResponseWriter, r *http.Request) {
	// Vérifiez la validité de la session
	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}
	// Récupérez l'ID de la discussion à partir de l'URL
	ticketID := r.URL.Path[len("/ticket/"):]
	// Convertissez l'ID de la discussion en un entier
	ticketIDInt, err := strconv.Atoi(ticketID)
	if err != nil {
		http.Error(w, "Invalid discussion ID", http.StatusBadRequest)
		return
	}
	// Ouverture de la connexion à la base de données
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Effectuez une requête SQL pour récupérer les détails de la discussion en fonction de l'ID
	var username, sujet, message string
	var imageData []byte // Utilisez un slice de bytes pour stocker les données binaires de l'image
	err = db.QueryRow("SELECT username, sujet, message, image FROM ticket_modo WHERE id = ?", ticketIDInt).Scan(&username, &sujet, &message, &imageData)
	if err != nil {
		http.Error(w, "Ticket not found", http.StatusNotFound)
		fmt.Println(ticketID, ticketIDInt)
		fmt.Println(err)
		return
	}

	// Effectuez une autre requête SQL pour récupérer les commentaires associés à cette discussion
	rows, err := db.Query("SELECT id, username, message FROM responses WHERE ticket_id = ?", ticketIDInt)
	if err != nil {
		http.Error(w, "Error fetching responses 1", http.StatusInternalServerError)
		fmt.Println(ticketID)
		fmt.Println(err)
		return
	}
	defer rows.Close()

	// Exécuter la requête SQL
	var userID int
	err = db.QueryRow("SELECT id FROM account_user WHERE username = ?", username).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}


	var modo bool
	var ismodo string

	modoCookie, err := r.Cookie("access_level")

	if err != nil {
		fmt.Println(err)
	}else {
		ismodo = modoCookie.Value
	}

	if ismodo == "modo" || ismodo == "admin"{
			modo = true
		} else {
			modo = false 
	
	}
	


	// Créez une structure de données pour stocker les détails de la discussion et les commentaires
	data := struct {
		Modo bool
		Username string
		Sujet    string
		ID       int
		Message  string
		Image    string
		Raison   *string
		UserID int
		Responses []struct {
			ID int
			Username string
			Message  string
		}
	}{
		Modo: modo,
		Username: username,
		Sujet:    sujet,
		Image:    base64.StdEncoding.EncodeToString(imageData),
		ID:       ticketIDInt,
		Message:  message,
		UserID: userID,
	}

	var raison sql.NullString
	err = db.QueryRow("SELECT COALESCE(raison, '') FROM ticket_modo WHERE id = ?", ticketIDInt).Scan(&raison)
	if err != nil {
		http.Error(w, "Raison not found", http.StatusNotFound)
		return
	}

	var raisonValue string
	if raison.Valid {
		raisonValue = raison.String
	}

	data.Raison = &raisonValue // Permet l'affiche du filtre {{ .Raison}}

	// Parcourez les commentaires et ajoutez-les à la structure de données
	for rows.Next() {
		var response struct {
			ID int
			Username string
			Message  string
		}
		fmt.Println(response.ID)

		if err := rows.Scan(&response.ID, &response.Username, &response.Message); err != nil {
			http.Error(w, "Error scanning responses", http.StatusInternalServerError)
			return
		}
		data.Responses = append(data.Responses, response)
	}

	// Affichez les détails de la discussion et les commentaires dans un modèle HTML
	tmpl := template.Must(template.ParseFiles("templates/show_ticket.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
}

func GetAllTicketsFromDB() ([]Ticket, error) {
	// Ouvrez la connexion à la base de données
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	// Exécutez une requête SQL pour récupérer toutes les discussions
	rows, err := db.Query("SELECT id, username, sujet, message FROM ticket_modo")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Créez une slice pour stocker les discussions
	var tickets []Ticket
	// Parcourez les résultats et stockez-les dans la slice
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(&ticket.ID, &ticket.Username, &ticket.Sujet, &ticket.Message)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	return tickets, nil
}


func GetTicketsFromDBByCategories(probleme string) ([]Ticket, error) {
	// Ouvrez la connexion à la base de données
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	// Exécutez une requête SQL pour récupérer toutes les discussions
	// Créez une requête préparée
	stmt, err := db.Prepare("SELECT id, username, sujet, message FROM ticket_modo WHERE raison = $1")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// Exécutez la requête préparée avec la variable probleme
	rows, err := stmt.Query(probleme)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	// Créez une slice pour stocker les discussions
	var tickets []Ticket
	// Parcourez les résultats et stockez-les dans la slice
	for rows.Next() {
		var ticket Ticket
		err := rows.Scan(&ticket.ID, &ticket.Username, &ticket.Sujet, &ticket.Message)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	return tickets, nil
}

func AddResponse(w http.ResponseWriter, r *http.Request) {
	// Vérifiez la validité de la session
	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		// Récupérez l'ID de la discussion à partir de l'URL
		ticketID := r.URL.Path[len("/add_response/"):]
		// Convertissez l'ID de la discussion en un entier
		ticketIDInt, err := strconv.Atoi(ticketID)
		if err != nil {
			http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
			return
		}

		message := r.FormValue("message")
		// Obtenez le nom d'utilisateur à partir du cookie "username"
		usernameCookie, err := r.Cookie("username")
		if err != nil {
			// Gérer l'erreur ici, par exemple, en redirigeant l'utilisateur vers une page de connexion s'il n'est pas connecté.
			http.Redirect(w, r, "/log_in", http.StatusSeeOther)
			return
		}
		username := usernameCookie.Value

		// Ouverture de la connexion à la base de données
		db, err := sql.Open("sqlite", "database/data.db")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Insérez le nouveau message dans la base de données en incluant l'ID de discussion
		_, err = db.Exec("INSERT INTO responses (ticket_id, username, message) VALUES (?, ?, ?)", ticketIDInt, username, message)
		if err != nil {
			log.Printf("Erreur lors de l'insertion du message : %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Redirigez l'utilisateur vers la page de discussion
		http.Redirect(w, r, fmt.Sprintf("/ticket/%d", ticketIDInt), http.StatusSeeOther)
		return
	}
	

	// Affichez la page pour écrire une discussion (write_discussion.html)
	tmpl := template.Must(template.ParseFiles("templates/show_ticket.html"))
	tmpl.Execute(w, nil)
}

func GetCategoryForTicketFromDB(ticketID int) (string, error) {
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		return "", err
	}
	defer db.Close()

	var probleme string
	err = db.QueryRow("SELECT COALESCE(raison, '') FROM ticket_modo WHERE id = ?", ticketID).Scan(&probleme)
	if err != nil {
		return "", err
	}

	return probleme, nil
}

func GetUniqueCategoriesFromTickets(tickets []Ticket) []string {
	uniqueProblemes := make(map[string]struct{})

	// Parcourez les discussions et ajoutez chaque catégorie à la carte uniqueProblemes
	for _, ticket := range tickets {
		uniqueProblemes[ticket.Probleme] = struct{}{}
	}

	// Créez une slice pour stocker les catégories uniques
	problemes := make([]string, 0, len(uniqueProblemes))

	// Parcourez la carte uniqueProblemes pour extraire les catégories uniques
	for probleme := range uniqueProblemes {
		problemes = append(problemes, probleme)
	}

	return problemes
}
func PromoteOrDemote(w http.ResponseWriter, req *http.Request) {
	var successMessage []string
	// Ouvrir la base de données SQLite
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		http.Error(w, "Erreur lors de l'ouverture de la base de données", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Récupérer les données du formulaire
	err = req.ParseForm()
	if err != nil {
		log.Printf("Erreur lors de la lecture des données du formulaire : %v", err)
		http.Error(w, "Erreur lors de la lecture des données du formulaire", http.StatusBadRequest)
		return
	}

	// Récupérer la valeur de l'option
	action := req.FormValue("itemType")

	// Récupérer l'ID de l'utilisateur
	userID := req.FormValue("userID")

	// Valider que l'ID est présent
	if userID == "" {
		log.Printf("ID de l'utilisateur manquant")
		http.Error(w, "ID de l'utilisateur manquant", http.StatusBadRequest)
		return
	}

	// Récupérer l'access_level actuel de l'utilisateur
	var accessLevel string
	err = db.QueryRow("SELECT access_level FROM account_user WHERE id = ?", userID).Scan(&accessLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Utilisateur non trouvé : %v", err)
			http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		} else {
			log.Printf("Erreur lors de la récupération de l'access_level : %v", err)
			http.Error(w, "Erreur lors de la récupération de l'access_level", http.StatusInternalServerError)
		}
		return
	}

	// Mettre à jour l'access_level en fonction de l'action
	switch action {
	case "promote":
		if accessLevel == "admin"{
			successMessage = append(successMessage, "L'utilisateur est déjà administrateur")
			break
		}else if accessLevel == "user" || accessLevel == "NULL" {
			accessLevel = "modo"
		} else if accessLevel == "modo" {
			accessLevel = "admin"
		}
		successMessage = append(successMessage, "L'utilisateur a été promu avec succès. Rôle actuel ", accessLevel)
	case "demote":
		if accessLevel == "user" || accessLevel == "NULL"{
			successMessage = append(successMessage, "L'utilisateur est déjà user")
			break
		}else if accessLevel == "admin" {
			accessLevel = "modo"
		} else if accessLevel == "modo" {
			accessLevel = "user"
		}
		successMessage = append(successMessage, "L'utilisateur a été rétrogradé avec succès. Rôle actuel ", accessLevel)
	default:
		http.Error(w, "Action invalide", http.StatusBadRequest)
		return
	}

	// Mettre à jour l'access_level dans la base de données
	_, err = db.Exec("UPDATE account_user SET access_level = ? WHERE id = ?", accessLevel, userID)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour de l'access_level : %v", err)
		http.Error(w, "Erreur lors de la mise à jour de l'access_level", http.StatusInternalServerError)
		return
	}

	
	http.Redirect(w, req, "/home?error="+url.QueryEscape(strings.Join(successMessage, "; ")), http.StatusSeeOther)
}