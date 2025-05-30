
# req2proto

This repo contains files relevant to the blog post [Decoding Google: Converting a Black Box to a White Box](https://brutecat.com/articles/decoding-google)

This project seeks to reverse-engineer Google internal protobuf definitions through error messages returned when sent protojson payloads.

> [!IMPORTANT]  
> This is still in an experimental state

```
$ go build; ./req2proto -H "Authorization: Bearer ya29...." -X POST -u https://people-pa.googleapis.com/v2/people -p google.internal.people.v2.InsertPersonRequest -o output
```

The `output` dir will then contain the request `.proto` files.


**TODO List**
- [ ] Add protojson response parsing support (in case the endpoint supports only protojson)
- [ ] Add automatic .proto import
- [ ] Support multithreading

**Example output:**

![req2proto output](./static/images/req2proto_output.png "Example req2proto output")
