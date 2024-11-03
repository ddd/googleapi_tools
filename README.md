
# googleapi_tools

This repo contains files relevant to the blog post [Decoding Google: Converting a Black Box to a White Box](https://brutecat.com/articles/decoding-google)

### [gapi-service](./gapi-service)
- Go tool to output required scopes as well as the gRPC service name of a Google API service
- Requires an Android refresh token

### [req2proto](./req2proto) (experimental)
- Go tool to find the request protobuf of any Google API endpoint by bruting all parameters via protojson error messages
> This tool is currently experimental

### [apkpure-go](./apkpure-go)
- Go tool to scrape apkpure.com for app signatures of Google APKs

### [aas-rs](./aas-rs)
- A Rust tool to output valid Android API clients given a list of Android package IDs as well as SHA1 signatures
- The output file can be found [data/android_clients.json](data/android_clients.json)

### aas-scope-rs (WIP)
- A Rust tool for finding Android API clients that are approved for a target scope
- This is useful if you are looking to get authentication on an endpoint and you found the scopes required from `gapi-service`