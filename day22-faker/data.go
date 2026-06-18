package main

// Petits viviers de données françaises. Rien d'exhaustif : juste assez de
// variété pour qu'un jeu de test ne ressemble pas à dix fois la même ligne.
// On reste sur des valeurs plausibles plutôt que farfelues — l'idée est de
// remplir une base de démo, pas d'écrire un annuaire.

var prenoms = []string{
	"Marie", "Jean", "Pierre", "Sophie", "Luc", "Camille", "Julien", "Emma",
	"Lucas", "Léa", "Hugo", "Chloé", "Nathan", "Manon", "Thomas", "Sarah",
	"Antoine", "Inès", "Maxime", "Clara", "Paul", "Alice", "Louis", "Jade",
	"Gabriel", "Louise", "Adam", "Mila", "Raphaël", "Anna", "Théo", "Zoé",
}

var noms = []string{
	"Martin", "Bernard", "Dubois", "Thomas", "Robert", "Petit", "Durand",
	"Leroy", "Moreau", "Simon", "Laurent", "Lefebvre", "Michel", "Garcia",
	"David", "Bertrand", "Roux", "Vincent", "Fournier", "Morel", "Girard",
	"André", "Mercier", "Dupont", "Lambert", "Bonnet", "François", "Martinez",
	"Legrand", "Garnier", "Faure", "Rousseau",
}

// Villes avec leur code postal : les deux vont ensemble, on les tire en bloc
// pour qu'une adresse parisienne n'hérite pas d'un CP marseillais.
var villes = []struct{ nom, cp string }{
	{"Paris", "75001"}, {"Lyon", "69001"}, {"Marseille", "13001"},
	{"Toulouse", "31000"}, {"Nice", "06000"}, {"Nantes", "44000"},
	{"Strasbourg", "67000"}, {"Montpellier", "34000"}, {"Bordeaux", "33000"},
	{"Lille", "59000"}, {"Rennes", "35000"}, {"Reims", "51100"},
	{"Grenoble", "38000"}, {"Dijon", "21000"}, {"Angers", "49000"},
	{"Nîmes", "30000"}, {"Toulon", "83000"}, {"Le Havre", "76600"},
}

var rues = []string{
	"rue de la République", "avenue Jean Jaurès", "boulevard Voltaire",
	"rue Victor Hugo", "rue de Paris", "place de la Mairie", "rue des Lilas",
	"rue Gambetta", "impasse des Roses", "rue du Moulin", "chemin des Vignes",
	"allée des Tilleuls", "rue Pasteur", "avenue de la Gare",
}

var entreprises = []string{
	"Orange", "Capgemini", "Decathlon", "Renault", "Carrefour", "Michelin",
	"BNP Paribas", "Dassault Systèmes", "Société Générale", "Thales",
	"Air France", "Veolia", "Sodexo", "Atos",
}

var domaines = []string{
	"gmail.com", "outlook.fr", "yahoo.fr", "orange.fr", "free.fr", "laposte.net",
}
