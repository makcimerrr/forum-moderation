package forum

type Discussion struct {
	ID            int
	Title         string
	Message       string
	Username      string
	Category      string
	Liked         bool // Champ pour indiquer si l'utilisateur a aimé cette discussion
	Disliked      bool
	NumberLike    int
	NumberDislike int
}

// Ajoutez cette structure pour représenter un message
type Comment struct {
	Idmessage int
	Username string
	Message  string
}

type Ticket struct {
	ID            int
	Sujet         string
	Message       string
	Username      string
	Probleme      string
	Liked         bool 
	Disliked      bool
	NumberLike    int
	NumberDislike int
}

// Ajoutez cette structure pour représenter une réponse
type Response struct {
	Idmessage int
	Username string
	Message  string
}
