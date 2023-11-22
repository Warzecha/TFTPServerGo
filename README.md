# In-memory TFTP Server

This is a simple in-memory TFTP server, implemented in Go.  It is
RFC1350-compliant, but doesn't implement the additions in later RFCs.  In
particular, options are not recognized.

# Usage

To start the server run:

```
go run cmd/tftp/main.go <port>
```

Example:

```
go run cmd/tftp/main.go 69
```

# Testing

To run tests, run:

```
go test -timeout 30s ncd/homework/tftp
```

# Currently unsupported

* Handling server side retransmission.