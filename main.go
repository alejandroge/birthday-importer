package main

import (
	"flag"
	"fmt"
	"log"

	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const (
	calendarName        = "Birthdays from Contacts"
	calendarDescription = "Calendar containing birthdays imported from Google Contacts"
)

func formatDate(birthday *people.Date) string {
	// If Year is 0, use a default year (e.g., 2000)
	year := birthday.Year
	if year == 0 {
		year = 1970
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, birthday.Month, birthday.Day)
}

func main() {
	accessToken := flag.String("token", "", "Access token for Google APIs")
	flag.Parse()

	if *accessToken == "" {
		log.Fatal("Access token is required")
	}

	ctx := context.Background()

	// Create an OAuth2 token source with the access token
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *accessToken})

	// Initialize People API client
	peopleService, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Fatalf("Unable to create People service: %v", err)
	}

	// Initialize Calendar API client
	calendarService, err := calendar.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		log.Fatalf("Unable to create Calendar service: %v", err)
	}
	// Create a new calendar for birthdays. Create a new calendar for every import. To avoid dealing with duplicates handling.
	log.Println("Creating a new calendar for birthdays.")
	cal, err := calendarService.Calendars.Insert(&calendar.Calendar{
		Summary:     calendarName,
		Description: calendarDescription,
		TimeZone:    "UTC",
	}).Do()
	if err != nil {
		log.Fatalf("Unable to create calendar: %v", err)
	}
	log.Printf("Created calendar: %s", cal.Summary)

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

		log.Printf("Found birthday for %s - on %d/%d/%d", name, birthday.Day, birthday.Month, birthday.Year)

		// Create a new event for the birthday, in the newly created calendar. Make it a recurring and all day event. Not all of the contacts have a year of birth, so we only include the year for the ones that have it.
		event := &calendar.Event{
			Summary:     name,
			Description: "Birthday",
			Start: &calendar.EventDateTime{
				Date:     formatDate(birthday),
				TimeZone: "UTC",
			},
			End: &calendar.EventDateTime{
				Date:     formatDate(birthday),
				TimeZone: "UTC",
			},
			Recurrence: []string{
				"RRULE:FREQ=YEARLY;COUNT=100",
			},
		}

		// Insert the event into the calendar
		_, err := calendarService.Events.Insert(cal.Id, event).Do()
		if err != nil {
			log.Printf("Unable to create event for %s: %v", name, err)
			continue
		}
		log.Printf("Created event for %s", name)
	}
}
