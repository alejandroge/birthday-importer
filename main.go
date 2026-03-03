package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

const (
	calendarName        = "Birthdays from Contacts"
	calendarDescription = "Calendar containing birthdays imported from Google Contacts"
	calendarMarker      = "[managed-by:birthday-importer;v=1]"
	eventManagedByKey   = "managedBy"
	eventManagedByValue = "birthday-importer"
)

type birthdayEntry struct {
	Date string
	Name string
}

func formatDate(birthday *people.Date) string {
	// If Year is 0, use a default year (e.g., 2000). Cannot leave it empty using the API.
	year := birthday.Year
	if year == 0 {
		year = 1970
	}
	return fmt.Sprintf("%04d-%02d-%02d", year, birthday.Month, birthday.Day)
}

func getBirthdaysToImport(ctx context.Context, tokenSource oauth2.TokenSource) ([]birthdayEntry, error) {
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

	birthdays := []birthdayEntry{}
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

		birthdays = append(birthdays, birthdayEntry{
			Date: formatDate(birthday),
			Name: name,
		})
	}
	return birthdays, nil
}

func managedCalendarDescription() string {
	return fmt.Sprintf("%s %s", calendarDescription, calendarMarker)
}

func findOrCreateManagedCalendar(calendarService *calendar.Service) (string, error) {
	log.Println("Looking for managed calendar.")
	legacyCalendarID := ""
	pageToken := ""

	for {
		calendarListCall := calendarService.CalendarList.List().PageToken(pageToken)
		calendars, err := calendarListCall.Do()
		if err != nil {
			return "", fmt.Errorf("unable to list calendars: %v", err)
		}

		for _, item := range calendars.Items {
			if item.Summary != calendarName {
				continue
			}

			if strings.Contains(item.Description, calendarMarker) {
				log.Printf("Found managed calendar: %s", item.Summary)
				return item.Id, nil
			}

			if item.Description == calendarDescription && legacyCalendarID == "" {
				legacyCalendarID = item.Id
			}
		}

		if calendars.NextPageToken == "" {
			break
		}

		pageToken = calendars.NextPageToken
	}

	if legacyCalendarID != "" {
		log.Println("Found legacy calendar without marker. Marking as managed.")
		_, err := calendarService.Calendars.Patch(legacyCalendarID, &calendar.Calendar{
			Description: managedCalendarDescription(),
		}).Do()
		if err != nil {
			return "", fmt.Errorf("unable to mark existing calendar as managed: %v", err)
		}
		log.Println("Marked legacy calendar as managed.")
		return legacyCalendarID, nil
	}

	log.Println("Managed calendar not found. Creating a new one.")
	cal, err := calendarService.Calendars.Insert(&calendar.Calendar{
		Summary:     calendarName,
		Description: managedCalendarDescription(),
		TimeZone:    "UTC",
	}).Do()
	if err != nil {
		return "", fmt.Errorf("unable to create calendar: %v", err)
	}

	log.Printf("Created calendar: %s", cal.Summary)
	return cal.Id, nil
}

func deleteManagedEvents(calendarService *calendar.Service, calendarID string) error {
	log.Println("Looking for previously managed events to delete.")
	type managedEvent struct {
		ID      string
		Summary string
	}

	eventsToDelete := []managedEvent{}
	pageToken := ""

	for {
		events, err := calendarService.Events.List(calendarID).
			PrivateExtendedProperty(fmt.Sprintf("%s=%s", eventManagedByKey, eventManagedByValue)).
			PageToken(pageToken).
			Do()
		if err != nil {
			return fmt.Errorf("unable to list managed events: %v", err)
		}

		for _, event := range events.Items {
			eventsToDelete = append(eventsToDelete, managedEvent{
				ID:      event.Id,
				Summary: event.Summary,
			})
		}

		if events.NextPageToken == "" {
			break
		}

		pageToken = events.NextPageToken
	}

	for _, event := range eventsToDelete {
		if err := calendarService.Events.Delete(calendarID, event.ID).Do(); err != nil {
			return fmt.Errorf("unable to delete managed event %s: %v", event.ID, err)
		}
		log.Printf("Deleted managed event: %s (%s)", event.Summary, event.ID)
	}

	log.Printf("Deleted %d managed events.", len(eventsToDelete))
	return nil
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
		fmt.Printf("\t %d: Date: %s, Name: %s\n", i+1, event.Date, event.Name)
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

	calendarID, err := findOrCreateManagedCalendar(calendarService)
	if err != nil {
		log.Fatalf("Unable to find or create managed calendar: %v", err)
	}

	if err := deleteManagedEvents(calendarService, calendarID); err != nil {
		log.Fatalf("Unable to delete previously managed events: %v", err)
	}

	// Create an event for each birthday
	for _, bdate := range birthdaysToImport {
		// Create a new event for the birthday as a recurring all-day event.
		event := &calendar.Event{
			Summary:     bdate.Name,
			Description: "Birthday",
			ExtendedProperties: &calendar.EventExtendedProperties{
				Private: map[string]string{
					eventManagedByKey: eventManagedByValue,
				},
			},
			Start: &calendar.EventDateTime{
				Date:     bdate.Date,
				TimeZone: "UTC",
			},
			End: &calendar.EventDateTime{
				Date:     bdate.Date,
				TimeZone: "UTC",
			},
			Recurrence: []string{
				"RRULE:FREQ=YEARLY;COUNT=100",
			},
		}

		// Insert the event into the calendar
		_, err := calendarService.Events.Insert(calendarID, event).Do()
		if err != nil {
			log.Printf("Unable to create event for %s: %v", bdate.Name, err)
			continue
		}
		log.Printf("Created event for %s", bdate.Name)
	}
}
