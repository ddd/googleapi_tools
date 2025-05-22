
# req2proto

This repo contains files relevant to the blog post [Decoding Google: Converting a Black Box to a White Box](https://brutecat.com/articles/decoding-google)

This project seeks to reverse-engineer Google internal protobuf definitions through error messages returned when sent protojson payloads.

> [!IMPORTANT]  
> This is still in an experimental state

```
$ go build; ./req2proto -H "Authorization: Bearer ya29...." -X POST -u https://people-pa.googleapis.com/v2/people -p google.internal.people.v2.InsertPersonRequest -o output
```

<<<<<<< HEAD
The `output` dir will then contain the request `.proto` files.


**TODO List**
- [ ] Add protojson response parsing support (in case the endpoint supports only protojson)
- [ ] Add automatic .proto import
- [ ] Support multithreading

**Example output:**

![req2proto output](./static/images/req2proto_output.png "Example req2proto output")
=======
### [aas-scope-rs](./aas-scope-rs)
- A Rust tool for finding Android API clients that are approved for a target scope
- This is useful if you are looking to get authentication on an endpoint and you found the scopes required from `gapi-service`

### [discovery2json](https://github.com/yopgh/discovery2json)
- Python tool to dump the request JSON for an endpoint given the discovery document
- Credits: [@yopgh](https://github.com/yopgh)
>>>>>>> 06f1e8fd68e164fcb4bcfacfc9216aa8a1505dfe
