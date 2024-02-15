package forum

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
	"strconv"

	_ "modernc.org/sqlite"
)

func codeErreur(w http.ResponseWriter, r *http.Request, url string, route string, html string) {
	if url != route {
		http.Redirect(w, r, "/404", http.StatusFound)
	}
	_, err := template.ParseFiles(html)
	if err != nil {
		http.Redirect(w, r, "/500", http.StatusFound)
	}
}

func HandleNotFound(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/404.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func HandleServerError(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/500.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func HandleBadRequest(w http.ResponseWriter, r *http.Request) {
	template.Must(template.ParseFiles("templates/400.html"))
}

func Logorsign(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/logorsign.html"))
	err := tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func Sign_up(w http.ResponseWriter, r *http.Request) {
	var formError []string

	if r.Method == http.MethodPost {
		// Récupération des informations du formulaire
		username := r.FormValue("username")
		email := r.FormValue("email")
		password := r.FormValue("password")

		hashpass := hash(password)

		// Ouverture de la connexion à la base de données
		db, err := sql.Open("sqlite", "database/data.db")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer db.Close()

		// Création de la table s'il n'existe pas
		createTable := `
           CREATE TABLE IF NOT EXISTS account_user (
               id INTEGER PRIMARY KEY AUTOINCREMENT,
               username TEXT,
               email TEXT,
               mot_de_passe INT
           )
       `
		_, err = db.Exec(createTable)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Vérification si le nom d'utilisateur est déjà utilisé
		var existingUsername string
		err = db.QueryRow("SELECT username FROM account_user WHERE username = ?", username).Scan(&existingUsername)
		if err == nil {
			formError = append(formError, "This Username Is Already Use !! ")
		}

		// Vérification si l'e-mail est déjà utilisé
		var existingEmail string
		err = db.QueryRow("SELECT email FROM account_user WHERE email = ?", email).Scan(&existingEmail)
		if err == nil {
			formError = append(formError, "This Email Is Already Use !!")
		}
		lvaccess := "user"

		if formError == nil {
			insertUser := "INSERT INTO account_user (username, email, mot_de_passe, access_level) VALUES (?, ?, ?, ?)"
			_, err = db.Exec(insertUser, username, email, hashpass,lvaccess)

			admin := false

			err := CreateAndSetSessionCookies(w, username,admin)

		
			if err != nil {
				fmt.Println(err)
				return
			}

			// Rediriger l'utilisateur vers la page "/home" après l'enregistrement
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/sign_up.html"))
	data := struct {
		Errors []string
	}{
		Errors: formError,
	}
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func Log_in(w http.ResponseWriter, r *http.Request) {
	var formError []string
	errMsg := r.URL.Query().Get("error") // Récupérez le message d'erreur de la requête

	if r.Method == http.MethodPost {
		loginemail := r.FormValue("loginemail")
		loginpassword := r.FormValue("loginpassword")

		// Ouverture de la connexion à la base de données
		db, err := sql.Open("sqlite", "database/data.db")
		if err != nil {
			formError = append(formError, "Internal Server Error")
			http.Redirect(w, r, "/log_in?error="+url.QueryEscape(strings.Join(formError, "; ")), http.StatusSeeOther)
			return
		}
		defer db.Close()

		var trueemail string
		var truepassword uint32
		var username string
		var admin string
		var Admin = false
		err = db.QueryRow("SELECT username, email, mot_de_passe ,access_level FROM account_user WHERE email = ?", loginemail).Scan(&username, &trueemail, &truepassword,&admin)
		
		if err != nil {
			formError = append(formError, "Email Doesn't exist.")
		} else {
			hashloginpassword := hash(loginpassword)

			// Vérifier le mot de passe
			if hashloginpassword != truepassword {
				formError = append(formError, "Password Failed.")
			} else {
				if admin == "admin" {
					Admin = true
				}
				// L'utilisateur est connecté avec succès
				err := CreateAndSetSessionCookies(w, username, Admin)
				if err != nil {
					formError = append(formError, "Internal Server Error")
					http.Redirect(w, r, "/log_in?error="+url.QueryEscape(strings.Join(formError, "; ")), http.StatusSeeOther)
					return
				}

				// Redirigez l'utilisateur vers la page "/"
				http.Redirect(w, r, "/home", http.StatusSeeOther)
				return
			}
		}
	}

	tmpl := template.Must(template.ParseFiles("templates/login.html"))
	data := struct {
		Error  string
		Errors []string
	}{
		Error:  errMsg,
		Errors: formError,
	}
	err := tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func generateSessionToken() (string, error) {
	token := make([]byte, 32) // Crée un slice de bytes de 32 octets

	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(token), nil
}

func CreateAndSetSessionCookies(w http.ResponseWriter, username string, admin bool) error {
	// Générer un nouveau jeton de session uniquement si le nom d'utilisateur n'est pas vide
	if username == "" {
		return errors.New("Username is empty")
	}

	// Convertir le statut d'administrateur en une chaîne de caractères
	accessLevel := "user"
	if admin {
		accessLevel = "admin"
	}

	// Ouvrir une connexion à la base de données
	db, err := sql.Open("sqlite", "database/data.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Vérifier si l'utilisateur a déjà une entrée dans la base de données
	var existingSessionToken string
	err = db.QueryRow("SELECT sessionToken FROM token_user WHERE username = ?", username).Scan(&existingSessionToken)
	if err == sql.ErrNoRows {
		// Si l'utilisateur n'a pas encore d'entrée, générer un nouveau jeton de session
		sessionToken, err := generateSessionToken()
		if err != nil {
			return err
		}

		// Insérer la nouvelle entrée dans la base de données
		_, err = db.Exec("INSERT INTO token_user (username, sessionToken, access_level) VALUES (?, ?, NULL)", username, sessionToken)
		if err != nil {
			return err
		}

		// Créer un cookie contenant le nom d'utilisateur
		userCookie := http.Cookie{
			Name:     "username",
			Value:    username,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &userCookie)

		// Créer un cookie contenant le jeton de session
		sessionCookie := http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &sessionCookie)

		// Créer un cookie contenant le niveau d'accès
		accessLevelCookie := http.Cookie{
			Name:     "access_level",
			Value:    accessLevel,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &accessLevelCookie)

	} else if err == nil {
		// Si l'utilisateur a déjà une entrée, mettre à jour le jeton de session existant
		sessionToken, err := generateSessionToken()
		if err != nil {
			return err
		}

		// Mettre à jour le jeton de session et le niveau d'accès dans la base de données
		_, err = db.Exec("UPDATE token_user SET sessionToken = ?, access_level = ? WHERE username = ?", sessionToken, accessLevel, username)
		if err != nil {
			return err
		}
		
		// Créer un cookie contenant le nom d'utilisateur
		userCookie := http.Cookie{
			Name:     "username",
			Value:    username,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &userCookie)

		// Créer un cookie contenant le jeton de session
		sessionCookie := http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &sessionCookie)


		// Créer un cookie contenant le niveau d'accès
		accessLevelCookie := http.Cookie{
			Name:     "access_level",
			Value:    accessLevel,
			Path:     "/",
			HttpOnly: true,
		}
		http.SetCookie(w, &accessLevelCookie)

	} else {
		// En cas d'erreur différente de "pas de lignes", renvoyer l'erreur
		return err
	}

	return nil
}




func Logout(w http.ResponseWriter, r *http.Request) {
	var notification []string
	// Supprimer le cookie "username"
	usernameCookie, err := r.Cookie("username")
	if err == nil {
		usernameCookie.Expires = time.Now().AddDate(0, 0, -1) // Définir une date d'expiration dans le passé pour supprimer le cookie
		http.SetCookie(w, usernameCookie)
	}

	// Supprimer le cookie "session"
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		sessionCookie.Expires = time.Now().AddDate(0, 0, -1) // Définir une date d'expiration dans le passé pour supprimer le cookie
		http.SetCookie(w, sessionCookie)
		//s
	}

	// Créer un message de notification
	notification = append(notification, "Déconnexion réussie.")

	// Rediriger vers la page "/home" avec le message de notification
	http.Redirect(w, r, "/log_in?error="+url.QueryEscape(strings.Join(notification, "; ")), http.StatusSeeOther)
}

func Home(w http.ResponseWriter, r *http.Request) {

	// Vérifiez la validité de la session
	validSession, errMsg := isSessionValid(r)
	if !validSession {
		clearSessionCookies(w)
		// La session n'est pas valide, redirigez l'utilisateur vers la page de connexion ou effectuez d'autres actions
		http.Redirect(w, r, "/log_in?error="+url.QueryEscape(errMsg), http.StatusSeeOther)
		return
	}

	// Récupérer le nom d'utilisateur à partir du cookie "username"
	usernameCookie, err := r.Cookie("username")
	var username string
	var admin bool

	adminCookie, err := r.Cookie("access_level")
	if err == nil {
 	   adminStr := adminCookie.Value
 	   admin, _ = strconv.ParseBool(adminStr)
		if adminStr == "admin" {
			admin = true
		} else {
			admin = false 
	
		}
	}


	

	if err == nil {
		username = usernameCookie.Value
	}

	var category string
	var discussions []Discussion

	category = r.URL.Query().Get(`category`)

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


	// Créer une structure de données pour passer les informations au modèle
	data := struct {
		Username    string
		Admin       bool


		Discussions []Discussion
		Categories  []string
	}{
		Username:    username,
		Admin:       admin,
		Discussions: discussions,
		Categories:  categories,
	}

	tmpl := template.Must(template.ParseFiles("templates/home.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
    postIDStr := r.FormValue("itemID")
    itemType := r.FormValue("itemType")


	if itemType == "filterCreate" {
		
	
	
	}

    if itemType == "filterDelete" {
        err := deleteFilterFromDB(postIDStr)
        if err != nil {
            http.Error(w, "Error deleting filter", http.StatusInternalServerError)
            return
		}
    } else {

	

    postID, err := strconv.Atoi(postIDStr)
	

    if err != nil {
        http.Error(w, "Invalid post ID", http.StatusBadRequest)
        return
    }

     if itemType == "post" {
        err = deletePostFromDB(postID)
        if err != nil {
            http.Error(w, "Error deleting post", http.StatusInternalServerError)
            return
        }
    }

    if itemType == "comment" {
        err = deleteCommentFromDB(postID)
        if err != nil {
            http.Error(w, "Error deleting comment", http.StatusInternalServerError)
            return
        }
    }

	}
    // Rediriger l'utilisateur vers la page d'accueil ou une autre page appropriée
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func deletePostFromDB(postID int) error {
    // Ouvrir la connexion à la base de données
    db, err := sql.Open("sqlite", "database/data.db")
    if err != nil {
        return err
    }
    defer db.Close()

    // Exécuter la requête SQL pour supprimer le post de la base de données
    _, err = db.Exec("DELETE FROM discussion_user WHERE id = ?", postID)
    if err != nil {
        return err
    }

    // Si tout s'est bien passé, retourner nil (pas d'erreur)
    return nil
}

func deleteCommentFromDB(commentID int) error {
    // Ouvrir la connexion à la base de données
    db, err := sql.Open("sqlite", "database/data.db")
    if err != nil {
        return err
    }
    defer db.Close()

    // Exécuter la requête SQL pour supprimer le commentaire de la base de données
    _, err = db.Exec("DELETE FROM comments WHERE id = ?", commentID)
    if err != nil {
        return err
    }

    // Si tout s'est bien passé, retourner nil (pas d'erreur)
    return nil
}

func deleteFilterFromDB(filterID string) error {
    // Ouvrir la connexion à la base de données
    db, err := sql.Open("sqlite", "database/data.db")
    if err != nil {
        return err
    }
    defer db.Close()

    // Exécuter la requête SQL pour supprimer le filtre de la base de données
    _, err = db.Exec("DELETE FROM discussion_user WHERE filter = ?", filterID)
    if err != nil {
        return err
    }

    // Si tout s'est bien passé, retourner nil (pas d'erreur)
    return nil
}


