# Dashcontrol


This program opens a chrome window full screen to a url.  The purpose of creating this was to open dashboard windows on remote machines.

### Dependencies

Requires xdotool and chrome browser

### Usage

```
./dashcontrol --nav=https://ebay.com

```

### Help
```
 -chrome string
    	Chrome Path
  -d duration
    	Wait Duration (default 1s)
  -nav string
    	nav (default "https://www.duckduckgo.com/")
  -port string
    	Chrome Port (default "9222")
  -refresh duration
    	Auto Refresh Duration (default 1m0s)
  -url string
    	devtools url (default "ws://127.0.0.1:9222")
  -v	verbose
```

### Api

#### New Url
```
curl 127.0.0.1/nav?url=https://new.url.com
```

#### Reload Existing

```
curl 127.0.0.1/refresh
```

#### Refresh original url

```
curl 127.0.0.1/reset
```

#### Scale Up Browser Window (chrome zoom)

```
curl 127.0.0.1/scaleup
```

```
curl 127.0.0.1/down
```

```
curl 127.0.0.1/scale/{int}
```

