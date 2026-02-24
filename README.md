# Birthday Importer

This project imports birthdays from Google Contacts into a dedicated Google Calendar.

This is needed for me since I live in Germany, and due to [regulatory issues](https://support.google.com/calendar/community-guide/302081881/birthdays-from-contacts-no-longer-showing-in-google-calendar?hl=en), Google Contacts does not show birthdays in the calendar app.

#### Features
- Extracts birthdays from Google Contacts
- Imports birthdays into Google Calendar

#### Authentication
Authentication is handled with [oauth2l](https://github.com/google/oauth2l) and a local oauth cache file.

`run.sh` expects a cached refresh token in `.oauth2l-cache.json` and uses it to get a fresh access token on every run.

#### Prerequisites

- Go 1.21 or later
- `oauth2l` installed
- OAuth client credentials in `.client_secret.json` (desktop app credentials)
- OAuth scopes:
  - `https://www.googleapis.com/auth/contacts.readonly`
  - `https://www.googleapis.com/auth/calendar`

## Installation

1. Clone this repository
2. Build the binary:

```bash
go build -o importer .
```

3. One-time OAuth cache setup:

Run this once to initialize `.oauth2l-cache.json` (interactive consent flow):

```bash
./setup.sh
```

## Usage

After the cache is initialized, run:

```bash
./run.sh
```

If `.oauth2l-cache.json` does not exist (or has no refresh token), `run.sh` exits with an error.

One can do a dry run with:

```bash
./run.sh --dry-run
```
