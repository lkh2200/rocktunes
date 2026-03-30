#!/bin/bash

URL="$1"

if [ -z "$URL" ]; then
  echo "Usage: $0 <youtube_url>"
  exit 1
fi

BASE_DIR="/run/media/lachlanhenderson/IPOD/Videos/Youtube"

echo "Fetching metadata..."

CHANNEL=$(yt-dlp --print "%(uploader)s" "$URL" | head -n 1)
TITLE=$(yt-dlp --print "%(title)s" "$URL" | head -n 1)

sanitize() {
  echo "$1" \
    | tr '\n' ' ' \
    | sed 's#[^a-zA-Z0-9._-]#_#g' \
    | sed 's/__\+/_/g' \
    | sed 's/^_//;s/_$//'
}

SAFE_CHANNEL=$(sanitize "$CHANNEL")
SAFE_TITLE=$(sanitize "$TITLE")

OUT_DIR="${BASE_DIR}/${SAFE_CHANNEL}"
mkdir -p "$OUT_DIR"

TEMP_FILE="${OUT_DIR}/${SAFE_TITLE}_temp.%(ext)s"
FINAL_FILE="${OUT_DIR}/${SAFE_TITLE}.mpg"

echo "Downloading video..."

yt-dlp \
  -f "bestvideo[height<=1080]+bestaudio/best[height<=1080]" \
  --merge-output-format mp4 \
  -o "$TEMP_FILE" \
  "$URL"

# Find the downloaded file safely
INPUT=$(find "$OUT_DIR" -maxdepth 1 -type f -name "${SAFE_TITLE}_temp.*" | head -n 1)

if [ -z "$INPUT" ]; then
  echo "Download failed."
  exit 1
fi

echo "Converting with ffmpeg..."

ffmpeg -y -i "$INPUT" \
  -c:v mpeg2video \
  -c:a mp3 \
  -ac 2 \
  -ar 44100 \
  -vf "yadif,scale=320:240:force_original_aspect_ratio=decrease,pad=320:240:(ow-iw)/2:(oh-ih)/2" \
  -b:v 4096k \
  -b:a 320k \
  -f mpeg \
  "$FINAL_FILE"

if [ $? -ne 0 ]; then
  echo "FFmpeg failed. Keeping original file:"
  echo "$INPUT"
  exit 1
fi

echo "Cleaning up..."
rm "$INPUT"

echo "Done! Output: $FINAL_FILE"
