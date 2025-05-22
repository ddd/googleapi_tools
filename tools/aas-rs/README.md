
## aas-rs

This tool can be used to output valid Android API clients given a list of Android package IDs as well as SHA1 signatures

## Usage

Set the `ANDROID_REFRESH_TOKEN` env variable to your Android refresh token

```
$ cargo run --release
```

The output file can be found [../data/android_clients.json](../data/android_clients.json)