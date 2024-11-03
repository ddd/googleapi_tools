
## gapi-service v1

This tool can be used to find gRPC service names and required scopes given an endpoint.

## Usage

Set the `ANDROID_REFRESH_TOKEN` env variable to your Android refresh token

```
$ go build; ./gapi-service -e https://youtubei.googleapis.com/youtubei/v1/browse -x POST -c protojson

scopes: https://www.googleapis.com/auth/youtube ...
method: youtube.innertube.OPInnerTubeService.GetBrowse
service: youtubei.googleapis.com
```