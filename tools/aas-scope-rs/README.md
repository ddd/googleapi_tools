
## aas-scope-rs

This tool can be used to find Android API clients that are approved for a target scope(s)

## Usage

Set the `ANDROID_REFRESH_TOKEN` env variable to your Android refresh token

```
$ cargo build --release
$ target/release/aas-scope-rs https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/peopleapi.readwrite

Found approved scope: https://www.googleapis.com/auth/peopleapi.readwrite for app=com.chrome.dev, sig=24bb24c05e47e0aefa68a58a766179d9b613a600
Found approved scope: https://www.googleapis.com/auth/peopleapi.readwrite for app=com.chrome.dev, sig=2698c0268c6eba8bb060ecaa27dae42098683b03
...
```

The output file `output/clients.json` should contain the list of clients which have the requested scopes enabled for it:

```json
{
  "com.google.vr.apps.ornament": [
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/peopleapi.readwrite"
  ],
  "com.google.android.as.oss": [
    "https://www.googleapis.com/auth/peopleapi.readwrite",
    "https://www.googleapis.com/auth/calendar"
  ],
  "com.google.android.inputmethod.latin": [
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/peopleapi.readwrite"
  ],
  "com.google.android.googlequicksearchbox": [
    "https://www.googleapis.com/auth/calendar",
    "https://www.googleapis.com/auth/peopleapi.readwrite"
  ],
  "com.google.android.apps.tachyon": [
    "https://www.googleapis.com/auth/calendar"
  ],
  ...
}
```