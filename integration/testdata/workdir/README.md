# Test project for BP_GO_WORKDIR

This is a test application to verify that the BP_GO_WORKDIR environment variable
works correctly.

The Go application main is located in the `main/` subdirectory, and the
buildpack should build from there when BP_GO_WORKDIR=main is set.
