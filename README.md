# Rocktunes

Rocktunes is a go program to help you sync your podcasts, audiobooks and music
from your linux pc to an ipod with Rockbox installed. It also has tools for 
downloading videos from youtube and playlists from archive.org.

https://github.com/user-attachments/assets/628f222c-8623-4521-86df-ea52ffb02f3b

## Dependencies

- yt-dlp
- rsync
- golang
- python

## How to run

- Make your .env file
    - Add your paths to the example.env file
    - Change the name to .env

- Clone the repository and run:
```
go mod init
go mod tidy
go run .
```
