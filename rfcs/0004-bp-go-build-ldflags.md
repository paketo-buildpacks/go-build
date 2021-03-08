# BP_GO_BUILD_LDFLAGS

## Proposal

Add an environment variable `BP_GO_BUILD_LDFLAGS` which would allow users to
set the value of `-ldflags` for their `go build` command.

## Motivation

The `-ldflags` flag can have complicated values with multiple sets of internal
quotation marks, an example of this complication can be seen in [this
issue](https://github.com/paketo-buildpacks/go-build/issues/129). The goal of
this environment variable is to remove one quotation level hopefully making it
easier to express the `-ldflags` value with little to no escaping.

## Implementation

If `BP_GO_BUILD_LDFLAGS` is set then `-ldflags` plus the value of the
environment variable will be added to the `go build` command.

## Unresolved Questions and Bikeshedding (Optional)

* Is this necessary? Should we just document the escaping better? Should we do
  both?

{{REMOVE THIS SECTION BEFORE RATIFICATION!}}
