package main

import (
	"flag"
	"log"

   	"context"
    "google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const (
	calendarName        = "Birthdays from Contacts"
	calendarDescription = "Calendar containing birthdays imported from Google Contacts"
)

func main() {
	accessToken := flag.String("token", "", "Access token for Google APIs")
	flag.Parse()

	if *accessToken == "" {
		log.Fatal("Access token is required")
	}

    ctx := context.Background()

	// Initialize People API client
	peopleService, err := people.NewService(ctx, option.WithCredentialsJSON([]byte(`{"access_token":"`+*accessToken+`"}`)))
	if err != nil {
		log.Fatalf("Unable to create People service: %v", err)
	}

	// Fetch contacts with birthdays
	connections, err := peopleService.People.Connections.List("people/me").
		PersonFields("names,birthdays").
		PageSize(1000).
		Do()
	if err != nil {
		log.Fatalf("Unable to retrieve contacts: %v", err)
	}

	// Process each contact
	for _, person := range connections.Connections {
		if len(person.Birthdays) == 0 {
			continue
		}

		name := "Unknown"
		if len(person.Names) > 0 {
			name = person.Names[0].DisplayName
		}

		birthday := person.Birthdays[0].Date
		if birthday == nil {
			continue
		}

		log.Printf("Found birthdayl for %s - on %s", name, birthday)
	}
}
