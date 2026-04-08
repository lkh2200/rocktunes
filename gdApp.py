import sys
import html
import json
import requests
from tqdm import tqdm
from mutagen.easyid3 import EasyID3
from mutagen.mp3 import MP3
from bs4 import BeautifulSoup
from pathlib import Path
from pathvalidate import sanitize_filename
import re

if len(sys.argv) != 2:
    print("Usage: python script.py <url>")
    sys.exit(1)

url = sys.argv[1]
response = requests.get(url)
data = response.text

soup = BeautifulSoup(data, 'html.parser')

# Get the <h1> heading
h1_tag = soup.find('h1')
if h1_tag:
    heading = h1_tag.get_text(strip=True)
    # Sanitize the heading to be safe for filenames
    safe_heading = re.sub(r'[<>:"/\\|?*]', '_', heading)
    dir_path = Path(f"~/Music/{safe_heading}").expanduser()
    dir_path.mkdir(exist_ok=True)
    print(f"H1 Heading: {heading}")
    print(f"Created directory: {dir_path.resolve()}\n")
else:
    print("No <h1> found.\n")
    heading = None

creator_span = soup.find("span", itemprop="creator")

if creator_span:
    creator = creator_span.get_text(strip=True)
    print(f"By: {creator}")
else:
    print("Creator not found.")

# Get the playlist attr
playlist_tag = soup.find(attrs={"playlist": True})
if playlist_tag:
    json_string = html.unescape(playlist_tag.get("playlist"))
    try:
        playlist = json.loads(json_string)
        counter = 0

        for item in playlist:

            counter += 1
            title = item.get("title", "<no title>")
            sources = item.get("sources", [])
            if sources:
                source = f"https://archive.org{sources[0].get('file')}"

            else:

                "<no source>"

            print(f"Title: {title}\nSource: {source}")

            # If the source URL is valid, download the file
            if source != "<no source>":
                # Customize the file extension if needed
                file_name = f"{counter}_{sanitize_filename(title)}.mp3"
                file_path = dir_path / file_name

                # Download the file
                try:
                    file_response = requests.get(source, stream=True)
                    if file_response.status_code == 200:
                        # Inside the download block:
                        file_size = int(
                            file_response.headers.get('content-length', 0))
                        chunk_size = 8192
                        progress = tqdm(total=file_size, unit='B',
                                        unit_scale=True, desc=title)

                        with open(file_path, 'wb') as f:
                            for chunk in file_response.iter_content(chunk_size=chunk_size):
                                if chunk:  # filter out keep-alive chunks
                                    f.write(chunk)
                                    progress.update(len(chunk))
                        progress.close()
                        print(f"Downloaded: {file_path}")
                        try:
                            audio = MP3(file_path, ID3=EasyID3)
                            audio["title"] = title
                            audio["artist"] = creator
                            audio["album"] = heading
                            audio["tracknumber"] = f"{counter}"
                            audio.save()
                            print(f"Tagged: {file_path}")
                        except Exception as e:
                            print(f"Failed to tag {file_path}: {e}")
                    else:
                        print(f"Failed to download {source}")
                except requests.RequestException as e:
                    print(f"Error downloading {source}: {e}")

            print()
    except json.JSONDecodeError as e:
        print("Error decoding JSON:", e)
else:
    print("Playlist input not found.")
