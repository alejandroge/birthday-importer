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
	// If Year is 0, use a default year (e.g., 2000). Cannot leave it empty using the API.
	year := birthday.Year
	if year == 0 {
		year = 1970
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, birthday.Month, birthday.Day)
}

func getBirthdaysToImport(ctx context.Context, tokenSource oauth2.TokenSource) ([][]string, error) {
	// Initialize People API client
	peopleService, err := people.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("unable to create People service: %v", err)
	}
	connections, err := peopleService.People.Connections.List("people/me").
		PersonFields("names,birthdays").
		PageSize(1000).
		Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve contacts: %v", err)
	}

	// birthdays is a map with a Person, and the Formatted date
	birthdays := [][]string{}
	for _, person := range connections.Connections {
		if len(person.Birthdays) <= 0 {
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

		birthdays = append(birthdays, []string{formatDate(birthday), name})
	}
	return birthdays, nil
}

func main() {
	accessToken := flag.String("token", "", "Access token for Google APIs")
	druRun := flag.Bool("dry-run", false, "Dry run mode (no changes made)")
	flag.Parse()

	ctx := context.Background()

	if *accessToken == "" {
		log.Fatal("Access token is required")
	}

	// Create an OAuth2 token source with the access token
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: *accessToken})

	birthdaysToImport, err := getBirthdaysToImport(ctx, tokenSource)
	if err != nil {
		log.Fatalf("Error fetching birthdays: %v", err)
	}

	fmt.Printf("Found %d birthdays to import.\n", len(birthdaysToImport))
	for i, event := range birthdaysToImport {
		fmt.Printf("\t %d: Date: %s, Name: %s\n", i+1, event[0], event[1])
	}

	if *druRun {
		log.Printf("Dry run, so we are done here.")
		return
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

	// Create an event for each birthday
	for _, bdate := range birthdaysToImport {
		// Create a new event for the birthday, in the newly created calendar. Make it a recurring and all day event.
		event := &calendar.Event{
			Summary:     bdate[1],
			Description: "Birthday",
			Start: &calendar.EventDateTime{
				Date:     bdate[0],
				TimeZone: "UTC",
			},
			End: &calendar.EventDateTime{
				Date:     bdate[0],
				TimeZone: "UTC",
			},
			Recurrence: []string{
				"RRULE:FREQ=YEARLY;COUNT=100",
			},
		}

		// Insert the event into the calendar
		_, err := calendarService.Events.Insert(cal.Id, event).Do()
		if err != nil {
			log.Printf("Unable to create event for %s: %v", bdate[1], err)
			continue
		}
		log.Printf("Created event for %s", bdate[1])
	}
}
