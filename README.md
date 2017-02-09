### usage

```
go get github.com/vektah/ior

cd $GOPATH/src/github.com/you/yourapp
ior -listen ":1234" -- [args passed through to your app]
```

![ior](http://i.imgur.com/yxN7Hty.jpg)

### What it does

1. Wait for a request.
2. Hashes all go files, excluding vendor, and other common large directories
3. If the hash changes, do a `go install`, show errors if it fails otherwise restart the server

Thats it. No inotify, no fsevents, no polling, no CPU chewing. 100% docker-machine friendly. 100% nfs friendly.

### How long does it take really?

On a project with ~30k lines across ~300 files it adds about 50ms to every request, this is just the time it takes to hash.

Because its using `go install` rather than `go build` it also leverages the pkg cache, so small changes are pretty quick to compile.

### Related projects
 - https://github.com/codegangsta/gin: This tool works in docker but uses a poll loop checking modified timestamps. This causes constant cpu load while running and nfs can cache the metadata.
 - https://github.com/pilu/fresh: Super configurable with nice output, but only uses fsevents/inotify so wont work over nfs boundaries.
 - https://github.com/tockins/realize: Full build system, lots of knobs, runs your tests, uses fsevents/inotify.
