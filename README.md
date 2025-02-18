# HTTPGlobe
This tool sends HTTP requests to a list of URLs through BrightData proxies in different geographical locations. This helps discover web servers that restrict access based on geographic location. The tool only supports [Brightdata](https://brightdata.com) proxies and their geolocation targeting feature.

![CleanShot 2025-02-18 at 13 03 13](https://github.com/user-attachments/assets/3a2b078d-ce03-4661-8cf1-1b1ca45faf7e)


## Installation

```bash
go install github.com/bebiksior/httpglobe@latest
```

## Configuration
The default configuration is located in `$HOME/.config/httpglobe/config.json`.
It will be created on the first run.

### Default Configuration
```json
{
  "countries": ["cn", "in", "us", "jp", "de"],
  "proxy": {
    "host": "example.com",
    "port": "12345",
    "username": "username",
    "password": "password"
  }
}
```

The tool will automatically append the country code to the username in the format `-country-XX` for each request.

## Where to obtain Brightdata proxy credentials
1. Buy Brightdata proxies zone at https://brightdata.com 

![CleanShot 2025-02-18 at 13 20 36](https://github.com/user-attachments/assets/ed366604-dcbc-427a-ad4c-9da0f3787c37)

3. Go to https://brightdata.com/cp/zones and click on your proxies zone
4. Copy/paste proxy credentials to the config file

![CleanShot 2025-02-18 at 13 17 21](https://github.com/user-attachments/assets/c6005f63-1bfa-4db5-8e74-b500d1d6e835)


## Usage

Basic scan with default settings:

```bash
cat urls.txt | httpglobe
```

High-concurrency scan:

```bash
cat urls.txt | httpglobe -concurrency 20 -output results.json
```

## Output
The tool creates a single JSON file containing:

```json
{
  "results": [
    {
      "url": "example.com",
      "responses": [
        {
          "status_code": 200,
          "content_length": 800,
          "title": "Example Domain",
          "country": "us",
          "error": ""
        },
        {
          "status_code": 403,
          "title": "Access Denied",
          "content_length": 150,
          "country": "cn",
          "error": ""
        }
      ],
      "has_differences": true
    }
  ]
}
```
