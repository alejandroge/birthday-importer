# Birthday Importer

This script imports birthdays from Google Contacts into a dedicated Google Calendar. It creates a new calendar named "Birthdays from Contacts" and adds recurring birthday events for all contacts that have birthday information.

## Google Contacts Birthdays Importer
This is a simple project for extracting birthdays from your Google Contacts, and importing them into your Google
Calendar.

#### Features
- Extracts birthdays from Google Contacts
- Imports birthdays into Google Calendar
- Handles duplicates

#### Authentication
This is a bring Your Own Authentication Token kind of project :sweat_smile: I have no published app for this (and don't
plan to) but one can get an access token from Google via using their own tool: [oauth2l](https://github.com/google/oauth2l)

#### Prerequisites

- Go 1.21 or later
- Access token from oauth2l with the following required scopes:
  - `https://www.googleapis.com/auth/contacts.readonly`
  - `https://www.googleapis.com/auth/calendar`

## Installation

1. Clone this repository
2. Run `go mod tidy` to download dependencies

## Usage

```bash
# Get the access token using oauth2l (not included)
ACCESS_TOKEN=$(oauth2l fetch --credentials /path/to/credentials.json \
  https://www.googleapis.com/auth/contacts.readonly \
  https://www.googleapis.com/auth/calendar)

# Run the script with your access token
go run main.go -token "$ACCESS_TOKEN"
```

The script will:
- Create a new calendar named "Birthdays from Contacts" if it doesn't exist
- Import birthdays from your contacts as recurring yearly events
- Skip any contacts without birthday information
- Log the progress as it runs

## Error Handling

The script will:
- Exit if no access token is provided
- Exit if it fails to initialize the Google API clients
- Exit if it fails to create/access the calendar
- Skip individual contacts if their birthday event creation fails (with logging)
